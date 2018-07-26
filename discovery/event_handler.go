package discovery

import (
	"errors"

	"github.com/hashicorp/serf/serf"
	"github.com/justloop/navigator/utils"
	log "github.com/sirupsen/logrus"
)

// NavigatorEventHandler is the wraper to agent.EventHanlder to hide serf from user
type NavigatorEventHandler struct {
	handler func(event MemberEvent) error
}

// NewEventHandler will return a agent.EventHandler from handler Function
func NewEventHandler(handler func(event MemberEvent) error) *NavigatorEventHandler {
	return &NavigatorEventHandler{
		handler: handler,
	}
}

// HandleEvent extends agent.EventHandler
func (h *NavigatorEventHandler) HandleEvent(event serf.Event) {
	err := h.processEvent(event)
	if err != nil {
		s := err.Error()
		log.Errorf(logTag, "EventHanlder error, type: %T; value: %q\n", s, s)
	}
}

func (h *NavigatorEventHandler) processEvent(event serf.Event) error {
	switch casted := event.(type) {
	case serf.MemberEvent:
		return h.processMemberEvent(casted)
	case serf.UserEvent:
		log.Debugf(logTag, "EventHanlder received user event: %s, ignored", utils.GetJSONStr(casted))
		return nil
	case *serf.Query:
		log.Debugf(logTag, "EventHanlder received query event: %s, ignored", utils.GetJSONStr(casted))
		return nil
	case error:
		err, _ := casted.(error)
		log.Warnf(logTag, "EventHanlder got error: %s, %s", utils.GetJSONStr(casted), err)
		// if err received, restart
		log.Debugf(logTag, "EventHanlder restarte")
		return err
	}
	return nil
}

func (h *NavigatorEventHandler) processMemberEvent(casted serf.MemberEvent) error {
	var eventType EventType

	switch casted.Type {
	case serf.EventMemberJoin:
		eventType = EventMemberJoin
	case serf.EventMemberFailed:
		eventType = EventMemberFailed
	case serf.EventMemberLeave:
		eventType = EventMemberFailed
	case serf.EventMemberReap:
		eventType = EventMemberReap
	case serf.EventMemberUpdate:
		return nil
	default:
		log.Warnf(logTag, "unknown member event received: %s", utils.GetJSONStr(casted))
		return errors.New("unknown member event received")
	}

	memberEvent := MemberEvent{
		Type:    eventType,
		Servers: GetServers(casted.Members),
	}
	return h.handler(memberEvent)
}
