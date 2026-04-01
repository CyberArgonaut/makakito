package fault

import (
	"context"
	"fmt"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// ContainerStopFault stops a Docker container and restarts it on rollback.
type ContainerStopFault struct {
	docker DockerFaultAPI
}

// NewContainerStopFault creates a ContainerStopFault.
func NewContainerStopFault(docker DockerFaultAPI) *ContainerStopFault {
	return &ContainerStopFault{docker: docker}
}

// Validate checks params — none required.
func (f *ContainerStopFault) Validate(_ engine.Params) error {
	return nil
}

// Apply stops the container and returns a rollback that starts it again.
func (f *ContainerStopFault) Apply(ctx context.Context, resource engine.Resource, _ engine.Params) (engine.RollbackFn, error) {
	rollback := engine.RollbackFn(func(ctx context.Context) error {
		if err := f.docker.ContainerStart(ctx, resource.ID, containerStartOptions()); err != nil {
			return fmt.Errorf("starting container %q: %w", resource.Name, err)
		}
		return nil
	})

	if err := f.docker.ContainerStop(ctx, resource.ID, containerStopOptions()); err != nil {
		return rollback, fmt.Errorf("stopping container %q: %w", resource.Name, err)
	}
	return rollback, nil
}
