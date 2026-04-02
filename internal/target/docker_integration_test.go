//go:build integration

package target

import (
	"context"
	"testing"

	"github.com/CyberArgonaut/makakito/internal/docker"
	"github.com/CyberArgonaut/makakito/internal/engine"
)

// TestDockerTargetIntegration tests with a real Docker daemon.
// Requires Docker to be running. Run with: go test -tags=integration ./internal/target/...
func TestDockerTargetIntegration(t *testing.T) {
	cli, err := docker.NewClientFromEnv()
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer cli.Close()

	tgt := NewDockerTarget(cli)
	ctx := context.Background()

	// Validate: ping daemon.
	if err := tgt.Validate(ctx); err != nil {
		t.Skipf("docker daemon unreachable: %v", err)
	}

	// Resolve with empty selector — should list all running containers.
	resources, err := tgt.Resolve(ctx, engine.Selector{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	t.Logf("found %d running containers", len(resources))

	for _, r := range resources {
		t.Logf("  container: id=%s name=%s", r.ID[:12], r.Name)
	}
}
