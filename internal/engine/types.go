package engine

import "context"

// Selector identifies resources to target.
type Selector map[string]string

// Resource represents an infrastructure resource that can be acted upon.
type Resource struct {
	ID     string
	Name   string
	Kind   string
	Labels map[string]string
}

// Params holds fault or probe configuration parameters.
type Params map[string]string

// RollbackFn undoes the effect of a fault. Must never be nil.
type RollbackFn func(ctx context.Context) error

// NoopRollback returns a RollbackFn that does nothing.
func NoopRollback() RollbackFn {
	return func(_ context.Context) error { return nil }
}

// ProbeResult captures the outcome of a runtime probe check.
type ProbeResult struct {
	Success bool
	Message string
}

// Target represents an infrastructure resource to act on.
type Target interface {
	Resolve(ctx context.Context, selector Selector) ([]Resource, error)
	Validate(ctx context.Context) error
}

// Fault is an injectable failure.
type Fault interface {
	Apply(ctx context.Context, resource Resource, params Params) (RollbackFn, error)
	Validate(params Params) error
}

// Probe checks the steady-state hypothesis.
type Probe interface {
	Check(ctx context.Context, params Params) (ProbeResult, error)
}

// TargetFactory creates a Target instance.
type TargetFactory func() (Target, error)

// FaultFactory creates a Fault instance.
type FaultFactory func() (Fault, error)

// ProbeFactory creates a Probe instance.
type ProbeFactory func() (Probe, error)
