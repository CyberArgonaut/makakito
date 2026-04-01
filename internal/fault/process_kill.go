package fault

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// ProcessKillFault sends a signal to a process. Rollback is a no-op.
type ProcessKillFault struct{}

// Validate checks params — none required.
func (f *ProcessKillFault) Validate(_ engine.Params) error {
	return nil
}

// Apply sends a signal to the process identified by resource.ID (PID).
// Rollback is a no-op — the process should self-heal.
func (f *ProcessKillFault) Apply(_ context.Context, resource engine.Resource, params engine.Params) (engine.RollbackFn, error) {
	sig := parseSignal(params["signal"])

	pid, err := strconv.Atoi(resource.ID)
	if err != nil {
		return engine.NoopRollback(), fmt.Errorf("process-kill: invalid PID %q: %w", resource.ID, err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return engine.NoopRollback(), fmt.Errorf("process-kill: finding process %d: %w", pid, err)
	}

	if err := proc.Signal(sig); err != nil {
		return engine.NoopRollback(), fmt.Errorf("process-kill: sending signal to %d: %w", pid, err)
	}

	return engine.NoopRollback(), nil
}

func parseSignal(s string) os.Signal {
	switch s {
	case "SIGKILL", "kill", "9":
		return syscall.SIGKILL
	case "SIGINT", "int", "2":
		return syscall.SIGINT
	case "SIGTERM", "term", "15", "":
		return syscall.SIGTERM
	default:
		return syscall.SIGTERM
	}
}
