package fault

import (
	"context"

	"github.com/docker/docker/api/types/container"
)

// DockerFaultAPI is the subset of the Docker client used by container faults.
type DockerFaultAPI interface {
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerPause(ctx context.Context, containerID string) error
	ContainerUnpause(ctx context.Context, containerID string) error
}
