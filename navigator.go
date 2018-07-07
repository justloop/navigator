/*
Package navigator is the entry point for the distributed framework, all the functions are wrapped up here.

Navigator can serve as the distributed layer. It very light weighted, just contains a set of libs for distributed system

Components

	1. Discovery: Used to discover new nodes and maintains a list of alive servers follows SWIM protocol
	2. HashRing: A consistent hashring implementation
	3. Partition: A partition module using user implemented PartitionStrategy, and resolve a key to a list of servers using Hashring information
	4. Navigator: Implementation of distributed layer by combining different components.

How to use?

Configuration of navigator is easy and trivial. Configuration file is all you need to configure the navigator node:
	// Config is the configuration related to Navigator
	type Config struct {

	       // ReplicaPoints is the number of replications, optional, default 1
	       ReplicaPoints int

	       // DConfig is the configuration for discovery client, optional, default use default discovery config
	       DConfig *discovery.Config

	       // Strategy is the partition strategy, optional, default IdentityStrategy, will resolve key to key itself and map key to a server
	       Strategy partition.Strategy

	       // SeedsService is a implementation of seedsService that used to auto get a list of seeds to join
           SeedsService seeds.Seeds

	       // OnMemberEvent is the callback handler to be invoked when repartition happens,
	       // and the node will need to reload data, optional, default, no rehashHandler
	       OnMemberEvent func(event discovery.MemberEvent) error

	       // OnShutdown is the callback when discovery is shutdown or reaped by the cluster
	       OnShutdown func()
	}

The rest all have default configuration except Strategy, application will have to extend the strategy depend on it's own scenario.

	navigatorConfig := &navigator.Config{
	       ReplicaPoints: 2,
	       Strategy:      myStrategy,
	       DConfig: &discovery.Config{
		      Address: iputil.GetHostname(),
		      Seed:    []string{}, // use seed service
	       },
	}


	n := navigator.New(navigatorConfig)
	// To start
	n.Start()


	// To stop
	n.Stop()

With navigator framework, you will be able to connect the individual nodes together to form a cluster. Then you will be able to Resolve your key to specific shard and to a list of servers holding the shard according to your partition strategy. By using the ResolveRead and ResolveWrite function

	// ResolveRead will resolve to a list of servers and partition id that this key is resolved to read from,
	// if any server read trail failed, proceed to next server in the list to read
	// return a map from shardId to servers
	ResolveRead(key string, op partition.Option) (map[string][]string, error)

	// ResolveWrite will resolve to a list of servers and partition id that this key is resolved to write to,
	// the writer will have to be responsible to write to all the servers, but ignore any fails
	// return a map from shardId to servers
	ResolveWrite(key string, op partition.Option) (map[string][]string, error)

Service Discovery

Service Discovery module is SWIM based raft consensus. Node will be able to discovery and add new nodes, and remove dead nodes by gossip protocols. More details about swim and raft: https://raft.github.io/, https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf

Currently we use Hashicorp serf implementation of SWIM, here is the url: https://www.serf.io/

The reason of choosen that is because serf is light-weighted, and it can provide all the functions we need, just enough:

	Gossip-based membership
	Failure detection
	Custom events
	Seed service

Because SWIM based service discovery has this draw back of having to provide initial seeds for the cluster to form. We want to get rid of this for easier deployment configurations. The seed service is meant for this purpose. With seed service, we would be able to get the latest seeds, rather than provide manually.

Consistent HashRing

Consistent hashring hashes a key to a server, and moreover when server change, the number of keys affected is minimized. https://dzone.com/articles/simple-magic-consistent. The replicas of the current shard are stored in the next nodes of current node in the ring.

Partition

Partition Resolver will use the user defined PartitionStrategy to hash a key to a shardID, and find the servers that is managing the shard and return it.

Why it works?

The core idea behind the distributed framework is the HashRing replication among all the nodes. By using the service discovery module, the Rings across all cluster nodes are assumed to be the same. With the feature of consistent hashing provided by the HashRing, the shard and replicas will be consistently assigned to servers.
*/
package navigator

import (
	"context"
	"sync"
	"time"

	"github.com/justloop/navigator/discovery"
	"github.com/justloop/navigator/hashring"
	"github.com/justloop/navigator/partition"
	"github.com/justloop/navigator/utils"
	log "github.com/sirupsen/logrus"
)

// logTag is the logging tag related to this navigator
var logTag = "navigator.service"

// state represents the internal state of a Navigator instance.
type state uint

const (
	// created means the V1 instance has been created, but it has not joined the cluster
	created state = iota
	// ready means navigator is now ready to receive requests.
	ready
	// destroyed means the navigator instance has been shut down, is no longer
	// ready for requests and cannot be revived.
	destroyed
)

// Navigator is interface for navigator package.
type Navigator interface {
	// Debug will return the debug information in navigator
	// as map[string]interface{}
	Debug() map[string]interface{}

	// Ready will return the readiness of the server
	Ready() bool

	// Stop will stop this node
	Stop()

	// Start will start to communicate with other nodes,
	// it will start to try seed nodes first, if no seed nodes provided,
	// it will contact redis for existing nodes,
	// if neither of these are provided will be a single-node cluster
	Start() error

	// SetTags will tag current server with a list of meta, which will be propergate to each server automatically.
	SetTags(tags map[string]string) error

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
}

// Impl is the implementation of V1
// this implementes register.Navigator interface
type Impl struct {
	// cancelCtx is the context to shutdown the goroutings except for node
	cancelCtx context.Context

	// cancelFuc is the cancelFunc for the cancelCtx
	cancelFuc context.CancelFunc

	// config is the navigator configuration
	config *Config

	// dNode is the discovery node
	dNode discovery.Node

	// ring is the hashring
	ring hashring.HashRing

	// resolver is the partition resolver
	resolver partition.Resolver

	// startTime is the start time of this node
	startTime time.Time

	// state is the state of current navigator
	state state

	// stateMutex is the mutex to access state
	stateMutex sync.RWMutex
}

// New will create a new V1 instance
func New(config *Config) Navigator {
	config = setDefaultConfig(config)
	navigator := &Impl{
		config: config,
	}
	navigator.cancelCtx, navigator.cancelFuc = context.WithCancel(context.Background())
	if config.KeyHashFunc == nil {
		navigator.ring = hashring.NewMapHashRing(config.ReplicaPoints)
		log.Info(logTag, "HashRing created...")
	} else {
		navigator.ring = hashring.NewMapHashRingWithKeyHash(config.ReplicaPoints, config.KeyHashFunc)
		log.Info(logTag, "HashRing with Key Hash Function created...")
	}

	navigator.resolver = partition.NewRingResolver(navigator.ring, config.Strategy)
	log.Info(logTag, "Resolver created...")

	dNode, err := discovery.NewSerfDiscoverNodeWithConfig(config.DConfig)
	navigator.dNode = dNode
	if err != nil {
		panic("Discovery Node creation failure" + err.Error())
	}
	log.Info(logTag, "[Navigator] Discovery node created...")
	err = navigator.dNode.AddEventHandler(hashring.NewEventHandler(navigator.ring).Handler)
	if err != nil {
		panic("Adding ring listenner to discovery failed" + err.Error())
	}
	err = navigator.dNode.AddEventHandler(config.OnMemberEvent)
	if err != nil {
		panic("Adding rehash listenner to discovery failed" + err.Error())
	}
	log.Info(logTag, "Discovery rehash listener added...")
	err = navigator.dNode.AddEventHandler(NewShutdownEventHandler(config.OnShutdown, discovery.GetClusterAddrStr(config.DConfig)).Handler)
	if err != nil {
		panic("Adding shutdown listenner to discovery failed" + err.Error())
	}
	log.Info(logTag, "Discovery shutdown listener adeed...")

	err = navigator.dNode.Start()
	if err != nil {
		panic("Create node failed" + err.Error())
	}
	log.Info(logTag, "Discovery node started...")

	// no seeds, will use the seeds service
	if len(config.DConfig.Seed) == 0 && navigator.config.SeedsService != nil {
		log.Info(logTag, "Seeds not provided, use seeds service instead...")
		navigator.config.SeedsService.Start()
		log.Info(logTag, "Seeds service started...")
	}

	navigator.setState(created)

	return navigator
}

// Debug will return the list of debug info related to navigator
func (n *Impl) Debug() map[string]interface{} {
	debug := map[string]interface{}{}
	debug["status"] = n.state
	debug["uptime"] = time.Since(n.startTime).Seconds()
	debug["discovery"] = n.dNode.AliveServers()
	nodes, _ := n.ring.GetNodes()
	debug["hashring"] = nodes
	debug["strategy"] = n.config.Strategy.GetName()
	for k, v := range n.dNode.Stats() {
		debug[k] = v
	}
	return debug
}

// Start implements Navigator
func (n *Impl) Start() error {
	if len(n.config.DConfig.Seed) > 0 {
		log.Infof(logTag, "Seeds provided, will join the cluster: %v", n.config.DConfig.Seed)
		_, err := n.dNode.Join(n.config.DConfig.Seed, false)
		if err != nil {
			return err
		}
		log.Infof(logTag, "Successfully joined the cluster with %d nodes", n.dNode.NumNodes())
	} else {
		seeds := []string{}
		var err error
		if n.config.SeedsService != nil {
			log.Info(logTag, "No seeds provided, will use seeds service")
			seeds, err = n.config.SeedsService.GetN(3)
			if err != nil {
				return err
			}
		}
		if len(seeds) > 0 {
			_, err = n.dNode.Join(seeds, false)
			if err != nil {
				return err
			}
			log.Infof(logTag, "Successfully joined the cluster with %d nodes", n.dNode.NumNodes())
		} else {
			log.Info(logTag, "No seeds found in seed service, join self as a single node cluster...")
			_, err = n.dNode.Join([]string{discovery.GetClusterAddrStr(n.config.DConfig)}, false)
			if err != nil {
				return err
			}
		}
	}
	n.startTime = time.Now()
	// periodically to join the cluster, eventually will get the whole cluster
	n.StartJoinService()
	n.setState(ready)
	return nil
}

// StartJoinService is a separate go routing will periodically join the new nodes
func (n *Impl) StartJoinService() {
	refreshTicker := time.NewTicker(n.config.SeedsRefreshInterval)
	go func() {
		defer utils.DoPanicRecovery("join-service")

		log.Infof(logTag, "Starting join service process...")
		for {
			select {
			case <-refreshTicker.C:
				seeds, err := n.config.SeedsService.GetN(3)
				if err != nil {
					log.Warnf(logTag, "Failed to get seeds", err)
				}
				if len(seeds) > 1 {
					_, err = n.dNode.Join(seeds, false)
					if err != nil {
						log.Warnf(logTag, "Failed to join cluster", err)
					}
					log.Infof(logTag, "Successfully joined the cluster with %d nodes", n.dNode.NumNodes())
				} else {
					log.Infof(logTag, "Not enough seeds to join", seeds)
				}
			case <-n.cancelCtx.Done():
				log.Infof(logTag, "Stop join service info received")
				refreshTicker.Stop()
				return
			}
		}
	}()
}

// Stop implements Navigator
func (n *Impl) Stop() {
	err := n.dNode.Leave()
	if err != nil {
		log.Errorf(logTag, "Node leave got error", err)
	}
	n.cancelFuc()
	n.setState(destroyed)
}

// Ready implements Navigator
func (n *Impl) Ready() bool {
	if n.getState() != ready {
		return false
	}
	return n.dNode.Ready()
}

// SetTags implements Navigator
func (n *Impl) SetTags(tags map[string]string) error {
	if n.getState() != ready {
		return ErrNotReady
	}
	return n.dNode.SetTags(tags)
}

// GetServerList implements Navigator
func (n *Impl) GetServerList() (map[string]map[string]string, error) {
	if n.getState() != ready {
		return map[string]map[string]string{}, ErrNotReady
	}
	return n.dNode.AliveServersWithTags(), nil
}

// ResolveRead implements Navigator
func (n *Impl) ResolveRead(key string, op partition.Option) (map[string][]string, error) {
	if n.getState() != ready {
		return nil, ErrNotReady
	}
	return n.resolver.ResolveRead(key, op)
}

// ResolveWrite implements Navigator
func (n *Impl) ResolveWrite(key string, op partition.Option) (map[string][]string, error) {
	if n.getState() != ready {
		return nil, ErrNotReady
	}
	return n.resolver.ResolveWrite(key, op)
}

// destroyed returns
func (n *Impl) destroyed() bool {
	return n.getState() == destroyed
}

// getState gets the state of the current navigator instance.
func (n *Impl) getState() state {
	n.stateMutex.RLock()
	r := n.state
	n.stateMutex.RUnlock()
	return r
}

// setState sets the state of the current navigator instance.
func (n *Impl) setState(s state) {
	n.stateMutex.Lock()
	n.state = s
	n.stateMutex.Unlock()
}
