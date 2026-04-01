// Package fault implements Phase 1 fault injectors for makakito.
package fault

import (
	"context"
	"fmt"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// ContainerPauseFault pauses a Docker container and unpauses it on rollback.
type ContainerPauseFault struct {
	docker DockerFaultAPI
}

// NewContainerPauseFault creates a ContainerPauseFault.
func NewContainerPauseFault(docker DockerFaultAPI) *ContainerPauseFault {
	return &ContainerPauseFault{docker: docker}
}

// Validate checks params — none required.
func (f *ContainerPauseFault) Validate(_ engine.Params) error {
	return nil
}

// Apply pauses the container and returns a rollback that unpauses it.
func (f *ContainerPauseFault) Apply(ctx context.Context, resource engine.Resource, _ engine.Params) (engine.RollbackFn, error) {
	rollback := engine.RollbackFn(func(ctx context.Context) error {
		if err := f.docker.ContainerUnpause(ctx, resource.ID); err != nil {
			return fmt.Errorf("unpausing container %q: %w", resource.Name, err)
		}
		return nil
	})

	if err := f.docker.ContainerPause(ctx, resource.ID); err != nil {
		return rollback, fmt.Errorf("pausing container %q: %w", resource.Name, err)
	}
	return rollback, nil
}
