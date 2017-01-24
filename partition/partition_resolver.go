package partition

import (
	log "github.com/Sirupsen/logrus"
	"github.com/justloop/navigator/hashring"
)

// Resolver resolves a key to corresponding servers
type Resolver interface {
	// ResolveRead will resolve to a list of servers and partition id that this key is resolved to read from,
	// if any server read trail failed, proceed to next server in the list to read
	// return a map from shardId to servers
	ResolveRead(key string, op Option) (map[string][]string, error)

	// ResolveWrite will resolve to a list of servers and partition id that this key is resolved to write to,
	// the writer will have to be responsible to write to all the servers, but ignore any fails
	// return a map from shardId to servers
	ResolveWrite(key string, op Option) (map[string][]string, error)
}

// logTagResolver is logging tag for RingResolver
var logTagResolver = "navigator.partition"

// RingResolver implements the PartitionResolver, but use ring to do the partition
type RingResolver struct {
	// Strategy is the partition strategy to be used
	Strategy Strategy
	// Ring is the actual HashRing instance
	Ring hashring.HashRing
}

// NewRingResolver will initialize a RingResolver from a ring
func NewRingResolver(ring hashring.HashRing, strategy Strategy) *RingResolver {
	return &RingResolver{
		Ring:     ring,
		Strategy: strategy,
	}
}

func (partitioner *RingResolver) resolve(key string, op Option, isRead bool) (map[string][]string, error) {
	result := make(map[string][]string)
	var (
		partitionIDs []string
		err          error
	)
	if isRead {
		partitionIDs, err = partitioner.Strategy.GetReadPartitions(key, op)
	} else {
		partitionIDs, err = partitioner.Strategy.GetWritePartitions(key, op)
	}

	// for each partition return find the servers
	for _, partitionID := range partitionIDs {
		servers, err1 := partitioner.Ring.GetKeyNodes(partitionID)
		if err1 != nil {
			err = err1
			log.Warnf(logTagResolver, "Ring.GetKeyNodes return error", err)
		}

		for _, server := range servers {
			if _, ok := result[server]; !ok {
				result[server] = []string{partitionID}
			} else {
				result[server] = append(result[server], partitionID)
			}
		}
	}

	return result, err
}

// ResolveRead will resolve to a list of servers that this key is resolved to read from,
// if any server read trail failed, proceed to next server in the list to read
// Even return error may contain result, because some intermediate step might have error
func (partitioner *RingResolver) ResolveRead(key string, op Option) (map[string][]string, error) {
	return partitioner.resolve(key, op, true)
}

// ResolveWrite will resolve to a list of servers that this key is resolved to write to,
// the writer will have to be responsible to write to all the servers, but ignore any fails
// Even return error may contain result, because some intermediate step might have error
func (partitioner *RingResolver) ResolveWrite(key string, op Option) (map[string][]string, error) {
	return partitioner.resolve(key, op, false)
}
