package partition

// Strategy is the strategy to map a key to its partition,
// one key might be able to be map to multiple partitions
type Strategy interface {
	// GetName will return the name of this strategy
	GetName() string

	// GetReadPartitions will return the list of partition Id this key is mapped to read from
	GetReadPartitions(key string, op Option) ([]string, error)

	// GetWritePartitions will return the list of partition Ids this key is mapped to write to
	GetWritePartitions(key string, op Option) ([]string, error)
}

// Option is the option to get the partition key
type Option interface{}
