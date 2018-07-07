package discovery

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/hashicorp/serf/cmd/serf/command/agent"
	"github.com/hashicorp/serf/serf"
	log "github.com/sirupsen/logrus"
)

var (
	// logTag is the logging tag for memberlist module
	logTag = "discovery.node"
)

// SerfDiscoverNode is an wrap object of the serf node
// url: https://www.serf.io/
// Serf is for cluster membership, failure detection and orchestration, lightweight and highly available
type SerfDiscoverNode struct {
	config    *Config
	serfConf  *serf.Config
	delegate  *agent.Agent
	agentConf *agent.Config
}

// NewSerfDiscoverNode initialize the serf module with a naive listener,
// After created, need to invoke Start to start the serf module
func NewSerfDiscoverNode(ctx context.Context) (*SerfDiscoverNode, error) {
	return NewSerfDiscoverNodeWithConfig(&Config{})
}

// NewSerfDiscoverNodeWithConfig initialize the serf module with a naive listener,
// After created, need to invoke Start to start the serf module
func NewSerfDiscoverNodeWithConfig(config *Config) (*SerfDiscoverNode, error) {
	node := &SerfDiscoverNode{
		config: config,
	}
	node.agentConf, node.serfConf = getAgentAndSerfConfig(config)
	// ignore the logs
	agent, err := agent.Create(node.agentConf, node.serfConf, nil)
	node.delegate = agent
	return node, err
}

// NewSerfDiscoverNodeWithHandler initialize the serf module with a naive listener,
// After created, need to invoke Start to start the serf module
func NewSerfDiscoverNodeWithHandler(config *Config, handlers []HandlerFunc) (*SerfDiscoverNode, error) {
	node, err := NewSerfDiscoverNodeWithConfig(config)
	for _, handler := range handlers {
		node.delegate.RegisterEventHandler(NewEventHandler(handler))
	}
	return node, err
}

// AddEventHandler adds a eventhandler in order to listen to the cluster events
func (node *SerfDiscoverNode) AddEventHandler(handler func(event MemberEvent) error) error {
	node.delegate.RegisterEventHandler(NewEventHandler(handler))
	return nil
}

// Start will start this node, can return error
func (node *SerfDiscoverNode) Start() error {
	log.Debugf(logTag, "[DiscoveryNode] %v start", node.serfConf.NodeName)
	l, err := net.Listen("tcp", GetClientAddrStr(node.config))

	lw := agent.NewLogWriter(512)
	mult := io.MultiWriter(os.Stderr, lw)
	agent.NewAgentIPC(node.delegate, "", l, mult, lw)
	err1 := node.delegate.Start()
	if err == nil {
		err = err1
	}
	return err
}

// Join joins an existing Serf cluster. Returns the number of nodes
// successfully contacted. The returned error will be non-nil only in the
// case that no nodes could be contacted. If ignoreOld is true, then any
// user messages sent prior to the join will be ignored.
func (node *SerfDiscoverNode) Join(seed []string, ignoreOld bool) (int, error) {
	if len(seed) == 0 {
		seed = node.config.Seed
	}
	log.Debugf(logTag, "Node %s join the cluster with seed %v", node.LocalMember().Name, seed)
	return node.delegate.Join(seed, ignoreOld)
}

// Leave gracefully exits the cluster. It is safe to call this multiple
// times.
func (node *SerfDiscoverNode) Leave() error {
	log.Debugf(logTag, "Node %s safely leaving the cluster.", node.LocalMember().Name)
	err := node.delegate.Leave()
	return err
}

// Shutdown forcefully shuts down the Serf instance, stopping all network
// activity and background maintenance associated with the instance.
//
// This is not a graceful shutdown, and should be preceded by a call
// to Leave. Otherwise, other nodes in the cluster will detect this node's
// exit as a node failure.
//
// It is safe to call this method multiple times.
func (node *SerfDiscoverNode) Shutdown() error {
	log.Warnf(logTag, "Node %s shutdown without safe leave the cluster.", node.LocalMember().Name)
	err := node.delegate.Shutdown()
	return err
}

// LocalMember returns the Member information for the local node
func (node *SerfDiscoverNode) LocalMember() serf.Member {
	return node.delegate.Serf().LocalMember()
}

// LocalServer returns the server address of the local node
func (node *SerfDiscoverNode) LocalServer() string {
	member := node.delegate.Serf().LocalMember()
	return getServer(member)
}

// Members returns a point-in-time snapshot of the members of this cluster.
func (node *SerfDiscoverNode) Members() []serf.Member {
	return node.delegate.Serf().Members()
}

// Servers returns a point-in-time snapshot of the members of this cluster, in the format of string slice
func (node *SerfDiscoverNode) Servers() []string {
	return GetServers(node.Members())
}

// AliveMembers returns a point-in-time all the alive nodes
func (node *SerfDiscoverNode) AliveMembers() []serf.Member {
	aliveMembers := []serf.Member{}
	allMembers := node.Members()
	for _, member := range allMembers {
		if member.Status == serf.StatusAlive {
			aliveMembers = append(aliveMembers, member)
		}
	}
	return aliveMembers
}

// AliveServers returns a point-in-time all the alive nodes, in the format of string slice
func (node *SerfDiscoverNode) AliveServers() []string {
	return GetServers(node.AliveMembers())
}

// NumNodes returns the number of nodes in the serf cluster, regardless of
// their health or status.
func (node *SerfDiscoverNode) NumNodes() int {
	return len(node.AliveServers())
}

// RemoveFailedNode forcibly removes a failed node from the cluster
// immediately, instead of waiting for the reaper to eventually reclaim it.
// This also has the effect that Serf will no longer attempt to reconnect
// to this node.
func (node *SerfDiscoverNode) RemoveFailedNode(nodeStr string) error {
	return node.delegate.Serf().RemoveFailedNode(nodeStr)
}

// State is the current state of this Serf instance.
func (node *SerfDiscoverNode) State() serf.SerfState {
	return node.delegate.Serf().State()
}

// Ready will check if state is ready
func (node *SerfDiscoverNode) Ready() bool {
	return node.delegate.Serf().State() == serf.SerfAlive
}

// UserEvent is used to broadcast a custom user event with a given
// name and payload. The events must be fairly small, and if the
// size limit is exceeded and error will be returned.
func (node *SerfDiscoverNode) UserEvent(name string, payload []byte) error {
	return node.delegate.UserEvent(name, payload, false)
}

// SetTags is used to dynamically update the tags associated with
// the local node. This will propagate the change to the rest of
// the cluster. Blocks until a the message is broadcast out.
func (node *SerfDiscoverNode) SetTags(tags map[string]string) error {
	return node.delegate.SetTags(tags)
}

//GetServersWithTags will return a map of server to it's tags
func (node *SerfDiscoverNode) GetServersWithTags() map[string]map[string]string {
	result := make(map[string]map[string]string)
	members := node.delegate.Serf().Members()
	for _, member := range members {
		result[getServer(member)] = member.Tags
	}
	return result
}

//AliveServersWithTags will return a map of alive server to it's tags
func (node *SerfDiscoverNode) AliveServersWithTags() map[string]map[string]string {
	result := make(map[string]map[string]string)
	allMembers := node.Members()
	for _, member := range allMembers {
		if member.Status == serf.StatusAlive {
			result[getServer(member)] = member.Tags
		}
	}
	return result
}

// Stats will return a list of stats regarding the discovery cluster
func (node *SerfDiscoverNode) Stats() map[string]map[string]string {
	return node.delegate.Stats()
}

// GetServers will convert to list of Member to Server info
func GetServers(members []serf.Member) []string {
	servers := []string{}
	for _, member := range members {
		servers = append(servers, member.Addr.String()+":"+strconv.Itoa(int(member.Port)))
	}
	return servers
}

// GetServer will convert member object to string info
func getServer(member serf.Member) string {
	return member.Addr.String() + ":" + strconv.Itoa(int(member.Port))
}
