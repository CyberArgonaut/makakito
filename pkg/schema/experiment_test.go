package schema

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestExperimentYAMLRoundTrip(t *testing.T) {
	exp := Experiment{
		Version:     "1",
		Name:        "test experiment",
		Description: "a test",
		Labels:      map[string]string{"env": "staging", "phase": "1"},
		Hypothesis: Hypothesis{
			Title: "service stays healthy",
			Probes: []ProbeSpec{
				{Type: "http", URL: "http://localhost:8080/healthz", ExpectedStatus: 200},
			},
		},
		Method: []Action{
			{
				Type: "fault",
				Name: "stop redis",
				Target: TargetSpec{
					Kind:     "docker",
					Selector: map[string]string{"name": "redis"},
				},
				Fault:    FaultSpec{Kind: "container-stop"},
				Duration: Duration{60 * time.Second},
				Rollback: "auto",
			},
		},
		Controls: Controls{
			BlastRadius:  BlastRadius{MaxTargets: 1},
			Timeout:      Duration{5 * time.Minute},
			Environments: []string{"staging"},
		},
	}

	data, err := yaml.Marshal(exp)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}

	var got Experiment
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}

	if got.Name != exp.Name {
		t.Errorf("name = %q, want %q", got.Name, exp.Name)
	}
	if got.Version != "1" {
		t.Errorf("version = %q, want %q", got.Version, "1")
	}
	if len(got.Method) != 1 {
		t.Fatalf("method len = %d, want 1", len(got.Method))
	}
	if got.Method[0].Duration.Duration != 60*time.Second {
		t.Errorf("duration = %v, want 60s", got.Method[0].Duration)
	}
	if got.Controls.Timeout.Duration != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", got.Controls.Timeout)
	}
}

func TestExperimentJSONRoundTrip(t *testing.T) {
	exp := Experiment{
		Version: "1",
		Name:    "json test",
		Hypothesis: Hypothesis{
			Title: "steady state",
			Probes: []ProbeSpec{
				{Type: "http", URL: "http://localhost/health", ExpectedStatus: 200},
			},
		},
		Method: []Action{
			{
				Type:     "fault",
				Name:     "kill process",
				Target:   TargetSpec{Kind: "process"},
				Fault:    FaultSpec{Kind: "process-kill", Params: map[string]string{"signal": "SIGTERM"}},
				Duration: Duration{30 * time.Second},
			},
		},
	}

	data, err := json.Marshal(exp)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	var got Experiment
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if got.Name != exp.Name {
		t.Errorf("name = %q, want %q", got.Name, exp.Name)
	}
	if got.Method[0].Duration.Duration != 30*time.Second {
		t.Errorf("duration = %v, want 30s", got.Method[0].Duration)
	}
}
