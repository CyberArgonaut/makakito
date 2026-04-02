package engine

import (
	"fmt"
	"sync"
)

// TargetRegistry is a thread-safe registry of target factories.
type TargetRegistry struct {
	mu      sync.RWMutex
	targets map[string]TargetFactory
}

// NewTargetRegistry creates a new TargetRegistry.
func NewTargetRegistry() *TargetRegistry {
	return &TargetRegistry{targets: make(map[string]TargetFactory)}
}

// Register adds a target factory for the given kind.
func (r *TargetRegistry) Register(kind string, factory TargetFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targets[kind] = factory
}

// Get returns a new Target instance for the given kind.
func (r *TargetRegistry) Get(kind string) (Target, error) {
	r.mu.RLock()
	factory, ok := r.targets[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown target kind: %q", kind)
	}
	return factory()
}

// FaultRegistry is a thread-safe registry of fault factories.
type FaultRegistry struct {
	mu     sync.RWMutex
	faults map[string]FaultFactory
}

// NewFaultRegistry creates a new FaultRegistry.
func NewFaultRegistry() *FaultRegistry {
	return &FaultRegistry{faults: make(map[string]FaultFactory)}
}

// Register adds a fault factory for the given kind.
func (r *FaultRegistry) Register(kind string, factory FaultFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.faults[kind] = factory
}

// Get returns a new Fault instance for the given kind.
func (r *FaultRegistry) Get(kind string) (Fault, error) {
	r.mu.RLock()
	factory, ok := r.faults[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown fault kind: %q", kind)
	}
	return factory()
}

// ProbeRegistry is a thread-safe registry of probe factories.
type ProbeRegistry struct {
	mu     sync.RWMutex
	probes map[string]ProbeFactory
}

// NewProbeRegistry creates a new ProbeRegistry.
func NewProbeRegistry() *ProbeRegistry {
	return &ProbeRegistry{probes: make(map[string]ProbeFactory)}
}

// Register adds a probe factory for the given kind.
func (r *ProbeRegistry) Register(kind string, factory ProbeFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.probes[kind] = factory
}

// Get returns a new Probe instance for the given kind.
func (r *ProbeRegistry) Get(kind string) (Probe, error) {
	r.mu.RLock()
	factory, ok := r.probes[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown probe kind: %q", kind)
	}
	return factory()
}
