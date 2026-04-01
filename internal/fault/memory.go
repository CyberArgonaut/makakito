package fault

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// MemoryFault allocates and holds memory.
type MemoryFault struct{}

// Validate checks params.
func (f *MemoryFault) Validate(params engine.Params) error {
	size := params["size"]
	if size == "" {
		return fmt.Errorf("memory fault: \"size\" param is required (e.g. \"100MB\" or \"100\")")
	}
	if _, err := parseMemorySize(size); err != nil {
		return fmt.Errorf("memory fault: %w", err)
	}
	return nil
}

// Apply allocates memory and holds it until rollback.
func (f *MemoryFault) Apply(_ context.Context, _ engine.Resource, params engine.Params) (engine.RollbackFn, error) {
	size := params["size"]
	if size == "" {
		return engine.NoopRollback(), fmt.Errorf("memory fault: \"size\" param is required")
	}

	bytes, err := parseMemorySize(size)
	if err != nil {
		return engine.NoopRollback(), fmt.Errorf("memory fault: %w", err)
	}

	// Allocate and touch memory to ensure it's physically allocated.
	mem := make([]byte, bytes)
	for i := range mem {
		mem[i] = 0xAA
	}

	held := mem // keep reference alive until rollback
	return func(_ context.Context) error {
		held = nil // release reference
		_ = held
		return nil
	}, nil
}

// parseMemorySize parses a size string like "100MB", "100mb", or "100" (MB).
func parseMemorySize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	var multiplier int64 = 1024 * 1024 // default: MB
	numStr := s

	switch {
	case strings.HasSuffix(upper, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "MB"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "KB"):
		multiplier = 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "B"):
		multiplier = 1
		numStr = s[:len(s)-1]
	}

	n, err := strconv.ParseInt(strings.TrimSpace(numStr), 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid size %q: must be a positive number (e.g. 100MB)", s)
	}
	return n * multiplier, nil
}
