package fault

import (
	"context"
	"fmt"
	"time"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// NetworkLatencyFault injects artificial latency on a network interface.
type NetworkLatencyFault struct {
	driver NetworkFaultDriver
}

// NewNetworkLatencyFault creates a NetworkLatencyFault.
func NewNetworkLatencyFault(driver NetworkFaultDriver) *NetworkLatencyFault {
	return &NetworkLatencyFault{driver: driver}
}

// Validate checks params.
func (f *NetworkLatencyFault) Validate(params engine.Params) error {
	if params["latency"] == "" {
		return fmt.Errorf("network-latency fault: \"latency\" param is required (e.g. \"200ms\")")
	}
	return nil
}

// Apply injects latency and returns a rollback that removes the qdisc.
func (f *NetworkLatencyFault) Apply(ctx context.Context, _ engine.Resource, params engine.Params) (engine.RollbackFn, error) {
	iface := params["interface"]
	if iface == "" {
		iface = "eth0"
	}

	latencyStr := params["latency"]
	if latencyStr == "" {
		return engine.NoopRollback(), fmt.Errorf("network-latency fault: \"latency\" param is required")
	}
	latency, err := time.ParseDuration(latencyStr)
	if err != nil {
		return engine.NoopRollback(), fmt.Errorf("network-latency fault: invalid latency %q: %w", latencyStr, err)
	}

	var jitter time.Duration
	if j := params["jitter"]; j != "" {
		jitter, err = time.ParseDuration(j)
		if err != nil {
			return engine.NoopRollback(), fmt.Errorf("network-latency fault: invalid jitter %q: %w", j, err)
		}
	}

	rollback := func(ctx context.Context) error {
		return f.driver.Remove(ctx, iface)
	}

	if err := f.driver.AddLatency(ctx, iface, latency, jitter); err != nil {
		return rollback, fmt.Errorf("network-latency fault: %w", err)
	}
	return rollback, nil
}
