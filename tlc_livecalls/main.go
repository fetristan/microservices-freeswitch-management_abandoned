package main

import (
	"context"
	"encoding/json"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fetristan/tlc_dispatcher/message"
	logger "github.com/fetristan/tlc_logger"
	"github.com/fetristan/tlc_sessions/sessionsservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	log *logger.Logger
)

func init() {
	//To use all CPU
	runtime.GOMAXPROCS(runtime.NumCPU())
	log = logger.New("tlc_livecalls.log", true)
	log.Debug("tlc_livecalls initialization")
}

func main() {
	//Config reader
	config, err := readConf("config.yml")
	if err != nil {
		log.Error("%s", err)
	}
	log.Debugf("tlc_livecalls config ready : livecalls cycle : %d / livecalls pole : %s / livecalls url_api : %s / grcp_sessions host : %s / grcp_sessions port : %s / grcp_sessions timeout : %d / grcp_dispatcher host : %s / grcp_dispatcher port : %s / grcp_dispatcher timeout : %d", config.LiveCalls.Cycle, config.LiveCalls.Pole, config.LiveCalls.UrlApi, config.GrcpSessions.Host, config.GrcpSessions.Port, config.GrcpSessions.Timeout, config.GrcpDispatcher.Host, config.GrcpDispatcher.Port, config.GrcpDispatcher.Timeout)

	//Grcp sessions connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	sessionsServiceConn, err := grpc.Dial(config.GrcpSessions.Host+":"+config.GrcpSessions.Port, opts...)
	if err != nil {
		log.Errorf("fail to dial: %v", err)
	}
	defer sessionsServiceConn.Close()
	sessionsServiceClient := sessionsservice.NewSessionsServiceClient(sessionsServiceConn)
	log.Debugf("tlc_livecalls grcp_sessions connected : host: %s / port: %s/ grcp_sessions timeout : %d", config.GrcpSessions.Host, config.GrcpSessions.Port, config.GrcpSessions.Timeout)

	//Grcp dispatcher connection
	dispatcherConn, err := grpc.Dial(config.GrcpDispatcher.Host+":"+config.GrcpDispatcher.Port, opts...)
	if err != nil {
		log.Errorf("fail to dial: %v", err)
	}
	defer dispatcherConn.Close()
	dispatcherClient := message.NewMessageServiceClient(dispatcherConn)
	log.Debugf("tlc_livecalls grcp_dispatcher connected : host: %s / port: %s/ grcp_sessions timeout : %d", config.GrcpDispatcher.Host, config.GrcpDispatcher.Port, config.GrcpDispatcher.Timeout)

	//Infinite loop to take sessions, transform to livecalls and send it via dispatcher
	for {
		//Cycle to send live_calls
		time.Sleep(time.Duration(config.LiveCalls.Cycle) * time.Second)

		//Grcp sessions timeout context
		sessionsServiceCtx, sessionsServiceCancel := context.WithTimeout(context.Background(), time.Duration(config.GrcpSessions.Timeout)*time.Second)
		defer sessionsServiceCancel()

		//Get sessions via GRCP
		sessionCopy, err := sessionsServiceClient.GetSessionsCopyService(sessionsServiceCtx, &sessionsservice.Nil{})
		if err != nil {
			log.Errorf("%v.GetSessionsCopyService(_) = _, %v", sessionsServiceClient, err)
		}
		sessions := sessionsservice.SessionsCopyServiceToSessions(sessionCopy)

		//Build live_calls
		var livecalls []map[string]string
		livecalls = getLiveCalls(sessions, false, config.LiveCalls.Pole)
		var unmaskedLivecalls []map[string]string
		unmaskedLivecalls = getLiveCalls(sessions, true, config.LiveCalls.Pole)

		//Livecalls slice to json
		jsonStr, _ := json.Marshal(livecalls)
		log.Debugf("Livecalls : %s", jsonStr)
		jsonStrUnsmasked, _ := json.Marshal(unmaskedLivecalls)
		log.Debugf("Unmasked livecalls : %s", jsonStrUnsmasked)

		//Grcp dispatcher timeout context
		dispatcherCtx, dispatcherCancel := context.WithTimeout(context.Background(), time.Duration(config.GrcpDispatcher.Timeout)*time.Second)
		defer dispatcherCancel()
		dispatcherCtxUnsmasked, dispatcherCancelUnsmasked := context.WithTimeout(context.Background(), time.Duration(config.GrcpDispatcher.Timeout)*time.Second)
		defer dispatcherCancelUnsmasked()

		//Send to dispatcher
		sendLiveCallsToDispatcher(string(jsonStr), dispatcherClient, dispatcherCtx, config.LiveCalls.UrlApi, config.LiveCalls.Pole)
		sendLiveCallsToDispatcher(string(jsonStrUnsmasked), dispatcherClient, dispatcherCtxUnsmasked, config.LiveCalls.UrlApiUnmasked, config.LiveCalls.Pole)
	}
}

// To set default value to UNKNOWN
func returnValueOrUnknown(value string) string {
	if value != "" {
		return value
	} else {
		return "UNKNOWN"
	}
}

// To set the good caller/callee type
func returnCallerCalleeType(typeToTransform string) string {
	switch typeToTransform {
	case "0":
		return "EXTERNAL"
	case "1":
		return "XXXXXX"
	case "2":
		return "YYYYYY"
	case "3":
		return "EXTENSION"
	case "4":
		return "IVR"
	default:
		return "IVR"
	}
}

func anonymiseNumber(value string, unmasked bool) string {
	if !unmasked {
		if _, err := strconv.Atoi(value); err == nil && len(value) > 9 {
			var tmpBeginString string = value[:len(value)-6]
			var tmpEndingString string = value[len(value)-4:]
			return tmpBeginString + "**" + tmpEndingString
		}
	}
	return value
}

func ifNilDontCreateEntry(call *map[string]string, key string, value string) {
	if value != "" {
		tmpCall := *call
		tmpCall[key] = value
		*call = tmpCall
	}
}

func ifNilDontCreateDateEntry(call *map[string]string, key string, value time.Time) {
	if !value.IsZero() {
		tmpCall := *call
		tmpCall[key] = strconv.FormatInt(value.Unix(), 10)
		*call = tmpCall
	}
}

// To get the livecall slice via sessions data
func getLiveCalls(sessions []sessionsservice.Session, unmasked bool, pole string) []map[string]string {
	var livecalls []map[string]string
	for _, session := range sessions {
		if session.Pole == pole {
			var call map[string]string
			call = make(map[string]string)
			ifNilDontCreateEntry(&call, "call_direction", session.CallDirection)
			if session.CallDirection == "outgoing" {
				ifNilDontCreateEntry(&call, "b_original_callee", anonymiseNumber(session.OriginalCalleeNum, unmasked))
				if session.CalleeNickname != "" && session.IvrState != "ATTENTE" {
					call["b_callee_name"] = returnValueOrUnknown(session.CalleeNickname)
				} else {
					call["b_callee_name"] = returnValueOrUnknown(strings.Replace(session.OtherLegCalleeIdName, session.CalleeNum, anonymiseNumber(session.CalleeNum, unmasked), 1))
				}
				ifNilDontCreateEntry(&call, "a_original_caller", session.OriginalCallerNum)
				if session.ServiceId == "" {
					ifNilDontCreateEntry(&call, "a_caller_num", session.CallerNum)
					ifNilDontCreateEntry(&call, "b_callee_num", anonymiseNumber(session.CalleeNum, unmasked))
					call["a_caller_name"] = returnValueOrUnknown(session.EffectiveCallerIdName)
				} else {
					ifNilDontCreateEntry(&call, "b_callee_num", session.CalleeNum)
					ifNilDontCreateEntry(&call, "a_caller_num", anonymiseNumber(session.CallerNum, unmasked))
					call["a_caller_name"] = returnValueOrUnknown(strings.Replace(session.EffectiveCallerIdName, session.CallerNum, anonymiseNumber(session.CallerNum, unmasked), 1))
				}
			} else {
				ifNilDontCreateEntry(&call, "a_original_caller", anonymiseNumber(session.OriginalCallerNum, unmasked))
				ifNilDontCreateEntry(&call, "a_caller_num", anonymiseNumber(session.CallerNum, unmasked))
				call["a_caller_name"] = returnValueOrUnknown(strings.Replace(session.EffectiveCallerIdName, session.CallerNum, anonymiseNumber(session.CallerNum, unmasked), 1))
				ifNilDontCreateEntry(&call, "b_original_callee", session.OriginalCalleeNum)
				ifNilDontCreateEntry(&call, "b_callee_num", session.CalleeNum)
				if session.CalleeNickname != "" && session.IvrState != "ATTENTE" {
					call["b_callee_name"] = returnValueOrUnknown(session.CalleeNickname)
				} else {
					call["b_callee_name"] = returnValueOrUnknown(session.EffectiveCalleeIdName)
				}
			}
			call["a_type"] = returnCallerCalleeType(session.CallerType)
			ifNilDontCreateDateEntry(&call, "a_create_timestamp", session.DateStart)
			ifNilDontCreateDateEntry(&call, "b_create_timestamp", session.DateStart)
			ifNilDontCreateDateEntry(&call, "a_answer_timestamp", session.DateCon)
			ifNilDontCreateDateEntry(&call, "b_answer_timestamp", session.DateCon)
			call["b_type"] = returnCallerCalleeType(session.CalleeType)
			if session.CallState == "" {
				call["call_state"] = "ACTIVE"
			} else {
				call["call_state"] = session.CallState
			}
			ifNilDontCreateEntry(&call, "a_uuid", session.CallerUid)
			ifNilDontCreateEntry(&call, "b_uuid", session.CalleeUid)
			livecalls = append(livecalls, call)
		}
	}
	if livecalls == nil {
		var call map[string]string
		call = make(map[string]string)
		livecalls = append(livecalls, call)
	}
	return livecalls
}

func sendLiveCallsToDispatcher(livecalls string, dispatcherClient message.MessageServiceClient, ctx context.Context, urlApi string, pole string) bool {
	_, err := dispatcherClient.New(ctx, &message.MessageRequest{
		Method:   "POST",
		Request:  urlApi + pole,
		Priority: 1,
		Timeout:  1,
		Data:     livecalls,
	})

	if err != nil {
		log.Error("%s", err)
		return false
	}
	return true
}
