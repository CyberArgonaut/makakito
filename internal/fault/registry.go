package fault

import "github.com/CyberArgonaut/makakito/internal/engine"

// RegisterAll registers all built-in faults into the given registry.
// dockerAPI may be nil if docker faults are not needed.
func RegisterAll(reg *engine.FaultRegistry, dockerAPI DockerFaultAPI) {
	netem := NewNetemDriver()

	reg.Register("cpu", func() (engine.Fault, error) {
		return &CPUFault{}, nil
	})
	reg.Register("memory", func() (engine.Fault, error) {
		return &MemoryFault{}, nil
	})
	reg.Register("process-kill", func() (engine.Fault, error) {
		return &ProcessKillFault{}, nil
	})
	reg.Register("network-latency", func() (engine.Fault, error) {
		return NewNetworkLatencyFault(netem), nil
	})
	reg.Register("packet-loss", func() (engine.Fault, error) {
		return NewPacketLossFault(netem), nil
	})

	if dockerAPI != nil {
		reg.Register("container-stop", func() (engine.Fault, error) {
			return NewContainerStopFault(dockerAPI), nil
		})
		reg.Register("container-pause", func() (engine.Fault, error) {
			return NewContainerPauseFault(dockerAPI), nil
		})
	}
}
