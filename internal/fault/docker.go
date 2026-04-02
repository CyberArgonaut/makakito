package fault

import (
	"context"

	"github.com/CyberArgonaut/makakito/internal/docker"
)

// DockerFaultAPI is the subset of the Docker client used by container faults.
type DockerFaultAPI interface {
	ContainerStop(ctx context.Context, containerID string, options docker.StopOptions) error
	ContainerStart(ctx context.Context, containerID string, options docker.StartOptions) error
	ContainerPause(ctx context.Context, containerID string) error
	ContainerUnpause(ctx context.Context, containerID string) error
}
