package partition

// IndentityStrategy use the key itself as the partition key
type IndentityStrategy struct{}

// NewIndentityStrategy will create a new IndentityStrategy instance
func NewIndentityStrategy() *IndentityStrategy {
	return &IndentityStrategy{}
}

// GetName will return the name of this strategy
func (s *IndentityStrategy) GetName() string {
	return "IndentityStrategy"
}

// GetReadPartitions will return the list of partition Id this key is mapped to read from
func (s *IndentityStrategy) GetReadPartitions(key string, op Option) ([]string, error) {
	return []string{key}, nil
}

// GetWritePartitions will return the list of partition Ids this key is mapped to write to
func (s *IndentityStrategy) GetWritePartitions(key string, op Option) ([]string, error) {
	return []string{key}, nil
}
