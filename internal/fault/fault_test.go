package fault

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CyberArgonaut/makakito/internal/engine"
	"github.com/docker/docker/api/types/container"
)

// --- Mock Docker API ---

type mockDockerFaultAPI struct {
	stopErr       error
	startErr      error
	pauseErr      error
	unpauseErr    error
	stopCalled    bool
	startCalled   bool
	pauseCalled   bool
	unpauseCalled bool
}

func (m *mockDockerFaultAPI) ContainerStop(_ context.Context, _ string, _ container.StopOptions) error {
	m.stopCalled = true
	return m.stopErr
}
func (m *mockDockerFaultAPI) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	m.startCalled = true
	return m.startErr
}
func (m *mockDockerFaultAPI) ContainerPause(_ context.Context, _ string) error {
	m.pauseCalled = true
	return m.pauseErr
}
func (m *mockDockerFaultAPI) ContainerUnpause(_ context.Context, _ string) error {
	m.unpauseCalled = true
	return m.unpauseErr
}

// --- Mock Network Driver ---

type mockNetworkDriver struct {
	addLatencyErr       error
	addPacketLossErr    error
	removeErr           error
	addLatencyCalled    bool
	addPacketLossCalled bool
	removeCalled        bool
}

func (m *mockNetworkDriver) AddLatency(_ context.Context, _ string, _, _ time.Duration) error {
	m.addLatencyCalled = true
	return m.addLatencyErr
}
func (m *mockNetworkDriver) AddPacketLoss(_ context.Context, _ string, _ float64) error {
	m.addPacketLossCalled = true
	return m.addPacketLossErr
}
func (m *mockNetworkDriver) Remove(_ context.Context, _ string) error {
	m.removeCalled = true
	return m.removeErr
}

// --- Test helpers ---

func testResource() engine.Resource {
	return engine.Resource{ID: "abc123", Name: "test-container", Kind: "docker"}
}

func assertNonNilRollback(t *testing.T, fn engine.RollbackFn, label string) {
	t.Helper()
	if fn == nil {
		t.Errorf("%s: rollback must never be nil", label)
	}
}

// --- ContainerStop ---

func TestContainerStopApplyAndRollback(t *testing.T) {
	mock := &mockDockerFaultAPI{}
	f := NewContainerStopFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "ContainerStop")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !mock.stopCalled {
		t.Error("ContainerStop not called")
	}

	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if !mock.startCalled {
		t.Error("ContainerStart not called on rollback")
	}
}

func TestContainerStopApplyErrorStillReturnsRollback(t *testing.T) {
	mock := &mockDockerFaultAPI{stopErr: errors.New("docker down")}
	f := NewContainerStopFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "ContainerStop error case")
	if err == nil {
		t.Error("expected error")
	}
}

// --- ContainerPause ---

func TestContainerPauseApplyAndRollback(t *testing.T) {
	mock := &mockDockerFaultAPI{}
	f := NewContainerPauseFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "ContainerPause")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if !mock.unpauseCalled {
		t.Error("ContainerUnpause not called on rollback")
	}
}

func TestContainerPauseApplyErrorStillReturnsRollback(t *testing.T) {
	mock := &mockDockerFaultAPI{pauseErr: errors.New("pause failed")}
	f := NewContainerPauseFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "ContainerPause error case")
	if err == nil {
		t.Error("expected error")
	}
}

// --- CPU ---

func TestCPUFaultApplyAndRollback(t *testing.T) {
	f := &CPUFault{}
	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{"cores": "1"})
	assertNonNilRollback(t, rollback, "CPUFault")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Give goroutines a moment to spin
	time.Sleep(10 * time.Millisecond)

	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestCPUFaultValidation(t *testing.T) {
	f := &CPUFault{}
	if err := f.Validate(engine.Params{"cores": "abc"}); err == nil {
		t.Error("expected error for invalid cores")
	}
	if err := f.Validate(engine.Params{"cores": "0"}); err == nil {
		t.Error("expected error for zero cores")
	}
	if err := f.Validate(engine.Params{}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Memory ---

func TestMemoryFaultApplyAndRollback(t *testing.T) {
	f := &MemoryFault{}
	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{"size": "1MB"})
	assertNonNilRollback(t, rollback, "MemoryFault")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestMemoryFaultInvalidSize(t *testing.T) {
	f := &MemoryFault{}
	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{"size": "invalid"})
	assertNonNilRollback(t, rollback, "MemoryFault invalid size")
	if err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestMemoryFaultMissingSize(t *testing.T) {
	f := &MemoryFault{}
	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "MemoryFault missing size")
	if err == nil {
		t.Error("expected error for missing size")
	}
}

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		isErr bool
	}{
		{"100MB", 100 * 1024 * 1024, false},
		{"1GB", 1 * 1024 * 1024 * 1024, false},
		{"512KB", 512 * 1024, false},
		{"1024B", 1024, false},
		{"100", 100 * 1024 * 1024, false},
		{"0MB", 0, true},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMemorySize(tt.input)
			if tt.isErr && err == nil {
				t.Error("expected error")
			}
			if !tt.isErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.isErr && got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// --- ProcessKill ---

func TestProcessKillInvalidPID(t *testing.T) {
	f := &ProcessKillFault{}
	rollback, err := f.Apply(context.Background(), engine.Resource{ID: "notanumber"}, engine.Params{})
	assertNonNilRollback(t, rollback, "ProcessKill invalid PID")
	if err == nil {
		t.Error("expected error for invalid PID")
	}
}

func TestProcessKillNoopRollback(t *testing.T) {
	f := &ProcessKillFault{}
	// Use PID 0 which is invalid on Linux, ensuring an error without killing anything
	rollback, _ := f.Apply(context.Background(), engine.Resource{ID: "0"}, engine.Params{})
	assertNonNilRollback(t, rollback, "ProcessKill noop")
}

// --- NetworkLatency ---

func TestNetworkLatencyFaultApplyAndRollback(t *testing.T) {
	mock := &mockNetworkDriver{}
	f := NewNetworkLatencyFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{
		"interface": "eth0",
		"latency":   "200ms",
		"jitter":    "50ms",
	})
	assertNonNilRollback(t, rollback, "NetworkLatency")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !mock.addLatencyCalled {
		t.Error("AddLatency not called")
	}

	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if !mock.removeCalled {
		t.Error("Remove not called on rollback")
	}
}

func TestNetworkLatencyFaultErrorStillReturnsRollback(t *testing.T) {
	mock := &mockNetworkDriver{addLatencyErr: errors.New("no root")}
	f := NewNetworkLatencyFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{"latency": "100ms"})
	assertNonNilRollback(t, rollback, "NetworkLatency error case")
	if err == nil {
		t.Error("expected error")
	}
}

func TestNetworkLatencyMissingParam(t *testing.T) {
	mock := &mockNetworkDriver{}
	f := NewNetworkLatencyFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{})
	assertNonNilRollback(t, rollback, "NetworkLatency missing param")
	if err == nil {
		t.Error("expected error for missing latency param")
	}
}

// --- PacketLoss ---

func TestPacketLossFaultApplyAndRollback(t *testing.T) {
	mock := &mockNetworkDriver{}
	f := NewPacketLossFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{
		"interface": "eth0",
		"percent":   "10",
	})
	assertNonNilRollback(t, rollback, "PacketLoss")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !mock.addPacketLossCalled {
		t.Error("AddPacketLoss not called")
	}

	if err := rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestPacketLossFaultValidation(t *testing.T) {
	mock := &mockNetworkDriver{}
	f := NewPacketLossFault(mock)

	if err := f.Validate(engine.Params{}); err == nil {
		t.Error("expected error for missing percent")
	}
	if err := f.Validate(engine.Params{"percent": "150"}); err == nil {
		t.Error("expected error for percent > 100")
	}
	if err := f.Validate(engine.Params{"percent": "10"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPacketLossFaultErrorStillReturnsRollback(t *testing.T) {
	mock := &mockNetworkDriver{addPacketLossErr: errors.New("no root")}
	f := NewPacketLossFault(mock)

	rollback, err := f.Apply(context.Background(), testResource(), engine.Params{"percent": "10"})
	assertNonNilRollback(t, rollback, "PacketLoss error case")
	if err == nil {
		t.Error("expected error")
	}
}

// Ensure unused import doesn't break compile
var _ = fmt.Sprintf
