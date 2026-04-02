package experiment

import (
	"fmt"
	"strings"
	"time"

	"github.com/CyberArgonaut/makakito/pkg/schema"
)

// ValidationError aggregates multiple validation errors.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("experiment validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

func (e *ValidationError) add(msg string, args ...interface{}) {
	e.Errors = append(e.Errors, fmt.Sprintf(msg, args...))
}

func (e *ValidationError) hasErrors() bool {
	return len(e.Errors) > 0
}

var validTargetKinds = map[string]bool{
	"docker":     true,
	"kubernetes": true,
	"aws":        true,
	"ssh":        true,
	"process":    true,
}

var validFaultKinds = map[string]bool{
	"cpu":             true,
	"memory":          true,
	"container-stop":  true,
	"container-pause": true,
	"process-kill":    true,
	"network-latency": true,
	"packet-loss":     true,
	"http-error":      true,
}

var validProbeTypes = map[string]bool{
	"http":       true,
	"prometheus": true,
	"log":        true,
}

var validRollback = map[string]bool{
	"":       true,
	"auto":   true,
	"manual": true,
}

// Validate checks an experiment for correctness, applying defaults as needed.
// Returns a *ValidationError if any rules are violated.
func Validate(exp *schema.Experiment) error {
	ve := &ValidationError{}

	if exp.Version != "1" {
		ve.add("version must be \"1\", got %q", exp.Version)
	}
	if strings.TrimSpace(exp.Name) == "" {
		ve.add("name is required")
	}
	if len(exp.Hypothesis.Probes) == 0 {
		ve.add("hypothesis must have at least one probe")
	}
	if len(exp.Method) == 0 {
		ve.add("method must have at least one action")
	}

	for i, probe := range exp.Hypothesis.Probes {
		if !validProbeTypes[probe.Type] {
			ve.add("hypothesis.probes[%d]: unknown probe type %q (valid: http, prometheus, log)", i, probe.Type)
		}
		if probe.Type == "http" {
			if probe.URL == "" {
				ve.add("hypothesis.probes[%d]: http probe requires url", i)
			}
			if probe.ExpectedStatus == 0 {
				ve.add("hypothesis.probes[%d]: http probe requires expected_status", i)
			}
		}
		if probe.Timeout.Duration < 0 {
			ve.add("hypothesis.probes[%d]: timeout must not be negative", i)
		}
	}

	for i, action := range exp.Method {
		if !validTargetKinds[action.Target.Kind] {
			ve.add("method[%d] %q: unknown target kind %q (valid: docker, kubernetes, aws, ssh, process)", i, action.Name, action.Target.Kind)
		}
		if !validFaultKinds[action.Fault.Kind] {
			ve.add("method[%d] %q: unknown fault kind %q", i, action.Name, action.Fault.Kind)
		}
		if !validRollback[action.Rollback] {
			ve.add("method[%d] %q: rollback must be \"auto\" or \"manual\", got %q", i, action.Name, action.Rollback)
		}
		if action.Duration.Duration < 0 {
			ve.add("method[%d] %q: duration must not be negative", i, action.Name)
		}
	}

	// Apply default timeout.
	if exp.Controls.Timeout.Duration == 0 {
		exp.Controls.Timeout.Duration = 5 * time.Minute
	}

	if ve.hasErrors() {
		return ve
	}
	return nil
}
