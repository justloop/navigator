/*
Package hashring is a pure hashring used for consistent hashing, will be used by navigator node and client

Two implementations: map hashring and red black tree hashring,

Currently using red black tree implementation because it's more efficient.
*/
package hashring

// The HashRing using consistent hashing with replication mechanism buildin
type HashRing interface {

	// Checksum will return the checksum of current ring, using farmhash
	Checksum() (uint32, error)

	// AddNode will add one node to the ring, node format: [address:port].
	AddNode(node string) error

	// RemoveNode will remove one node from the ring, node format: [address:port].
	RemoveNode(node string) error

	// GetNodes return the list of nodes in the ring, not necessarily in order
	GetNodes() ([]string, error)

	// GetNumNodes return the number of nodes in the ring
	GetNumNodes() (int, error)

	// GetKeyNode return on any node this key is hashed on
	GetKeyNode(key string) (string, error)

	// GetKeyNodes returns the all the nodes this key is hashed on according to replication level
	GetKeyNodes(key string) ([]string, error)
}
