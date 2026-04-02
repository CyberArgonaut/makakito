package target

import (
	"fmt"

	"github.com/CyberArgonaut/makakito/internal/engine"
	dockerclient "github.com/docker/docker/client"
)

// RegisterAll registers all built-in targets into the given registry.
func RegisterAll(reg *engine.TargetRegistry) {
	reg.Register("docker", func() (engine.Target, error) {
		cli, err := dockerclient.NewClientWithOpts(
			dockerclient.FromEnv,
			dockerclient.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return nil, fmt.Errorf("creating docker client: %w", err)
		}
		return NewDockerTarget(cli), nil
	})
}
