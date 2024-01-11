package sessionsservice

import (
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var sessions Sessions

type Sessions struct {
	mutex sync.Mutex
	list  []Session
}

type Session struct {
	//Used on CDR
	CallerUid               string
	CalleeUid               string
	DateStart               time.Time
	OriginalCallerNum       string
	OriginalCalleeNum       string
	CallerNum               string
	CalleeNum               string
	CallDirection           string
	CallEvent               string
	FsDirection             string
	HangupSide              string
	HangupReason            string
	DateRing                time.Time
	DateCon                 time.Time
	CallState               string
	OriginationCallerIdName string
	OriginationCalleeIdName string
	EffectiveCallerIdName   string
	EffectiveCalleeIdName   string
	OtherLegCalleeIdName    string
}

func LockSessions() {
	sessions.mutex.Lock()
}

func UnlockSessions() {
	sessions.mutex.Unlock()
}

func GetSessions() *[]Session {
	return &sessions.list
}

func SetSessions(newSessions []Session) []Session {
	sessions.list = newSessions
	return sessions.list
}

func SessionToSessionsService(session *Session) *SessionCopy {
	return &SessionCopy{CallerUid: session.CallerUid,
		CalleeUid:               session.CalleeUid,
		DateStart:               timestamppb.New(session.DateStart),
		OriginalCallerNum:       session.OriginalCallerNum,
		OriginalCalleeNum:       session.OriginalCalleeNum,
		CallerNum:               session.CallerNum,
		CalleeNum:               session.CalleeNum,
		CallDirection:           session.CallDirection,
		CallEvent:               session.CallEvent,
		FsDirection:             session.FsDirection,
		HangupSide:              session.HangupSide,
		HangupReason:            session.HangupReason,
		DateRing:                timestamppb.New(session.DateRing),
		DateCon:                 timestamppb.New(session.DateCon),
		CallState:               session.CallState,
		OriginationCallerIdName: session.OriginationCallerIdName,
		OriginationCalleeIdName: session.OriginationCalleeIdName,
		EffectiveCallerIdName:   session.EffectiveCallerIdName,
		EffectiveCalleeIdName:   session.EffectiveCalleeIdName,
		OtherLegCalleeIdName:    session.OtherLegCalleeIdName,
	}
}

func GetSessionsCopyService(sessions []Session) *SessionsCopy {
	var sessionsCopy SessionsCopy
	for _, session := range sessions {
		sessionsCopy.SessionCopy = append(sessionsCopy.SessionCopy, SessionToSessionsService(&session))
	}

	return &sessionsCopy
}

func SessionServiceToSession(sessionCopy *SessionCopy) *Session {
	var session Session
	session.CallerUid = sessionCopy.GetCallerUid()
	session.CalleeUid = sessionCopy.GetCalleeUid()
	session.DateStart = sessionCopy.GetDateStart().AsTime()
	session.OriginalCallerNum = sessionCopy.GetOriginalCallerNum()
	session.OriginalCalleeNum = sessionCopy.GetOriginalCalleeNum()
	session.CallerNum = sessionCopy.GetCallerNum()
	session.CalleeNum = sessionCopy.GetCalleeNum()
	session.CallDirection = sessionCopy.GetCallDirection()
	session.CallEvent = sessionCopy.GetCallEvent()
	session.FsDirection = sessionCopy.GetFsDirection()
	session.HangupSide = sessionCopy.GetHangupSide()
	session.HangupReason = sessionCopy.GetHangupReason()
	session.DateRing = sessionCopy.GetDateRing().AsTime()
	session.DateCon = sessionCopy.GetDateCon().AsTime()
	session.CallState = sessionCopy.GetCallState()
	session.OriginationCallerIdName = sessionCopy.GetOriginationCallerIdName()
	session.OriginationCalleeIdName = sessionCopy.GetOriginationCalleeIdName()
	session.EffectiveCallerIdName = sessionCopy.GetEffectiveCallerIdName()
	session.EffectiveCalleeIdName = sessionCopy.GetEffectiveCalleeIdName()
	session.OtherLegCalleeIdName = sessionCopy.GetOtherLegCalleeIdName()
	return &session
}

func SessionsCopyServiceToSessions(sessionsCopy *SessionsCopy) []Session {
	sessions.list = nil
	for _, session := range sessionsCopy.SessionCopy {
		sessions.list = append(sessions.list, *SessionServiceToSession(session))
	}
	return sessions.list
}

func GetSession(callerUid string, calleeUid string, exactly bool, onlyOneUid bool) (*Session, int, bool) {
	if exactly {
		for id, session := range sessions.list {
			if session.CallerUid == callerUid && session.CalleeUid == calleeUid {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CallerUid == calleeUid && session.CalleeUid == callerUid {
				return &session, id, true
			}
		}
	} else {
		if calleeUid == "" {
			calleeUid = "not_exist"
		}
		if callerUid == "" {
			callerUid = "not_exist"
		}
		for id, session := range sessions.list {
			if session.CallerUid == callerUid && session.CalleeUid == calleeUid {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CallerUid == calleeUid && session.CalleeUid == callerUid {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CallerUid == callerUid && session.CalleeUid == "" {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CalleeUid == calleeUid && session.CallerUid == "" {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CalleeUid == callerUid && session.CallerUid == "" {
				return &session, id, true
			}
		}
		for id, session := range sessions.list {
			if session.CallerUid == calleeUid && session.CalleeUid == "" {
				return &session, id, true
			}
		}
		if onlyOneUid {
			for id, session := range sessions.list {
				if session.CallerUid == callerUid {
					return &session, id, true
				}
			}
			for id, session := range sessions.list {
				if session.CalleeUid == calleeUid {
					return &session, id, true
				}
			}
			for id, session := range sessions.list {
				if session.CallerUid == calleeUid {
					return &session, id, true
				}
			}
			for id, session := range sessions.list {
				if session.CalleeUid == callerUid {
					return &session, id, true
				}
			}
		}

	}

	return nil, 0, false
}

func RemoveSession(callerUid string, calleeUid string) bool {
	for i, session := range sessions.list {
		if session.CallerUid == callerUid && session.CalleeUid == calleeUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}
	for i, session := range sessions.list {
		if session.CallerUid == calleeUid && session.CalleeUid == callerUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}
	for i, session := range sessions.list {
		if session.CallerUid == callerUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}
	for i, session := range sessions.list {
		if session.CalleeUid == calleeUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}
	for i, session := range sessions.list {
		if session.CalleeUid == callerUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}
	for i, session := range sessions.list {
		if session.CallerUid == calleeUid {
			sessions.list = append(sessions.list[:i], sessions.list[i+1:]...)
			return true
		}
	}

	return false
}
