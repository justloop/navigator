/*
Package discovery does all the service discovery and clustering of the nodes

It can register EventListener to listen to the cluster events, namely [Join, Leave, Fail, Reap]

a more interesting thing is it can broadcast UserEvent to all the nodes, but the UserEvent has to be small enough not to overwhelm the discovery gossip system
*/
package discovery

// Node cluster interface
type Node interface {
	// SetEventListener, set the cluster event listener, all the cluter events and user events will be processed
	AddEventHandler(handler func(event MemberEvent) error) error
	// Start will start this node, without start this node is static, cannot use
	Start() error
	// Ready will identify if this node is ready
	Ready() bool
	// Join the current node to list of seed nodes
	Join(seed []string, ignoreOld bool) (int, error)
	// Gracefully leave the cluster system
	Leave() error
	// Force shutdown the system, should be able to call multiple times
	Shutdown() error
	// LocalServer will return the current server information as [address:port]
	LocalServer() string
	// Servers returns a point-in-time snapshot of the members of this cluster
	Servers() []string
	// AliveServers returns a point-in-time all the alive nodes
	AliveServers() []string
	// NumNodes return the number of nodes in the cluster
	NumNodes() int
	// UserEvent Propagate user events to the cluster
	UserEvent(name string, payload []byte) error
	// SetTags is used to dynamically update the tags associated with
	// the local node. This will propagate the change to the rest of
	// the cluster. Blocks until a the message is broadcast out.
	SetTags(tags map[string]string) error
	//GetServersWithTags will return a map of server to it's tags
	GetServersWithTags() map[string]map[string]string
	//AliveServersWithTags will return a map of alive server to it's tags
	AliveServersWithTags() map[string]map[string]string
}

// Client is the client to talk to Node
type Client interface {
	// Servers will return the servers
	Servers() ([]string, error)
	// AliveServers will return the current alive servers
	AliveServers() ([]string, error)
	//AliveServersWithTags will return a map of alive server to it's tags
	AliveServersWithTags() (map[string]map[string]string, error)
	// Close will close the session to discovery cluster
	Close() error
}
