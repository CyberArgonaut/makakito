package engine

import (
	"context"
	"sync"
	"testing"
)

type mockTarget struct{}

func (m *mockTarget) Resolve(_ context.Context, _ Selector) ([]Resource, error) {
	return nil, nil
}
func (m *mockTarget) Validate(_ context.Context) error { return nil }

type mockFault struct{}

func (m *mockFault) Apply(_ context.Context, _ Resource, _ Params) (RollbackFn, error) {
	return NoopRollback(), nil
}
func (m *mockFault) Validate(_ Params) error { return nil }

type mockProbe struct{}

func (m *mockProbe) Check(_ context.Context, _ Params) (ProbeResult, error) {
	return ProbeResult{Success: true}, nil
}

func TestTargetRegistryRegisterAndGet(t *testing.T) {
	reg := NewTargetRegistry()
	reg.Register("docker", func() (Target, error) { return &mockTarget{}, nil })

	target, err := reg.Get("docker")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if target == nil {
		t.Fatal("expected non-nil target")
	}
}

func TestTargetRegistryUnknownKind(t *testing.T) {
	reg := NewTargetRegistry()
	_, err := reg.Get("unknown")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestFaultRegistryRegisterAndGet(t *testing.T) {
	reg := NewFaultRegistry()
	reg.Register("cpu", func() (Fault, error) { return &mockFault{}, nil })

	fault, err := reg.Get("cpu")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fault == nil {
		t.Fatal("expected non-nil fault")
	}
}

func TestProbeRegistryRegisterAndGet(t *testing.T) {
	reg := NewProbeRegistry()
	reg.Register("http", func() (Probe, error) { return &mockProbe{}, nil })

	probe, err := reg.Get("http")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if probe == nil {
		t.Fatal("expected non-nil probe")
	}
}

func TestRegistryConcurrentAccess(_ *testing.T) {
	reg := NewTargetRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Register("docker", func() (Target, error) { return &mockTarget{}, nil })
		}()
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Get("docker") //nolint:errcheck // return value intentionally discarded in concurrency stress test
		}()
	}
	wg.Wait()
}
