package experiment

import (
	"strings"
	"testing"
	"time"

	"github.com/CyberArgonaut/makakito/pkg/schema"
)

func validExp() *schema.Experiment {
	return &schema.Experiment{
		Version: "1",
		Name:    "test experiment",
		Hypothesis: schema.Hypothesis{
			Title: "steady state",
			Probes: []schema.ProbeSpec{
				{Type: "http", URL: "http://localhost/health", ExpectedStatus: 200},
			},
		},
		Method: []schema.Action{
			{
				Type:     "fault",
				Name:     "stop redis",
				Target:   schema.TargetSpec{Kind: "docker"},
				Fault:    schema.FaultSpec{Kind: "container-stop"},
				Duration: schema.Duration{Duration: 30 * time.Second},
				Rollback: "auto",
			},
		},
	}
}

func TestValidateValid(t *testing.T) {
	if err := Validate(validExp()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDefaultTimeout(t *testing.T) {
	exp := validExp()
	if err := Validate(exp); err != nil {
		t.Fatal(err)
	}
	if exp.Controls.Timeout.Duration != 5*time.Minute {
		t.Errorf("default timeout = %v, want 5m", exp.Controls.Timeout.Duration)
	}
}

func TestValidateFromFiles(t *testing.T) {
	tests := []struct {
		file          string
		expectError   bool
		errorContains string
	}{
		{"testdata/valid_experiment.yaml", false, ""},
		{"testdata/invalid_no_name.yaml", true, "name is required"},
		{"testdata/invalid_no_probes.yaml", true, "at least one probe"},
		{"testdata/invalid_unknown_target.yaml", true, "unknown target kind"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			exp, err := Load(tt.file)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			err = Validate(exp)
			if tt.expectError && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errorContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			}
		})
	}
}

func TestValidateBadVersion(t *testing.T) {
	exp := validExp()
	exp.Version = "2"
	err := Validate(exp)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error %q does not mention version", err.Error())
	}
}

func TestValidateUnknownProbeType(t *testing.T) {
	exp := validExp()
	exp.Hypothesis.Probes[0].Type = "custom"
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "unknown probe type") {
		t.Errorf("expected unknown probe type error, got: %v", err)
	}
}

func TestValidateHTTPProbeRequiresURL(t *testing.T) {
	exp := validExp()
	exp.Hypothesis.Probes[0].URL = ""
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Errorf("expected url error, got: %v", err)
	}
}

func TestValidateHTTPProbeRequiresStatus(t *testing.T) {
	exp := validExp()
	exp.Hypothesis.Probes[0].ExpectedStatus = 0
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "expected_status") {
		t.Errorf("expected expected_status error, got: %v", err)
	}
}

func TestValidateUnknownFaultKind(t *testing.T) {
	exp := validExp()
	exp.Method[0].Fault.Kind = "unknown-fault"
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "unknown fault kind") {
		t.Errorf("expected unknown fault error, got: %v", err)
	}
}

func TestValidateInvalidRollback(t *testing.T) {
	exp := validExp()
	exp.Method[0].Rollback = "whenever"
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "rollback") {
		t.Errorf("expected rollback error, got: %v", err)
	}
}

func TestValidateNoActions(t *testing.T) {
	exp := validExp()
	exp.Method = nil
	err := Validate(exp)
	if err == nil || !strings.Contains(err.Error(), "at least one action") {
		t.Errorf("expected no actions error, got: %v", err)
	}
}
