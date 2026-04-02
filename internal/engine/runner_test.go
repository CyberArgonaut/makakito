package engine

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CyberArgonaut/makakito/internal/store"
	"github.com/CyberArgonaut/makakito/pkg/schema"
)

// --- Test helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { s.Close() }) //nolint:errcheck // best-effort close in test cleanup
	return s
}

// simpleRunner builds a Runner with the http probe registered.
func simpleRunner(t *testing.T, _ string) *Runner {
	t.Helper()
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	// Register a no-op target that always returns one resource.
	targets.Register("docker", func() (Target, error) {
		return &fakeTarget{}, nil
	})
	// Register a no-op fault.
	faults.Register("container-stop", func() (Fault, error) {
		return &fakeFault{}, nil
	})
	// Register http probe.
	probes.Register("http", func() (Probe, error) {
		return &fakeProbe{success: true}, nil
	})

	return NewRunner(targets, faults, probes, testStore(t), testLogger())
}

func baseExperiment(serverURL string) *schema.Experiment {
	return &schema.Experiment{
		Version: "1",
		Name:    "test experiment",
		Hypothesis: schema.Hypothesis{
			Title: "steady state",
			Probes: []schema.ProbeSpec{
				{Type: "http", URL: serverURL + "/health", ExpectedStatus: 200},
			},
		},
		Method: []schema.Action{
			{
				Type:     "fault",
				Name:     "stop redis",
				Target:   schema.TargetSpec{Kind: "docker", Selector: map[string]string{"name": "redis"}},
				Fault:    schema.FaultSpec{Kind: "container-stop"},
				Duration: schema.Duration{Duration: 0}, // no wait in tests
				Rollback: "auto",
			},
		},
		Controls: schema.Controls{
			Timeout: schema.Duration{Duration: 30 * time.Second},
		},
	}
}

// --- Fake implementations ---

type fakeTarget struct {
	resources []Resource
}

func (f *fakeTarget) Resolve(_ context.Context, _ Selector) ([]Resource, error) {
	if f.resources != nil {
		return f.resources, nil
	}
	return []Resource{{ID: "abc", Name: "redis", Kind: "docker"}}, nil
}
func (f *fakeTarget) Validate(_ context.Context) error { return nil }

type fakeFault struct {
	applyErr    error
	rollbackErr error
	applied     bool
}

func (f *fakeFault) Apply(_ context.Context, _ Resource, _ Params) (RollbackFn, error) {
	f.applied = true
	rb := RollbackFn(func(_ context.Context) error { return f.rollbackErr })
	return rb, f.applyErr
}
func (f *fakeFault) Validate(_ Params) error { return nil }

type fakeProbe struct {
	success bool
}

func (f *fakeProbe) Check(_ context.Context, _ Params) (ProbeResult, error) {
	return ProbeResult{Success: f.success}, nil
}

// --- Tests ---

func TestRunnerHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	runner := simpleRunner(t, srv.URL)
	exp := baseExperiment(srv.URL)

	result, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != schema.StatusPassed {
		t.Errorf("status = %q, want passed", result.Status)
	}
	if result.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
}

func TestRunnerFailedBeforeProbe(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	ff := &fakeFault{}
	targets.Register("docker", func() (Target, error) { return &fakeTarget{}, nil })
	faults.Register("container-stop", func() (Fault, error) { return ff, nil })
	probes.Register("http", func() (Probe, error) { return &fakeProbe{success: false}, nil })

	runner := NewRunner(targets, faults, probes, testStore(t), testLogger())
	exp := baseExperiment("http://localhost")
	result, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != schema.StatusAborted {
		t.Errorf("status = %q, want aborted", result.Status)
	}
	if ff.applied {
		t.Error("fault should not have been applied when before-probe fails")
	}
}

func TestRunnerFailedAfterProbe(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	callCount := 0
	targets.Register("docker", func() (Target, error) { return &fakeTarget{}, nil })
	faults.Register("container-stop", func() (Fault, error) { return &fakeFault{}, nil })
	probes.Register("http", func() (Probe, error) {
		callCount++
		// First call (before) succeeds, second call (after) fails.
		return &fakeProbe{success: callCount <= 1}, nil
	})

	runner := NewRunner(targets, faults, probes, testStore(t), testLogger())
	exp := baseExperiment("http://localhost")
	result, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != schema.StatusFailed {
		t.Errorf("status = %q, want failed", result.Status)
	}
}

func TestRunnerTimeout(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	targets.Register("docker", func() (Target, error) { return &fakeTarget{}, nil })
	faults.Register("container-stop", func() (Fault, error) { return &fakeFault{}, nil })
	probes.Register("http", func() (Probe, error) { return &fakeProbe{success: true}, nil })

	runner := NewRunner(targets, faults, probes, testStore(t), testLogger())
	exp := baseExperiment("http://localhost")
	// Very short timeout.
	exp.Controls.Timeout = schema.Duration{Duration: 1 * time.Millisecond}
	// Long fault duration to trigger timeout.
	exp.Method[0].Duration = schema.Duration{Duration: 10 * time.Second}

	result, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Should be aborted or error due to timeout.
	if result.Status == schema.StatusPassed {
		t.Error("expected non-passed status due to timeout")
	}
}

func TestRunnerDryRunNeverAppliesFault(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	ff := &fakeFault{}
	targets.Register("docker", func() (Target, error) { return &fakeTarget{}, nil })
	faults.Register("container-stop", func() (Fault, error) { return ff, nil })
	probes.Register("http", func() (Probe, error) { return &fakeProbe{success: true}, nil })

	runner := NewRunner(targets, faults, probes, testStore(t), testLogger())
	exp := baseExperiment("http://localhost")
	result, err := runner.Run(context.Background(), exp, RunnerConfig{DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ff.applied {
		t.Error("fault must not be applied in dry-run mode")
	}
	if result.Status != schema.StatusPassed {
		t.Errorf("dry-run status = %q, want passed", result.Status)
	}
}

func TestRunnerBlastRadiusEnforcement(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	appliedCount := 0
	targets.Register("docker", func() (Target, error) {
		return &fakeTarget{resources: []Resource{
			{ID: "1", Name: "r1", Kind: "docker"},
			{ID: "2", Name: "r2", Kind: "docker"},
			{ID: "3", Name: "r3", Kind: "docker"},
		}}, nil
	})
	faults.Register("container-stop", func() (Fault, error) {
		return &countingFault{counter: &appliedCount}, nil
	})
	probes.Register("http", func() (Probe, error) { return &fakeProbe{success: true}, nil })

	runner := NewRunner(targets, faults, probes, testStore(t), testLogger())
	exp := baseExperiment("http://localhost")
	exp.Controls.BlastRadius = schema.BlastRadius{MaxTargets: 1}

	_, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if appliedCount > 1 {
		t.Errorf("blast radius violated: fault applied to %d resources, expected max 1", appliedCount)
	}
}

func TestRunnerEnvironmentRejection(t *testing.T) {
	runner := simpleRunner(t, "http://localhost")
	exp := baseExperiment("http://localhost")
	exp.Controls.Environments = []string{"staging"}

	_, err := runner.Run(context.Background(), exp, RunnerConfig{Environment: "production"})
	if err == nil {
		t.Fatal("expected error for disallowed environment")
	}
}

func TestRunnerStorePersistence(t *testing.T) {
	targets := NewTargetRegistry()
	faults := NewFaultRegistry()
	probes := NewProbeRegistry()

	targets.Register("docker", func() (Target, error) { return &fakeTarget{}, nil })
	faults.Register("container-stop", func() (Fault, error) { return &fakeFault{}, nil })
	probes.Register("http", func() (Probe, error) { return &fakeProbe{success: true}, nil })

	st := testStore(t)
	runner := NewRunner(targets, faults, probes, st, testLogger())
	exp := baseExperiment("http://localhost")

	result, err := runner.Run(context.Background(), exp, RunnerConfig{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	stored, err := st.Get(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Status != schema.StatusPassed {
		t.Errorf("stored status = %q, want passed", stored.Status)
	}
}

type countingFault struct {
	counter *int
}

func (f *countingFault) Apply(_ context.Context, _ Resource, _ Params) (RollbackFn, error) {
	*f.counter++
	return NoopRollback(), nil
}
func (f *countingFault) Validate(_ Params) error { return nil }
