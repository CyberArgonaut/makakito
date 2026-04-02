package schema

// Experiment is the top-level YAML experiment definition.
type Experiment struct {
	Version     string            `yaml:"version" json:"version"`
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Hypothesis  Hypothesis        `yaml:"hypothesis" json:"hypothesis"`
	Method      []Action          `yaml:"method" json:"method"`
	Controls    Controls          `yaml:"controls,omitempty" json:"controls,omitempty"`
}

// Hypothesis defines the steady-state checked before and after the experiment.
type Hypothesis struct {
	Title  string      `yaml:"title" json:"title"`
	Probes []ProbeSpec `yaml:"probes" json:"probes"`
}

// ProbeSpec defines a single steady-state probe.
type ProbeSpec struct {
	Type           string   `yaml:"type" json:"type"`
	URL            string   `yaml:"url,omitempty" json:"url,omitempty"`
	ExpectedStatus int      `yaml:"expected_status,omitempty" json:"expected_status,omitempty"`
	Timeout        Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Query          string   `yaml:"query,omitempty" json:"query,omitempty"`
	Endpoint       string   `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Pattern        string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Source         string   `yaml:"source,omitempty" json:"source,omitempty"`
	MaxMatches     *int     `yaml:"max_matches,omitempty" json:"max_matches,omitempty"`
}

// Action represents a single fault injection step.
type Action struct {
	Type     string     `yaml:"type" json:"type"`
	Name     string     `yaml:"name" json:"name"`
	Target   TargetSpec `yaml:"target" json:"target"`
	Fault    FaultSpec  `yaml:"fault" json:"fault"`
	Duration Duration   `yaml:"duration" json:"duration"`
	Rollback string     `yaml:"rollback,omitempty" json:"rollback,omitempty"`
}

// TargetSpec identifies an infrastructure target.
type TargetSpec struct {
	Kind     string            `yaml:"kind" json:"kind"`
	Selector map[string]string `yaml:"selector,omitempty" json:"selector,omitempty"`
}

// FaultSpec describes the fault to inject.
type FaultSpec struct {
	Kind   string            `yaml:"kind" json:"kind"`
	Params map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
}

// Controls defines safety boundaries for the experiment.
type Controls struct {
	BlastRadius     BlastRadius `yaml:"blast_radius,omitempty" json:"blast_radius,omitempty"`
	Timeout         Duration    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Environments    []string    `yaml:"environments,omitempty" json:"environments,omitempty"`
	RequireApproval bool        `yaml:"require_approval,omitempty" json:"require_approval,omitempty"`
}

// BlastRadius defines the maximum scope of an experiment.
type BlastRadius struct {
	MaxTargets int `yaml:"max_targets,omitempty" json:"max_targets,omitempty"`
	Percentage int `yaml:"percentage,omitempty" json:"percentage,omitempty"`
}
