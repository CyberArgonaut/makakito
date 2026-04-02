package fault

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// PacketLossFault injects packet loss on a network interface.
type PacketLossFault struct {
	driver NetworkFaultDriver
}

// NewPacketLossFault creates a PacketLossFault.
func NewPacketLossFault(driver NetworkFaultDriver) *PacketLossFault {
	return &PacketLossFault{driver: driver}
}

// Validate checks params.
func (f *PacketLossFault) Validate(params engine.Params) error {
	s := params["percent"]
	if s == "" {
		return fmt.Errorf("packet-loss fault: \"percent\" param is required (0–100)")
	}
	pct, err := strconv.ParseFloat(s, 64)
	if err != nil || pct < 0 || pct > 100 {
		return fmt.Errorf("packet-loss fault: \"percent\" must be 0–100, got %q", s)
	}
	return nil
}

// Apply injects packet loss and returns a rollback that removes the qdisc.
func (f *PacketLossFault) Apply(ctx context.Context, _ engine.Resource, params engine.Params) (engine.RollbackFn, error) {
	iface := params["interface"]
	if iface == "" {
		iface = "eth0"
	}

	s := params["percent"]
	if s == "" {
		return engine.NoopRollback(), fmt.Errorf("packet-loss fault: \"percent\" param is required")
	}
	pct, err := strconv.ParseFloat(s, 64)
	if err != nil || pct < 0 || pct > 100 {
		return engine.NoopRollback(), fmt.Errorf("packet-loss fault: \"percent\" must be 0–100, got %q", s)
	}

	rollback := func(ctx context.Context) error {
		return f.driver.Remove(ctx, iface)
	}

	if err := f.driver.AddPacketLoss(ctx, iface, pct); err != nil {
		return rollback, fmt.Errorf("packet-loss fault: %w", err)
	}
	return rollback, nil
}
