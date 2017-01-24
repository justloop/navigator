package client

import (
	"context"
	"time"

	"github.com/justloop/navigator/partition"
	"github.com/justloop/navigator/utils"
)

const (
	// DefaultReplication is the default replicas
	DefaultReplication = 1
	// DefaultTimeout is the default IO timeout for the client
	DefaultTimeout = time.Second
)

// Config is the context related to Navigator Client
type Config struct {
	// Ctx is the cancellable context
	Ctx context.Context

	// ReplicaPoints is the number of replications, optional, default 1, means no replication
	ReplicaPoints int

	// ClusterRPCPort is any address of discovery node
	ClusterRPCAddr string

	// ClusterRPCPort is the port of cluster protocol running on
	ClusterRPCPort int

	// RingCacheTimeout is frequency to check for new cluster nodes, optional, default 1 second
	RingCacheTimeout time.Duration

	// Strategy is the partition strategy, optional, default IdentityStrategy
	Strategy partition.Strategy
}

// SetDefaultContext will set the default context
func SetDefaultContext(config *Config) *Config {
	// set default replication
	config.ReplicaPoints = utils.SelectInt(config.ReplicaPoints, DefaultReplication)
	// set default timeout
	config.RingCacheTimeout = utils.SelectDuration(config.RingCacheTimeout, DefaultTimeout)
	// set default strategyu
	if config.Strategy == nil {
		config.Strategy = partition.NewIndentityStrategy()
	}
	return config
}
