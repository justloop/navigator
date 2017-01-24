/*
Package client is the client to communicate with the cluster, used by SDK or external client.

	1. it is cluster aware, knows all the alive servers
	2. it has it's own hashring, which meanings, client has to have a way to update the ring in the proper time.
	3. it also should be able to send request to a specific node, and tell the server how to forward the request (forward options)

Currently not using this because it has to expose a separate port to all other servers to perform tcp connections
*/
package client

import (
	"time"

	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/justloop/navigator/discovery"
	"github.com/justloop/navigator/hashring"
	"github.com/justloop/navigator/partition"
	"github.com/justloop/navigator/utils"
)

// logTag is the logging tag for NavigatorClient
var logTag = "navigator.client"

// Client is used to send request to the cluster
type Client interface {
	// GetServerList returns a list of connected alive servers together with their tags
	GetServerList() (map[string]map[string]string, error)

	// ResolveRead will resolve to a list of servers and partition id that this key is resolved to read from,
	// if any server read trail failed, proceed to next server in the list to read
	// return a map from shardId to servers
	ResolveRead(key string, op partition.Option) (map[string][]string, error)

	// ResolveWrite will resolve to a list of servers and partition id that this key is resolved to write to,
	// the writer will have to be responsible to write to all the servers, but ignore any fails
	// return a map from shardId to servers
	ResolveWrite(key string, op partition.Option) (map[string][]string, error)

	//Stats will return the stats regarding the discovery cluster
	Stats() (map[string]map[string]string, error)
}

// Impl is the implementation of client
type Impl struct {
	// the context
	config *Config

	// ring is the hashring
	ring hashring.HashRing

	// the discovery client
	dClient discovery.Client

	// resolver is the partition resolver
	resolver partition.Resolver
}

// New will create a new client
func New(ctx *Config) (*Impl, error) {
	ring := hashring.NewMapHashRing(ctx.ReplicaPoints)
	log.Info(logTag, "[NavigatorClient] HashRing created...")
	resolver := partition.NewRingResolver(ring, ctx.Strategy)

	serfClient, err := discovery.NewSerfClient(net.JoinHostPort(ctx.ClusterRPCAddr, strconv.Itoa(ctx.ClusterRPCPort)))
	if err != nil {
		return nil, nil
	}

	impl := &Impl{
		config:   SetDefaultContext(ctx),
		ring:     ring,
		resolver: resolver,
		dClient:  serfClient,
	}

	// start the clusterChangeProcessor
	go impl.startClusterChangeProcessor()

	return impl, nil
}

// constantly check the aliveservers from discover module, if changed, we will change the ring
func (c *Impl) startClusterChangeProcessor() {
	defer utils.DoPanicRecovery("NavigatorClient")
	log.Info(logTag, "[NavigatorClient] ClusterChangeProcessor started")
	tickChan := time.NewTicker(c.config.RingCacheTimeout).C
	for {
		// Add another loop so that if error happens, we should skip the replace of ring,
		// and restart the eventloop
	eventLoop:
		for {
			select {
			case <-c.config.Ctx.Done():
				log.Info(logTag, "[NavigatorClient] shutdown...")
				return
			case <-tickChan:
				servers, err := c.dClient.AliveServers()
				if err != nil {
					log.Warnf(logTag, "[NavigatorClient] GetAliveServers got error: %s", err)
					break eventLoop
				}
				checksum := utils.GetCheckSumFromNodes(servers)
				ringChecksum, err := c.ring.Checksum()
				if err != nil {
					log.Errorf(logTag, "[NavigatorClient] Get ring checksum got error: %s", err)
					break eventLoop
				}
				if checksum != ringChecksum {
					// got change of ring
					err = c.replaceRing(servers)
					if err != nil {
						log.Errorf(logTag, "[NavigatorClient] replaceRing %v got error: %s", servers, err)
					}
				}
			}
		}
	}
}

func (c *Impl) replaceRing(servers []string) error {
	nodes, err := c.ring.GetNodes()
	if err != nil {
		return err
	}
	toRemove, toAdd := utils.Difference(servers, nodes)
	for _, node := range toRemove {
		err1 := c.ring.RemoveNode(node)
		if err1 != nil {
			err = err1
		}
	}
	for _, node := range toAdd {
		err2 := c.ring.AddNode(node)
		if err2 != nil {
			err = err2
		}
	}
	return err
}

// GetServerList implements the Client interface
func (c *Impl) GetServerList() (map[string]map[string]string, error) {
	return c.dClient.AliveServersWithTags()
}

// ResolveRead implements the Client interface
func (c *Impl) ResolveRead(key string, op partition.Option) (map[string][]string, error) {
	return c.resolver.ResolveRead(key, op)
}

// ResolveWrite implements the Client interface
func (c *Impl) ResolveWrite(key string, op partition.Option) (map[string][]string, error) {
	return c.resolver.ResolveWrite(key, op)
}

//Stats implements the Client interface
func (c *Impl) Stats() (map[string]map[string]string, error) {
	return c.dClient.Stats()
}
