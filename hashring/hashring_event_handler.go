package hashring

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/justloop/navigator/discovery"
)

// logTagListener is the logging tag for EventListener
var logTagListener = "navigator.hashring"

// EventHandler is a event listener that responsible to update hash ring
type EventHandler struct {
	ring HashRing
}

// NewEventHandler create a new EventListener from a ring instance
func NewEventHandler(ring HashRing) *EventHandler {
	return &EventHandler{
		ring: ring,
	}
}

func (handler *EventHandler) print(tag string) {
	nodes, err := handler.ring.GetNodes()
	if err != nil {
		log.Warnf(logTagListener, "ring GetNumNodes failed", err)
	}
	log.Debugf(logTagListener, fmt.Sprintf("ring %s: %v", tag, nodes))
}

// Handler is the handler of memberevents
func (handler *EventHandler) Handler(event discovery.MemberEvent) error {
	log.Infof(logTagListener, fmt.Sprintf("ring received message: %v", event))
	handler.print("before")
	for _, server := range event.Servers {
		var err error
		switch event.Type {
		case discovery.EventMemberJoin:
			err = handler.ring.AddNode(server)
		case discovery.EventMemberFailed:
			err = handler.ring.RemoveNode(server)
		case discovery.EventMemberLeave:
			err = handler.ring.RemoveNode(server)
		case discovery.EventMemberReap:
			err = handler.ring.RemoveNode(server)
		}
		if err != nil {
			return err
		}
	}
	handler.print("after")
	return nil
}
