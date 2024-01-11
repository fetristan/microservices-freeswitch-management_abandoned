package main

import (
	events "github.com/fetristan/tlc_events"
	"github.com/fetristan/tlc_sessions/sessionsservice"
)

func logSession(event events.Event, session *sessionsservice.Session, text string) {
	var newSession sessionsservice.Session
	if session == nil {
		session = &newSession
	}
	log.Debugf("%s : %s : %s (%s / %s / %s) (%s:%s)(%s:%s) (%s) : %s -> %s", event.EventDate.Format("2006-01-02 15:04:05.999999 MST"), text, event.EventName, event.EventSubclass, event.ApiCommand, event.ApiCommandArgument, event.CallerNumber, event.CalleeNumber, session.CallerNum, session.CalleeNum, event.RecordId, event.UniqueId, event.OtherId)
}
