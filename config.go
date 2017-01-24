package navigator

import (
	"github.com/justloop/navigator/discovery"
	"github.com/justloop/navigator/partition"
	"github.com/justloop/navigator/seeds"
	"github.com/justloop/navigator/utils"
)

const (
	// defaultReplication is the default replication points
	defaultReplication = 1
)

// Config is the configuration related to Navigator
type Config struct {

	// ReplicaPoints is the number of replications, optional, default 1
	ReplicaPoints int

	// DConfig is the configuration for discovery client, optional, default use default discovery config
	DConfig *discovery.Config

	// Strategy is the partition strategy, optional, default IdentityStrategy
	Strategy partition.Strategy

	// SeedsService is a implementation of seedsService that used to auto get a list of seeds to join
	SeedsService seeds.Seeds

	// OnMemberEvent is the callback handler to be invoked when repartition happens,
	// and the node will need to reload data, optional, default, no rehashHandler
	OnMemberEvent func(event discovery.MemberEvent) error

	// OnShutdown is the callback when discovery is shutdown or reaped by the cluster
	OnShutdown func()
}

// setDefaultConfig sets the default context
func setDefaultConfig(config *Config) *Config {
	// Set default discovery config
	if config.DConfig == nil {
		config.DConfig = discovery.DefaultConfig()
	}
	// Set default partition strategy
	if config.Strategy == nil {
		config.Strategy = partition.NewIndentityStrategy()
	}
	if config.SeedsService == nil {

	}

	// Set default replication
	config.ReplicaPoints = utils.SelectInt(config.ReplicaPoints, defaultReplication)
	// set default rehashHandler
	if config.OnMemberEvent == nil {
		config.OnMemberEvent = func(event discovery.MemberEvent) error { return nil }
	}
	// set default shutdown
	if config.OnShutdown == nil {
		config.OnShutdown = func() {}
	}
	return config
}
