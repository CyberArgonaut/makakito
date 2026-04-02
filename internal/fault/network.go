package fault

import (
	"context"
	"time"
)

// NetworkFaultDriver abstracts tc-netem for Phase 1/2, swappable to eBPF in Phase 3.
type NetworkFaultDriver interface {
	AddLatency(ctx context.Context, iface string, latency time.Duration, jitter time.Duration) error
	AddPacketLoss(ctx context.Context, iface string, percent float64) error
	Remove(ctx context.Context, iface string) error
}
