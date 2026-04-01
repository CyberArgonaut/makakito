package store

import (
	"context"

	"github.com/CyberArgonaut/makakito/pkg/schema"
)

// Store persists experiment results.
type Store interface {
	Init(ctx context.Context) error
	Save(ctx context.Context, result *schema.ExperimentResult) error
	Get(ctx context.Context, id string) (*schema.ExperimentResult, error)
	List(ctx context.Context, limit int) ([]schema.ExperimentResult, error)
	ListByStatus(ctx context.Context, status schema.ExperimentStatus, limit int) ([]schema.ExperimentResult, error)
	Close() error
}
