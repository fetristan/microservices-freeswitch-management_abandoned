package main

import (
	"unicode/utf8"

	events "github.com/fetristan/tlc_events"
	"github.com/fetristan/tlc_sessions/sessionsservice"
)

type NumData struct {
	DidData       DidDataFromDb
	ExtensionData ExtensionDataFromDb
	IsExternal    bool
	IsExtension   bool
	IsDid         bool
}

func mergeEventMapIntoSession(eventMap map[string]string, session *sessionsservice.Session) *sessionsservice.Session {
	session.RecordingName = events.GetValueIfExistsString(eventMap, "Record-File-Path", session.RecordingName)
	session.OriginalCallerNum = events.GetValueIfExistsString(eventMap, "original_caller", session.OriginalCallerNum)
	session.OriginalCalleeNum = events.GetValueIfExistsString(eventMap, "original_callee", session.OriginalCalleeNum)
	session.CallerType = events.GetValueIfExistsString(eventMap, "CALLER_TYPE", session.CallerType)
	session.CalleeType = events.GetValueIfExistsString(eventMap, "CALLEE_TYPE", session.CalleeType)
	session.CallDirection = events.GetValueIfExistsString(eventMap, "CALL_DIRECTION", session.CallDirection)
	session.CallType = events.GetValueIfExistsString(eventMap, "CALL_TYPE", session.CallType)
	session.OriginationCallerIdName = events.GetValueIfExistsString(eventMap, "origination_caller_id_name", session.OriginationCallerIdName)
	session.OriginationCalleeIdName = events.GetValueIfExistsString(eventMap, "origination_callee_id_name", session.OriginationCalleeIdName)
	session.EffectiveCallerIdName = events.GetValueIfExistsString(eventMap, "effective_caller_id_name", session.EffectiveCallerIdName)
	session.EffectiveCalleeIdName = events.GetValueIfExistsString(eventMap, "effective_callee_id_name", session.EffectiveCalleeIdName)
	session.OtherLegCalleeIdName = events.GetValueIfExistsString(eventMap, "sip_callee_id_name", session.OtherLegCalleeIdName)
	return session
}

func fixSessionUids(event events.Event, session *sessionsservice.Session) {
	if session.CallerUid != event.UniqueId {
		session.CallerUid = event.UniqueId
	}
	if session.CalleeUid != event.OtherId {
		session.CalleeUid = event.OtherId
	}
}

func mergeKeyValueMap(mapSrc map[string]string, mapDst map[string]string) map[string]string {
	for k, v := range mapSrc {
		mapDst[k] = v
	}
	return mapDst
}

func updateSessionFromDatabase(session *sessionsservice.Session, event events.Event) {
	values := make(map[string]string)
	callerData := getNumData(session.CallerNum)
	calleeData := getNumData(session.CalleeNum)
	values = mergeKeyValueMap(checkCallerCalleeType(session, callerData, calleeData), values)
	values = mergeKeyValueMap(checkCopyDataFromOtherLeg(session, callerData, calleeData, event), values)
	SetVarMultiple(session.CallerUid, session.CalleeUid, values)
}

func getNumData(num string) NumData {
	var NumData NumData
	NumData.ExtensionData = getExtensionParamsByExtension(num)
	if !NumData.ExtensionData.Num.Valid {
		NumData.DidData = getDidParamsByDid(num)
		if !NumData.DidData.Sda.Valid {
			NumData.IsExternal = true
		} else {
			NumData.IsDid = true
		}
	} else {
		NumData.IsExtension = true
	}
	return NumData
}

func checkCallerCalleeType(session *sessionsservice.Session, callerData NumData, calleeData NumData) map[string]string {
	values := make(map[string]string)
	values["CALLER_TYPE"] = checkNumType(callerData)
	values["CALLEE_TYPE"] = checkNumType(calleeData)
	return values
}

func checkNumType(numData NumData) string {
	var numType = "0"
	if numData.IsExtension {
		if numData.ExtensionData.IsAgent.Valid && numData.ExtensionData.IsAgent.String == "0" && numData.ExtensionData.IsLogged.Valid && numData.ExtensionData.IsLogged.String != "0" &&
			numData.ExtensionData.UserId.Valid && numData.ExtensionData.UserId.String != "0" {
			numType = "2"
		} else if numData.ExtensionData.IsAgent.Valid && numData.ExtensionData.IsAgent.String == "1" {
			numType = "1"
		} else if numData.ExtensionData.BeforeCallIvrArgs.Valid && utf8.RuneCountInString(numData.ExtensionData.BeforeCallIvrArgs.String) > 7 {
			numType = "4"
		} else {
			numType = "3"
		}
	}
	return numType
}

func setCustomsVariablesNeededFromEvent(event events.Event, session *sessionsservice.Session) {
	session.OriginalCallerNum = event.OriginalCaller2
	session.OriginalCalleeNum = event.OriginalCallee
	session.CallerType = event.CallerType
	session.CalleeType = event.CalleeType
	session.CallDirection = event.CallDirection
	session.CallType = event.CallType
	session.OriginationCallerIdName = event.OriginationCallerIdName
	session.OriginationCalleeIdName = event.OriginationCalleeIdName
	session.EffectiveCallerIdName = event.EffectiveCallerIdName
	session.EffectiveCalleeIdName = event.EffectiveCalleeIdName
	session.OtherLegCalleeIdName = event.OtherLegCalleeIdName
}
