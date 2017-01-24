package seeds

// Seeds provides a service to get a list of seeds
type Seeds interface {
	// GetN will return a list of n alive seeds
	GetN(n int) ([]string, error)

	// Start will start the current seed service, constantly refresh the seeds
	Start()
}
