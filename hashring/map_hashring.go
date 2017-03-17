package hashring

import (
	"sort"
	"strings"
	"sync"

	"github.com/dgryski/go-farm"
	mapring "github.com/justloop/hashring"
)

// MapHashRing is the hashring implementation using map
type MapHashRing struct {
	replication int
	ring        *mapring.HashRing
	sync.RWMutex
}

// NewMapHashRing will create a new MapHashRing instance
func NewMapHashRing(replication int) *MapHashRing {
	return &MapHashRing{
		ring:        mapring.New([]string{}),
		replication: replication,
	}
}

// NewMapHashRingWithKeyHash will create a new MapHashRing instance with customized key hash function
func NewMapHashRingWithKeyHash(replication int, keyHashFunc func(key string) uint32) *MapHashRing {
	return &MapHashRing{
		ring:        mapring.NewWithHash([]string{}, keyHashFunc),
		replication: replication,
	}
}

// AddNode will add one node to the ring, node format: [address:port].
func (r *MapHashRing) AddNode(node string) error {
	r.Lock()
	defer r.Unlock()
	r.ring = r.ring.AddNode(node)
	return nil
}

// RemoveNode will remove one node from the ring, node format: [address:port].
func (r *MapHashRing) RemoveNode(node string) error {
	r.Lock()
	defer r.Unlock()
	r.ring = r.ring.RemoveNode(node)
	return nil
}

// GetNodes return the list of nodes in the ring, not necessarily in order
func (r *MapHashRing) GetNodes() ([]string, error) {
	r.RLock()
	defer r.RUnlock()
	return r.ring.GetRingNodes(), nil
}

// GetNumNodes return the number of nodes in the ring
func (r *MapHashRing) GetNumNodes() (int, error) {
	r.RLock()
	defer r.RUnlock()
	return len(r.ring.GetRingNodes()), nil
}

// GetKeyNode return on any node this key is hashed on
func (r *MapHashRing) GetKeyNode(key string) (string, error) {
	r.RLock()
	defer r.RUnlock()
	if node, ok := r.ring.GetNode(key); ok {
		return node, nil
	}

	return "", ErrGetKeyNode
}

// GetKeyNodes returns the all the nodes this key is hashed on according to replication level
func (r *MapHashRing) GetKeyNodes(key string) ([]string, error) {
	r.RLock()
	defer r.RUnlock()
	if nodes, ok := r.ring.GetNodes(key, r.replication); ok {
		return nodes, nil
	}
	return []string{}, ErrGetKeyNodes
}

// Checksum not supported yet
func (r *MapHashRing) Checksum() (uint32, error) {
	r.RLock()
	defer r.RUnlock()
	nodes, err := r.GetNodes()
	if err != nil {
		return 0, err
	}
	sort.Strings(nodes)
	bytes := []byte(strings.Join(nodes, ";"))
	return farm.Fingerprint32(bytes), nil
}
