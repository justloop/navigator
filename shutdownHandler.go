package navigator

import (
	log "github.com/Sirupsen/logrus"
	"github.com/justloop/navigator/discovery"
	"github.com/justloop/navigator/utils"
)

// logTagListener is the logging tag for EventListener
var logTagListener = "navigator.ShutdownEventListener"

// ShutdownHandler is a event listener that responsible to update hash ring
type ShutdownHandler struct {
	// identity is the identity of current server
	identity string

	// onShutdown is the callback when discovery is shutdown or reaped by the cluster
	onShutdown func()
}

// NewShutdownEventHandler create a new EventListener from a ring instance
func NewShutdownEventHandler(onShutdown func(), identity string) *ShutdownHandler {
	return &ShutdownHandler{
		onShutdown: onShutdown,
		identity:   identity,
	}
}

func (h *ShutdownHandler) handle(servers []string) {
	if utils.StrSliceContains(servers, h.identity) {
		log.Info(logTagListener, "shutdown message detected")
		h.onShutdown()
	}
}

// Handler to handle shutdown events
// In case this server is reaped by the cluster
func (h *ShutdownHandler) Handler(event discovery.MemberEvent) error {
	log.Debugf(logTagListener, "shutdown handler received message: %v", event)
	switch event.Type {
	case discovery.EventMemberJoin:
		log.Debugf(logTagListener, "shutdown handler ignored message: %v", event)
	case discovery.EventMemberFailed:
		h.handle(event.Servers)
	case discovery.EventMemberLeave:
		h.handle(event.Servers)
	case discovery.EventMemberReap:
		h.handle(event.Servers)
	}
	return nil
}
