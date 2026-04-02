// Package target implements infrastructure target drivers for makakito.
package target

import (
	"context"
	"fmt"
	"strings"

	"github.com/CyberArgonaut/makakito/internal/docker"
	"github.com/CyberArgonaut/makakito/internal/engine"
)

// DockerAPI is the subset of the Docker client used by DockerTarget.
type DockerAPI interface {
	ContainerList(ctx context.Context, options docker.ListOptions) ([]docker.ContainerSummary, error)
	Ping(ctx context.Context) (docker.Ping, error)
}

// DockerTarget resolves Docker containers as engine resources.
type DockerTarget struct {
	client DockerAPI
}

// NewDockerTarget creates a DockerTarget backed by the given client.
func NewDockerTarget(client DockerAPI) *DockerTarget {
	return &DockerTarget{client: client}
}

// Resolve returns running containers matching the selector.
// Selector keys: "name", "label", "id".
func (t *DockerTarget) Resolve(ctx context.Context, sel engine.Selector) ([]engine.Resource, error) {
	opts := docker.ListOptions{All: false}

	containers, err := t.client.ContainerList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing docker containers: %w", err)
	}

	var resources []engine.Resource
	for i := range containers {
		if !matchesSelector(&containers[i], sel) {
			continue
		}
		name := ""
		if len(containers[i].Names) > 0 {
			name = strings.TrimPrefix(containers[i].Names[0], "/")
		}
		resources = append(resources, engine.Resource{
			ID:     containers[i].ID,
			Name:   name,
			Kind:   "docker",
			Labels: containers[i].Labels,
		})
	}
	return resources, nil
}

// Validate checks that the Docker daemon is reachable.
func (t *DockerTarget) Validate(ctx context.Context) error {
	if _, err := t.client.Ping(ctx); err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}
	return nil
}

func matchesSelector(c *docker.ContainerSummary, sel engine.Selector) bool {
	if name, ok := sel["name"]; ok {
		matched := false
		for _, n := range c.Names {
			if strings.Contains(strings.TrimPrefix(n, "/"), name) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if id, ok := sel["id"]; ok {
		if !strings.HasPrefix(c.ID, id) {
			return false
		}
	}

	if label, ok := sel["label"]; ok {
		parts := strings.SplitN(label, "=", 2)
		key := parts[0]
		if len(parts) == 2 {
			val, exists := c.Labels[key]
			if !exists || val != parts[1] {
				return false
			}
		} else {
			if _, exists := c.Labels[key]; !exists {
				return false
			}
		}
	}

	return true
}
