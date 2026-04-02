package probe

import "github.com/CyberArgonaut/makakito/internal/engine"

// RegisterAll registers all built-in probes into the given registry.
func RegisterAll(reg *engine.ProbeRegistry) {
	reg.Register("http", func() (engine.Probe, error) {
		return &HTTPProbe{}, nil
	})
}
