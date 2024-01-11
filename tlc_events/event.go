package events

import (
	"strconv"
	"strings"
	"time"

	"github.com/cgrates/fsock"
)

type Event struct {
	EventName               string
	EventSubclass           string
	UniqueId                string
	OtherId                 string
	OriginalCaller          string
	CallerName              string
	DestName                string
	HangupCause             string
	CallState               string
	FsDirection             string
	OtherType               string
	CreateTime              time.Time
	AnsweredTime            time.Time
	ProgressTime            time.Time
	HangupTime              time.Time
	TransfertTime           time.Time
	BridgedTime             time.Time
	LastHoldTime            time.Time
	AccumHold               string
	EndpointDispo           string
	BridgeDest              string
	BridgeSignalBond        string
	LastBridgeTo            string
	LastBridgehangup        string
	OtherLegDestNumber      string
	Dtmf                    string
	DtmfDuration            string
	SipHangupDisposition    string
	EventDate               time.Time
	Who                     string
	ApiCommand              string
	ApiCommandArgument      string
	OriginationCallerIdName string
	OriginationCalleeIdName string
	EffectiveCallerIdName   string
	EffectiveCalleeIdName   string
	SipCalleeIdName         string
	EffectiveCalleeIdNumber string
	StartTime               time.Time
	OtherLegCalleeIdName    string
	CallerNumber            string
}

func CreateEvent(eventStr string) Event {
	eventMap := fsock.FSEventStrToMap(eventStr, []string{})
	return CreateEventFromMap(eventMap)
}

func CreateEventFromMap(eventMap map[string]string) Event {
	var event Event
	event.EventName = eventMap["Event-Name"]
	event.EventSubclass = eventMap["Event-Subclass"]
	event.UniqueId = eventMap["Unique-ID"]
	if eventMap["variable_signal_bond"] != "" {
		event.OtherId = eventMap["variable_signal_bond"]
	} else {
		event.OtherId = eventMap["Other-Leg-Unique-ID"]
	}
	event.OriginalCaller = eventMap["Caller-Orig-Caller-ID-Number"]
	event.CallerName = eventMap["Caller-Caller-ID-Name"]
	event.DestName = eventMap["Caller-Callee-ID-Name"]
	event.HangupCause = eventMap["Hangup-Cause"]
	event.CallState = eventMap["Channel-Call-State"]
	event.FsDirection = eventMap["Call-Direction"]
	event.OtherType = eventMap["Other-Type"]
	event.CreateTime = UnixMicroStrToTime(eventMap["Caller-Channel-Created-Time"])
	event.AnsweredTime = UnixMicroStrToTime(eventMap["Caller-Channel-Answered-Time"])
	event.ProgressTime = UnixMicroStrToTime(eventMap["Caller-Channel-Progress-Time"])
	event.HangupTime = UnixMicroStrToTime(eventMap["Caller-Channel-Hangup-Time"])
	event.TransfertTime = UnixMicroStrToTime(eventMap["Caller-Channel-Transfer-Time"])
	event.BridgedTime = UnixMicroStrToTime(eventMap["Caller-Channel-Bridged-Time"])
	event.LastHoldTime = UnixMicroStrToTime(eventMap["Caller-Channel-Last-Hold"])
	event.AccumHold = eventMap["Caller-Channel-Hold-Accum"]
	event.EndpointDispo = eventMap["variable_endpoint_disposition"]
	event.BridgeDest = eventMap["Caller-Callee-ID-Number"]
	event.BridgeSignalBond = eventMap["variable_signal_bond"]
	event.LastBridgeTo = eventMap["variable_last_bridge_to"]
	event.LastBridgehangup = eventMap["variable_last_bridge_hangup_cause"]
	event.OtherLegDestNumber = eventMap["Other-Leg-Destination-Number"]
	event.Dtmf = eventMap["DTMF-Digit"]
	event.DtmfDuration = eventMap["DTMF-Duration"]
	event.EventDate = UnixMicroStrToTime(eventMap["Event-Date-Timestamp"])
	if eventMap["Presence-Call-Direction"] == "inbound" {
		event.Who = "callee"
	} else {
		event.Who = "caller"
	}
	if eventMap["Caller-Caller-ID-Number"] != "anonymous" && eventMap["Caller-Caller-ID-Number"] != "" {
		event.CallerNumber = eventMap["Caller-Caller-ID-Number"]
	} else if eventMap["Caller-Orig-Caller-ID-Number"] != "" {
		event.CallerNumber = eventMap["Caller-Orig-Caller-ID-Number"]
	} else {
		event.CallerNumber = "anonymous"
	}
	if eventMap["Caller-Callee-ID-Number"] != "" {
		event.CalleeNumber = eventMap["Caller-Callee-ID-Number"]
	} else {
		event.CalleeNumber = eventMap["Caller-Destination-Number"]
	}
	event.ApiCommand = eventMap["API-Command"]
	event.ApiCommandArgument = eventMap["API-Command-Argument"]
	event.OriginationCallerIdName = eventMap["variable_origination_caller_id_name"]
	event.OriginationCalleeIdName = eventMap["variable_origination_callee_id_name"]
	event.EffectiveCallerIdName = eventMap["variable_effective_caller_id_name"]
	event.EffectiveCalleeIdName = eventMap["variable_effective_callee_id_name"]
	event.SipCalleeIdName = eventMap["variable_sip_callee_id_name"]
	event.EffectiveCalleeIdNumber = eventMap["variable_effective_callee_id_number"]
	event.StartTime = UnixStrToTime(eventMap["variable__START_TIME"])
	event.OtherLegCalleeIdName = eventMap["Other-Leg-Callee-ID-Name"]
	return event
}

func FormatDateTimeFromMicrosecondsLogFormat(timestamp string) string {
	if timestamp == "" {
		return timestamp
	}
	timestampInt64, _ := strconv.ParseInt(timestamp, 10, 64)
	timeMicroSeconds := time.Unix(0, timestampInt64*int64(time.Microsecond))
	timestampResult := strconv.FormatInt(timeMicroSeconds.Unix(), 10)
	return timestampResult
}

func GetValueIfExistsString(eventMap map[string]string, key string, currentValue string) string {
	if _, exist := eventMap[key]; exist {
		return eventMap[key]
	} else {
		return currentValue
	}
}

func GetValueIfExistsTime(eventMap map[string]string, key string, currentValue time.Time) time.Time {
	if _, exist := eventMap[key]; exist {
		return UnixMicroStrToTime(eventMap[key])
	} else {
		return currentValue
	}
}

func UnixMicroStrToTime(s string) time.Time {
	t, _ := strconv.ParseInt(s, 10, 64)
	if t == 0 {
		return time.Time{}
	} else {
		return time.UnixMicro(t)
	}
}

func UnixStrToTime(s string) time.Time {
	t, _ := strconv.ParseInt(s, 10, 64)
	if t == 0 {
		return time.Time{}
	} else {
		return time.Unix(t, 0)
	}
}

func ParseKvStr(s string) map[string]string {
	res := make(map[string]string)
	if s == "" {
		return res
	}
	items := strings.Split(s, "|")
	for _, item := range items {
		kv := strings.Split(item, "=")
		if len(kv) != 2 {
			continue
		}
		res[kv[0]] = kv[1]
	}
	return res
}

func NormStr(s string) string {
	if s == "nil" {
		return ""
	} else {
		return s
	}
}
