package schema

import "time"

// ExperimentStatus represents the outcome of an experiment.
type ExperimentStatus string

// Experiment status values.
const (
	StatusRunning ExperimentStatus = "running"
	StatusPassed  ExperimentStatus = "passed"
	StatusFailed  ExperimentStatus = "failed"
	StatusAborted ExperimentStatus = "aborted"
	StatusError   ExperimentStatus = "error"
)

// ExperimentResult captures the full outcome of an experiment run.
type ExperimentResult struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Status       ExperimentStatus `json:"status"`
	StartedAt    time.Time        `json:"started_at"`
	FinishedAt   *time.Time       `json:"finished_at,omitempty"`
	ProbeResults []ProbeResult    `json:"probe_results,omitempty"`
	ActionLogs   []ActionLog      `json:"action_logs,omitempty"`
	Error        string           `json:"error,omitempty"`
}

// ProbePhase indicates when a probe was checked.
type ProbePhase string

// Probe phase values.
const (
	ProbePhaseBefore ProbePhase = "before"
	ProbePhaseAfter  ProbePhase = "after"
)

// ProbeResult records the outcome of a single probe check.
type ProbeResult struct {
	ProbeType string     `json:"probe_type"`
	Success   bool       `json:"success"`
	Message   string     `json:"message,omitempty"`
	CheckedAt time.Time  `json:"checked_at"`
	Phase     ProbePhase `json:"phase"`
}

// ActionLog records what happened during a fault action.
type ActionLog struct {
	ActionName string   `json:"action_name"`
	TargetKind string   `json:"target_kind"`
	FaultKind  string   `json:"fault_kind"`
	Resources  []string `json:"resources,omitempty"`
	RolledBack bool     `json:"rolled_back"`
	Error      string   `json:"error,omitempty"`
}
