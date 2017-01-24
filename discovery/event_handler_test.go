package discovery

import (
	"net"
	"testing"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/assert"
)

var portInd = 0

func createMemberEvent(eventType serf.EventType) serf.MemberEvent {
	portInd++
	return serf.MemberEvent{
		Type: eventType,
		Members: []serf.Member{
			{
				Addr: net.ParseIP("127.0.0.1"),
				Port: uint16(portInd),
			},
		},
	}
}

// TestListen will test if all the listen can be done, and shutdown function is working
func TestListen(t *testing.T) {
	eventHandler := newTestEventHandler(false)
	handler := NewEventHandler(eventHandler.handler)

	eventHandler.wg.Add(4)
	event1 := createMemberEvent(serf.EventMemberJoin)
	handler.HandleEvent(event1)
	event2 := createMemberEvent(serf.EventMemberFailed)
	handler.HandleEvent(event2)
	event3 := createMemberEvent(serf.EventMemberReap)
	handler.HandleEvent(event3)
	event4 := createMemberEvent(serf.EventMemberLeave)
	handler.HandleEvent(event4)

	// we don't handle member update, userEvent and Query
	event5 := createMemberEvent(serf.EventMemberUpdate)
	handler.HandleEvent(event5)
	handler.HandleEvent(serf.UserEvent{})
	handler.HandleEvent(&serf.Query{})

	eventHandler.wg.Wait()

	assert.True(t, eventHandler.containMemberEvent(event1), "join event not propergated")
	assert.True(t, eventHandler.containMemberEvent(event2), "fail event not propergated")
	assert.True(t, eventHandler.containMemberEvent(event3), "reap event not propergated")
	assert.True(t, eventHandler.containMemberEvent(event4), "leave event not propergated")
	assert.False(t, eventHandler.containMemberEvent(event5), "update event is propergated")
}
