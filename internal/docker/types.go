// Package docker provides a minimal Docker daemon client using only stdlib net/http.
// It replaces github.com/docker/docker to avoid upstream CVEs in that module.
package docker

// ContainerSummary is a minimal container representation returned by the Docker daemon.
type ContainerSummary struct {
	ID     string
	Names  []string
	Labels map[string]string
}

// Ping is returned by the Docker daemon health-check endpoint.
type Ping struct{}

// ListOptions controls which containers are listed.
type ListOptions struct {
	All bool
}

// StopOptions controls how a container is stopped.
type StopOptions struct{}

// StartOptions controls how a container is started.
type StartOptions struct{}
