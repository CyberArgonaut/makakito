package engine

import (
	"context"
	"fmt"
	"log/slog"
)

// RollbackStack collects RollbackFns and executes them in LIFO order.
type RollbackStack struct {
	fns    []RollbackFn
	logger *slog.Logger
}

// NewRollbackStack creates a new RollbackStack.
func NewRollbackStack(logger *slog.Logger) *RollbackStack {
	return &RollbackStack{logger: logger}
}

// Push adds a rollback function to the stack.
func (s *RollbackStack) Push(fn RollbackFn) {
	s.fns = append(s.fns, fn)
}

// Execute runs all rollbacks in LIFO order using a fresh context.
// It never stops on individual failures — all rollbacks are attempted.
func (s *RollbackStack) Execute() {
	ctx := context.Background()
	for i := len(s.fns) - 1; i >= 0; i-- {
		fn := s.fns[i]
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("panic in rollback", "recover", fmt.Sprintf("%v", r))
				}
			}()
			if err := fn(ctx); err != nil {
				s.logger.Error("rollback error", "error", err)
			}
		}()
	}
	s.fns = nil
}
