package discovery

// EventListener is the listener of all the Events, including MemberEvent and UserEvent

// EventType is the potential event type for member event
type EventType int

// All the message types related to member events
const (
	EventMemberJoin EventType = iota
	EventMemberLeave
	EventMemberFailed
	EventMemberReap
)

// MemberEvent is the member event received
type MemberEvent struct {
	// Type is one of the EventType
	Type EventType
	// Servers is the list of servers related to this event, format: [address:port]
	Servers []string
}

// HandlerFunc defines a function to handle the member events
type HandlerFunc func(event MemberEvent) error
