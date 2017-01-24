/*
Package navigator is the entry point for the distributed framework, all the functions are wrapped up here.
*/
package navigator

import (
	"context"
	"time"

	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/justloop/navigator/discovery"
	"github.com/justloop/navigator/hashring"
	"github.com/justloop/navigator/partition"
	"github.com/justloop/navigator/seeds"
)

// logTag is the logging tag related to this navigator
var logTag = "navigator.service"

// default the cpu info is refreshed every 5 seconds
var defaultCPURefreshInterval = time.Second * 5

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

	// seedsService can be implemented by user for navigator use to get seeds
	seedsService seeds.Seeds

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
	navigator.ring = hashring.NewMapHashRing(config.ReplicaPoints)
	log.Info(logTag, "HashRing created...")

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
	if len(config.DConfig.Seed) == 0 {
		log.Info(logTag, "Seeds not provided, use seeds service instead...")
		navigator.seedsService.Start()
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
		log.Info(logTag, "No seeds provided, will use seeds service")
		seeds, err := n.seedsService.GetN(3)
		if err != nil {
			return err
		}
		if len(seeds) > 0 {
			_, err = n.dNode.Join(seeds, false)
			if err != nil {
				return err
			}
			log.Infof(logTag, "Successfully joined the cluster with %d nodes", n.dNode.NumNodes())
		} else {
			log.Info(logTag, "No seeds found in redis, join self as a single node cluster...")
			_, err = n.dNode.Join([]string{discovery.GetClusterAddrStr(n.config.DConfig)}, false)
			if err != nil {
				return err
			}
		}
	}
	n.startTime = time.Now()
	n.setState(ready)
	return nil
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
