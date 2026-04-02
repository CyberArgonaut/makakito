package fault

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// CPUFault stresses CPU by spinning goroutines.
type CPUFault struct{}

// Validate checks params.
func (f *CPUFault) Validate(params engine.Params) error {
	if s, ok := params["cores"]; ok {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return fmt.Errorf("cpu fault: \"cores\" must be a positive integer, got %q", s)
		}
	}
	return nil
}

// Apply starts CPU stress goroutines. Rollback cancels them.
func (f *CPUFault) Apply(_ context.Context, _ engine.Resource, params engine.Params) (engine.RollbackFn, error) {
	cores := runtime.NumCPU()
	if s, ok := params["cores"]; ok {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return engine.NoopRollback(), fmt.Errorf("cpu fault: invalid cores %q: %w", s, err)
		}
		cores = n
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	for i := 0; i < cores; i++ {
		go func() {
			for stopCtx.Err() == nil {
				runtime.Gosched() // yield to scheduler while keeping CPU active
			}
		}()
	}

	return func(_ context.Context) error {
		cancel()
		return nil
	}, nil
}
