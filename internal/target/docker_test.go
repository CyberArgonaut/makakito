package target

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberArgonaut/makakito/internal/docker"
	"github.com/CyberArgonaut/makakito/internal/engine"
)

type mockDockerAPI struct {
	containers []docker.ContainerSummary
	pingErr    error
	listErr    error
}

func (m *mockDockerAPI) ContainerList(_ context.Context, _ docker.ListOptions) ([]docker.ContainerSummary, error) {
	return m.containers, m.listErr
}

func (m *mockDockerAPI) Ping(_ context.Context) (docker.Ping, error) {
	return docker.Ping{}, m.pingErr
}

func TestDockerTargetResolveByName(t *testing.T) {
	mock := &mockDockerAPI{
		containers: []docker.ContainerSummary{
			{ID: "abc123", Names: []string{"/redis"}, Labels: map[string]string{}},
			{ID: "def456", Names: []string{"/postgres"}, Labels: map[string]string{}},
		},
	}
	tgt := NewDockerTarget(mock)
	resources, err := tgt.Resolve(context.Background(), engine.Selector{"name": "redis"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len = %d, want 1", len(resources))
	}
	if resources[0].Name != "redis" {
		t.Errorf("name = %q, want redis", resources[0].Name)
	}
}

func TestDockerTargetResolveByLabel(t *testing.T) {
	mock := &mockDockerAPI{
		containers: []docker.ContainerSummary{
			{ID: "abc", Names: []string{"/app"}, Labels: map[string]string{"env": "prod", "team": "platform"}},
			{ID: "def", Names: []string{"/worker"}, Labels: map[string]string{"env": "staging"}},
		},
	}
	tgt := NewDockerTarget(mock)
	resources, err := tgt.Resolve(context.Background(), engine.Selector{"label": "env=prod"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("len = %d, want 1", len(resources))
	}
}

func TestDockerTargetResolveByID(t *testing.T) {
	mock := &mockDockerAPI{
		containers: []docker.ContainerSummary{
			{ID: "abc123def", Names: []string{"/app"}, Labels: map[string]string{}},
			{ID: "xyz789", Names: []string{"/worker"}, Labels: map[string]string{}},
		},
	}
	tgt := NewDockerTarget(mock)
	resources, err := tgt.Resolve(context.Background(), engine.Selector{"id": "abc123"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("len = %d, want 1", len(resources))
	}
}

func TestDockerTargetResolveEmpty(t *testing.T) {
	mock := &mockDockerAPI{
		containers: []docker.ContainerSummary{
			{ID: "abc", Names: []string{"/postgres"}, Labels: map[string]string{}},
		},
	}
	tgt := NewDockerTarget(mock)
	resources, err := tgt.Resolve(context.Background(), engine.Selector{"name": "redis"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected empty, got %d results", len(resources))
	}
}

func TestDockerTargetValidateSuccess(t *testing.T) {
	mock := &mockDockerAPI{}
	tgt := NewDockerTarget(mock)
	if err := tgt.Validate(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDockerTargetValidateFailure(t *testing.T) {
	mock := &mockDockerAPI{pingErr: errors.New("connection refused")}
	tgt := NewDockerTarget(mock)
	if err := tgt.Validate(context.Background()); err == nil {
		t.Error("expected error for failed ping")
	}
}
