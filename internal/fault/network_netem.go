package fault

import (
	"context"
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
)

// NetemDriver implements NetworkFaultDriver via tc-netem.
type NetemDriver struct{}

// NewNetemDriver creates a NetemDriver.
func NewNetemDriver() *NetemDriver {
	return &NetemDriver{}
}

// AddLatency adds latency+jitter to the given interface via tc-netem.
func (d *NetemDriver) AddLatency(_ context.Context, iface string, latency, jitter time.Duration) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("netem: finding interface %q: %w", iface, err)
	}

	qdisc := netlink.NewNetem(netlink.QdiscAttrs{
		LinkIndex: link.Attrs().Index,
		Handle:    netlink.MakeHandle(1, 0),
		Parent:    netlink.HANDLE_ROOT,
	}, netlink.NetemQdiscAttrs{
		Latency: uint32(latency.Microseconds()),
		Jitter:  uint32(jitter.Microseconds()),
	})

	if err := netlink.QdiscAdd(qdisc); err != nil {
		return fmt.Errorf("netem: adding qdisc to %q: %w", iface, err)
	}
	return nil
}

// AddPacketLoss adds packet loss to the given interface via tc-netem.
func (d *NetemDriver) AddPacketLoss(_ context.Context, iface string, percent float64) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("netem: finding interface %q: %w", iface, err)
	}

	qdisc := netlink.NewNetem(netlink.QdiscAttrs{
		LinkIndex: link.Attrs().Index,
		Handle:    netlink.MakeHandle(1, 0),
		Parent:    netlink.HANDLE_ROOT,
	}, netlink.NetemQdiscAttrs{
		Loss: float32(percent),
	})

	if err := netlink.QdiscAdd(qdisc); err != nil {
		return fmt.Errorf("netem: adding packet-loss qdisc to %q: %w", iface, err)
	}
	return nil
}

// Remove removes the netem qdisc from the given interface.
func (d *NetemDriver) Remove(_ context.Context, iface string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("netem: finding interface %q: %w", iface, err)
	}

	qdiscs, err := netlink.QdiscList(link)
	if err != nil {
		return fmt.Errorf("netem: listing qdiscs for %q: %w", iface, err)
	}

	for _, q := range qdiscs {
		if _, ok := q.(*netlink.Netem); ok {
			if err := netlink.QdiscDel(q); err != nil {
				return fmt.Errorf("netem: removing qdisc from %q: %w", iface, err)
			}
		}
	}
	return nil
}
