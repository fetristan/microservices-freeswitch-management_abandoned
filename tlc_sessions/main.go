package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/cgrates/fsock"
	events "github.com/fetristan/tlc_events"
	logger "github.com/fetristan/tlc_logger"
	"github.com/fetristan/tlc_sessions/sessionsservice"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	log *logger.Logger
	fs  []*fsock.FSock
	db  *Db
)

// fibDuration returns successive Fibonacci numbers converted to time.Duration.
func fibDuration(durationUnit, maxDuration time.Duration) func() time.Duration {
	a, b := 0, 1
	return func() time.Duration {
		a, b = b, a+b
		fibNrAsDuration := time.Duration(a) * durationUnit
		if maxDuration > 0 && maxDuration < fibNrAsDuration {
			return maxDuration
		}
		return fibNrAsDuration
	}
}

type server struct {
	sessionsservice.UnimplementedSessionsServiceServer
}

func init() {
	//To use all CPU
	runtime.GOMAXPROCS(runtime.NumCPU())
	log = logger.New("tlc_sessions.log", true)
	log.Debug("tlc_session initialization")
}

func main() {
	//Config reader
	config, err := readConf("config.yml")
	if err != nil {
		log.Error("%s", err)
	}
	log.Debugf("tlc_session config ready : database host: %s  / database port: %s / database user: %s / database pass: %s / database dbname: %s / grcp_listener port : %d", config.Database.Host, config.Database.Port, config.Database.User, config.Database.Pass, config.Database.Dbname, config.GrcpListener.Port)
	for _, freeswitchConf := range config.Freeswitch {
		log.Debugf("tlc_session config ready : freeswitch host: %s  / freeswitch port: %s / freeswitch pass: %s / freeswitch pole : %s / freeswitch retry number : %d", freeswitchConf.Host, freeswitchConf.Port, freeswitchConf.Pass, freeswitchConf.Pole, freeswitchConf.RetryNumber)
	}
	//Grcp listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.GrcpListener.Port))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	sessionsservice.RegisterSessionsServiceServer(grpcServer, &server{})
	go grpcServer.Serve(lis)
	log.Debugf("tlc_session grcp connected : port: %d", config.GrcpListener.Port)

	//Database connection
	db = newDb(config)
	log.Debugf("tlc_sessions database connected : database host: %s  / database port: %s / database user: %s / database pass: %s / database dbname: %s", config.Database.Host, config.Database.Port, config.Database.User, config.Database.Pass, config.Database.Dbname)

	//Redis connection
	connectToRedisDatabase(config.Redis.Host, config.Redis.Port, config.Redis.Pass, config.Redis.Dbname)
	log.Debugf("tlc_sessions redis connected : host: %s  / port: %s / pass: %s / db : %d", config.Redis.Host, config.Redis.Port, config.Redis.Pass, config.Redis.Dbname)

	//Freeswitch event listener routing
	evFilters := make(map[string][]string)
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_CREATE")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_PROGRESS")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_ANSWER")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_BRIDGE")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_UNBRIDGE")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_DESTROY")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_HOLD")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_UNHOLD")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_PARK")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CHANNEL_UNPARK")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "RECORD_START")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "PLAYBACK_START")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "API")
	evFilters["Event-Name"] = append(evFilters["Event-Name"], "CUSTOM")
	//evFilters["Event-Name"] = append(evFilters["Event-Name"], "monitor::ivr_state")
	evHandlers := map[string][]func(string, int){
		"CHANNEL_CREATE":   {channelCreate},
		"CHANNEL_PROGRESS": {channelProgress},
		"CHANNEL_BRIDGE":   {channelBridge},
		"CHANNEL_UNBRIDGE": {channelUnbridge},
		"CHANNEL_DESTROY":  {channelDestroy},
		"CHANNEL_HOLD":     {channelHold},
		"CHANNEL_UNHOLD":   {channelUnhold},
		"CHANNEL_PARK":     {channelPark},
		"CHANNEL_UNPARK":   {channelUnpark},
		"RECORD_START":     {recordStart},
		"PLAYBACK_START":   {playbackStart},
		"API":              {apiCommand},
		//"CUSTOM monitor::ivr_state": {customivrState},
	}

	//Get sessions into redis before connect to freeswitch
	sessions, errBool := getRedisDatabaseSessions()
	log.Debugf("tlc_sessions sessions found in redis after restart : %s", sessions)

	for _, freeswitchConf := range config.Freeswitch {
		fstmp, err := fsock.NewFSock(freeswitchConf.Host+":"+freeswitchConf.Port, freeswitchConf.Pass, freeswitchConf.RetryNumber, 0, fibDuration, evHandlers, evFilters, nil, 0, true)
		fs = append(fs, fstmp)
		defer fs[len(fs)-1].Disconnect()
		if err != nil {
			log.Errorf("FreeSWITCH error: %s", err)
			fs[len(fs)-1] = nil
			break
		}
		log.Debugf("tlc_sessions freeswitch connected : host: %s  / port: %s / pass: %s / pole : %s / freeswitch retry number : %d", freeswitchConf.Host, freeswitchConf.Port, freeswitchConf.Pass, freeswitchConf.Pole, freeswitchConf.RetryNumber)
		//Set sessions from redis into sessions memory if they exist in one freeswitch
		if !errBool {
			for _, session := range sessions {
				result, err := fs[len(fs)-1].SendApiCmd("show calls as json")
				if err != nil {
					panic(err)
				}
				liveCalls := &Livecalls{}
				err = json.Unmarshal([]byte(result), liveCalls)
				if err != nil {
					panic(err)
				}
				for _, liveCall := range liveCalls.Rows {
					if (session.CallerUid == liveCall.Uuid && session.CalleeUid == liveCall.BUuid) || (session.CallerUid == liveCall.BUuid && session.CalleeUid == liveCall.Uuid) {
						setSessions(append(*sessionsservice.GetSessions(), session))
					}
				}
			}
			log.Debugf("tlc_sessions session found in freeswitch after restart : %s", sessionsservice.GetSessions())
		}
		go fs[len(fs)-1].ReadEvents()
	}

	//Infinite loop to debug with log
	for {
		time.Sleep(time.Duration(config.Sessions.Cycle) * time.Second)
	}
}

// Used via GRCP to dump one session found by caller uuid and callee uuid
func (s *server) GetSessionCopyService(ctx context.Context, in *sessionsservice.CallerCalleeUid) (*sessionsservice.SessionCopy, error) {
	var session *sessionsservice.Session
	var found bool
	log.Debugf("Received:GetSessionCopyService : %v / %v", in.GetCallerUid(), in.GetCalleeUid())
	session, _, found = sessionsservice.GetSession(in.GetCallerUid(), in.GetCalleeUid(), in.GetExactly(), in.GetOnlyOneUid())
	if found {
		log.Debugf("Received:GetSessionCopyService : Found and send")
		return sessionsservice.SessionToSessionsService(session), nil
	} else {
		log.Debugf("Received:GetSessionCopyService : Not found")
		return &sessionsservice.SessionCopy{}, nil
	}
}

// Set variable to session from GRPC
func (s *server) SetVar(ctx context.Context, in *sessionsservice.Var) (*wrapperspb.BoolValue, error) {
	SetVar(in.GetCallerUid(), in.GetCalleeUid(), in.GetNeededKey(), in.GetNeededValue())
	return &wrapperspb.BoolValue{Value: true}, nil
}

// Set variable to session
func SetVar(callerUuid string, calleeUid string, key string, value string) bool {
	for _, oneFs := range fs {
		oneFs.SendBgapiCmd("uuid_setvar " + callerUuid + " " + key + " " + value)
		oneFs.SendBgapiCmd("uuid_setvar " + calleeUid + " " + key + " " + value)
	}
	return true
}

// Set multiple variable from GRPC
func (s *server) SetVarMultiple(ctx context.Context, in *sessionsservice.VarMultiple) (*wrapperspb.BoolValue, error) {
	SetVarMultiple(in.GetCallerUid(), in.GetCalleeUid(), in.GetNeededKeyValue())
	return &wrapperspb.BoolValue{Value: true}, nil
}

// Set multiple variable to session
func SetVarMultiple(callerUuid string, calleeUuid string, neededKeyValue map[string]string) bool {
	var keyValueCommand string = ""
	for neededKey, neededValue := range neededKeyValue {
		if keyValueCommand == "" {
			keyValueCommand = neededKey + "=" + neededValue
		} else {
			keyValueCommand = keyValueCommand + ";" + neededKey + "=" + neededValue
		}
	}
	for _, oneFs := range fs {
		oneFs.SendBgapiCmd("uuid_setvar_multi " + callerUuid + " " + keyValueCommand)
		oneFs.SendBgapiCmd("uuid_setvar_multi " + calleeUuid + " " + keyValueCommand)
	}
	return true
}

// Used via GRCP to dump all session
func (s *server) GetSessionsCopyService(ctx context.Context, empty *sessionsservice.Nil) (*sessionsservice.SessionsCopy, error) {
	log.Debugf("Received:GetSessionsCopy")
	return sessionsservice.GetSessionsCopyService(*sessionsservice.GetSessions()), nil
}

func setSessions(sessions []sessionsservice.Session) {
	sessionsservice.SetSessions(sessions)
	setRedisDatabaseSessions(sessions)
}

func removeSessions(uniqueId string, otherId string) {
	sessionsservice.RemoveSession(uniqueId, otherId)
	setRedisDatabaseSessions(*sessionsservice.GetSessions())
}

// Called when a channel is created on freeswitch
func channelCreate(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	var session *sessionsservice.Session
	var sessionId int
	var foundSession bool
	/*var isRobot bool = false
	if event.IsRobot == "1" {
		isRobot = true
		session, sessionId, foundSession = sessionsservice.GetSession(event.UniqueId, event.AcdUuid, false)
		if !foundSession {
			isRobot = false
			session, sessionId, foundSession = sessionsservice.GetSession(event.UniqueId, event.OtherId, false)
		}
	} else {*/
	session, sessionId, foundSession = sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	//}
	if !foundSession {
		/*if event.IsC2C != "" && len(event.CallerNumber) > 4 && len(event.CalleeNumber) > 4 && event.OriginalCaller2 == event.CallerNumber && event.OriginalCallee == event.CalleeNumber {
			logSession(event, session, "SESSION BLOCKED BECAUSE C2C OUTGOING CLONE")
		} else if event.IsC2C == "" && event.ServiceId == "" && len(event.CallerNumber) > 4 && len(event.CalleeNumber) > 4 && event.OriginalCaller2 == event.CallerNumber && event.OriginalCallee == event.CalleeNumber {
			logSession(event, session, "SESSION BLOCKED BECAUSE OUTGOING CLONE")
			} else if event.IsRobotCustomer != "" {
			logSession(event, session, "SESSION BLOCKED BECAUSE ROBOT OUTGOING CLONE")
		} else {*/
		logSession(event, session, "SESSION CREATE")
		var newSession sessionsservice.Session
		session = &newSession
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallerNum = event.CallerNumber
		session.CalleeNum = event.CalleeNumber
		session.CallerUid = event.UniqueId
		session.CalleeUid = event.OtherId
		session.CallState = event.CallState
		session.Pole = event.Pole
		session.DateStart = event.CreateTime
		session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
		setSessions(append(*sessionsservice.GetSessions(), *session))
		log.Debugf("AFTER : %+v", session)
		//}
	} else {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		fixSessionUids(event, session)
		session.Pole = event.Pole
		/*if isRobot {
			if event.UniqueId != "" {
				session.CallerUid = event.UniqueId
			}
			if event.AcdUuid != "" {
				session.CalleeUid = event.AcdUuid
			}
		}*/
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
	}
}

// Called when a channel is in ringing on freeswitch
func channelProgress(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		/*if event.IsC2C != "" && event.OtherType == "" {
			logSession(event, session, "SESSION BLOCKED BECAUSE C2C CALLER PROGRESS")
			log.Debugf("BEFORE SESSION : %+v", session)
			if event.IsC2C == "1" {
				session.CallerNum = event.CalleeNumber
				session.CalleeNum = event.CallerNumber
			} else {
				session.CallerNum = event.CallerNumber
				session.CalleeNum = event.CalleeNumber
			}
			session.CallState = "RINGING"
			session.Pole = event.Pole
			session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
			sessions := *sessionsservice.GetSessions()
			sessions[sessionId] = *session
			setSessions(sessions)
			log.Debugf("AFTER : %+v", session)
		} else {*/
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		session.DateRing = event.ProgressTime
		if len(event.CallerNumber) == 4 {
			session.CallerNum = event.CallerNumber
		}
		if len(event.CalleeNumber) == 4 {
			session.CalleeNum = event.CalleeNumber
		}
		session.CallState = "RINGING"
		session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
		session.Pole = event.Pole
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
		//}
	} else {
		logSession(event, session, "SESSION NOT FOUND")
	}
}

//Called when a channel is answered on freeswitch
/*func channelAnswer(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event Event = createEvent(eventStr)
	session, sessionId, foundSession := sessionsservice.GetSession(event.uniqueId, event.otherId)
	if foundSession {
		if event.isC2C != "" && event.otherType == "" {
			logSession(event, session, "SESSION BLOCKED BECAUSE C2C CALLER PROGRESS")
		} else if event.otherType == "" {
			logSession(event, session, "SESSION BLOCKED BECAUSE IVR ANSWER")
		} else {
			logSession(event, session, "SESSION FOUND")
			setCustomsVariablesNeededFromEvent(event, session)
			session.CallerNum = event.callerNumber
			session.CalleeNum = event.calleeNumber
			session.CallState = event.callState
			session.Pole = event.pole
			session.DateCon = formatDateTimeFromMicrosecondsLogFormat(event.answeredTime)
			if session.DateRing == "" {
				session.DateRing = session.DateCon
			}
			sessions := *sessionsservice.GetSessions()
			sessions[sessionId] = *session
			setSessions(sessions)
		}
	} else {
		logSession(event, session, "SESSION NOT FOUND")
	}
}*/

// Called when a channel is bridged on freeswitch
func channelBridge(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		/*if len(event.CallerNumber) == 4 {
			session.CallerNum = event.CallerNumber
		}
		if event.CalleeNumber != "" && len(event.CalleeNumber) < 5 {
			session.CalleeNum = event.CalleeNumber
		} else if event.EffectiveCalleeIdNumber != "" {
			session.CalleeNum = event.EffectiveCalleeIdNumber
		}*/
		session.CallState = event.CallState
		session.Pole = event.Pole
		session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
		fixSessionUids(event, session)
		//if session.DateCon == "" {
		session.DateCon = event.EventDate
		//}
		//updateSessionFromDatabase(session, event)
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
	} else {
		logSession(event, session, "SESSION CREATE")
		var newSession sessionsservice.Session
		session = &newSession
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallerNum = event.CallerNumber
		if event.CalleeNumber != "" && len(event.CalleeNumber) < 5 {
			session.CalleeNum = event.CalleeNumber
		} else if event.EffectiveCalleeIdNumber != "" {
			session.CalleeNum = event.EffectiveCalleeIdNumber
		}
		session.CallerUid = event.UniqueId
		session.CalleeUid = event.OtherId
		session.CallState = event.CallState
		session.Pole = event.Pole
		session.DateStart = event.EventDate
		session.DateRing = event.EventDate
		session.DateCon = event.EventDate
		//updateSessionFromDatabase(session, event)
		setSessions(append(*sessionsservice.GetSessions(), *session))
		log.Debugf("AFTER : %+v", session)
	}
}

// Called when a channel is unbridged on freeswitch
func channelUnbridge(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, _, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		removeSessions(event.UniqueId, event.OtherId)
	} else {
		logSession(event, session, "SESSION NOT FOUND")
	}
}

// Called when a channel is destroyed on freeswitch
func channelDestroy(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	var session *sessionsservice.Session
	var foundSession bool
	/*if event.IsRobot == "1" {
		session, _, foundSession = sessionsservice.GetSession(event.UniqueId, event.OtherId, false)
	} else {*/
	session, _, foundSession = sessionsservice.GetSession(event.UniqueId, event.OtherId, true, false)
	//}
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		removeSessions(event.UniqueId, event.OtherId)
	} else {
		logSession(event, session, "SESSION NOT FOUND")
		session, _, foundSession = sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
		if foundSession {
			logSession(event, session, "SESSION FOUND (NOT EXACTLY)")
			removeSessions(event.UniqueId, event.OtherId)
		}
	}
}

// Called when a channel is parked (virtual agents) on freeswitch
func channelPark(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, _, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
	} else {
		logSession(event, session, "SESSION CREATE")
		log.Debugf("BEFORE SESSION : %+v", session)
		var newSession sessionsservice.Session
		session = &newSession
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallerNum = event.CalleeNumber
		session.CalleeNum = event.CalleeNumber
		session.CallerUid = event.UniqueId
		session.CalleeUid = event.OtherId
		session.CallState = event.CallState
		session.DateStart = event.EventDate
		session.DateRing = event.EventDate
		session.DateCon = event.EventDate
		session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
		setSessions(append(*sessionsservice.GetSessions(), *session))
		log.Debugf("AFTER : %+v", session)
	}
}

// Called when a channel is unpark (virtual agents) on freeswitch
func channelUnpark(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, _, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, true, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		removeSessions(event.UniqueId, event.OtherId)
	} else {
		logSession(event, session, "SESSION NOT FOUND")
	}
}

// Called when a sound is played by IVR on the call
func playbackStart(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, true, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		session.DateRing = event.ProgressTime
		if session.DateCon.IsZero() {
			session.DateCon = session.DateStart
		}
		//session.CallerNum = event.CallerNumber
		//session.CalleeNum = event.EffectiveCalleeIdNumber
		session.Pole = event.Pole
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
	} else {
		logSession(event, session, "SESSION CREATE")
		var newSession sessionsservice.Session
		session = &newSession
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallerNum = event.CallerNumber
		session.CalleeNum = event.EffectiveCalleeIdNumber
		session.Pole = event.Pole
		session.CallerUid = event.UniqueId
		session.CalleeUid = event.OtherId
		session.DateStart = event.CreateTime
		session.DateRing = event.CreateTime
		session.DateCon = event.CreateTime
		setSessions(append(*sessionsservice.GetSessions(), *session))
		log.Debugf("AFTER : %+v", session)
	}
}

// Called when a call record start on freeswitch
func recordStart(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		session.RecordId = strings.Split(strings.Split(event.RecordId, "/RECORDING/")[1], ".oga")[0]
		session.RecordingName = event.RecordId
		session.IsRecorded = "1"
		fixSessionUids(event, session)
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
		session_clone, _, foundSessionClone := sessionsservice.GetSession(event.OtherId, "", false, false)
		if foundSessionClone {
			if (session_clone.CallerUid + session_clone.CalleeUid) != (session.CallerUid + session.CalleeUid) {
				logSession(event, session_clone, "SESSION CLONE FOUND AND DESTROYED")
				removeSessions(session_clone.CallerUid, session_clone.CalleeUid)
			}
		}
	} else {
		logSession(event, session, "SESSION NOT FOUND")
	}
}

// Called when a API command is executed on freeswitch
func apiCommand(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	if event.ApiCommand == "uuid_setvar" || event.ApiCommand == "uuid_setvar_multi" {
		log.Debugf("BEFORE EVENT : %+v", event)
		log.Debugf("BEFORE EVENTSTR : %s", strings.Replace(eventStr, "\n", " / ", -1))
		//log.Debugf("DEBUG LO : %+v", event)
		//log.Debugf("before : %+v", sessionsservice.GetSessions())
		var uuid = strings.Split(event.ApiCommandArgument, " ")[0]
		commandArgsWithUuid := strings.Split(event.ApiCommandArgument, ";")
		commandArgsWithUuid[0] = strings.ReplaceAll(commandArgsWithUuid[0], uuid+" ", "")
		var commandArgsWithoutUuid = make(map[string]string)
		for _, arg := range commandArgsWithUuid {
			argString := string(arg)
			var commandSplitted []string
			if strings.Contains(argString, "=") {
				commandSplitted = strings.Split(argString, "=")
			} else {
				commandSplitted = strings.Split(argString, " ")
			}
			if len(commandSplitted) > 1 {
				commandArgsWithoutUuid[commandSplitted[0]] = commandSplitted[1]
			} else {
				commandArgsWithoutUuid[commandSplitted[0]] = ""
			}
		}
		session, sessionId, foundSession := sessionsservice.GetSession(uuid, "", false, true)
		if foundSession {
			logSession(event, session, "SESSION FOUND")
			log.Debugf("BEFORE SESSION : %+v", session)
			//log.Debugf("before : %+v", session)
			session = mergeEventMapIntoSession(commandArgsWithoutUuid, session)
			//log.Debugf("after : %+v", sessionsservice.GetSessions())
			sessions := *sessionsservice.GetSessions()
			sessions[sessionId] = *session
			setSessions(sessions)
			log.Debugf("AFTER : %+v", session)
			//log.Debugf("after : %+v", sessionsservice.GetSessions())
		}
	}
}

//Called when step of a ivr change
/*func customivrState(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event Event = createEvent(eventStr)
	session, _, _ := sessionsservice.GetSession(event.uniqueId, event.otherId)
	logSession(event, session, "LOGGER ")
	log.Debugf("TEST : %s", eventStr)
}*/

func channelHold(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallState = event.CallState
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
	}
}

func channelUnhold(eventStr string, connIdx int) {
	sessionsservice.LockSessions()
	defer sessionsservice.UnlockSessions()
	var event events.Event = events.CreateEvent(eventStr)
	log.Debugf("BEFORE EVENT : %+v", event)
	session, sessionId, foundSession := sessionsservice.GetSession(event.UniqueId, event.OtherId, false, false)
	if foundSession {
		logSession(event, session, "SESSION FOUND")
		log.Debugf("BEFORE SESSION : %+v", session)
		setCustomsVariablesNeededFromEvent(event, session)
		session.CallState = "ACTIVE"
		sessions := *sessionsservice.GetSessions()
		sessions[sessionId] = *session
		setSessions(sessions)
		log.Debugf("AFTER : %+v", session)
	}
}
