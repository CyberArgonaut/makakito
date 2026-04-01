package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CyberArgonaut/makakito/pkg/schema"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	return s
}

func TestSQLiteStoreInit(t *testing.T) {
	s := newTestStore(t)
	// Init already called in constructor; calling again should be idempotent.
	if err := s.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
}

func TestSQLiteStoreSaveAndGet(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	result := &schema.ExperimentResult{
		ID:        "exp-001",
		Name:      "redis blackout",
		Status:    schema.StatusPassed,
		StartedAt: now,
	}
	if err := s.Save(context.Background(), result); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get(context.Background(), "exp-001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != result.ID {
		t.Errorf("ID = %q, want %q", got.ID, result.ID)
	}
	if got.Name != result.Name {
		t.Errorf("Name = %q, want %q", got.Name, result.Name)
	}
	if got.Status != schema.StatusPassed {
		t.Errorf("Status = %q, want %q", got.Status, schema.StatusPassed)
	}
}

func TestSQLiteStoreList(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		r := &schema.ExperimentResult{
			ID:        fmt.Sprintf("exp-%03d", i),
			Name:      fmt.Sprintf("experiment %d", i),
			Status:    schema.StatusPassed,
			StartedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}
	results, err := s.List(ctx, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("len = %d, want 3", len(results))
	}
}

func TestSQLiteStoreListByStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	statuses := []schema.ExperimentStatus{schema.StatusPassed, schema.StatusFailed, schema.StatusPassed}
	for i, st := range statuses {
		r := &schema.ExperimentResult{
			ID:        fmt.Sprintf("exp-%d", i),
			Name:      "x",
			Status:    st,
			StartedAt: time.Now(),
		}
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	results, err := s.ListByStatus(ctx, schema.StatusPassed, 10)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len = %d, want 2", len(results))
	}
}

func TestSQLiteStoreUpsert(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	r := &schema.ExperimentResult{
		ID:        "exp-upsert",
		Name:      "original",
		Status:    schema.StatusRunning,
		StartedAt: time.Now(),
	}
	if err := s.Save(ctx, r); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	r.Status = schema.StatusPassed
	r.Name = "updated"
	if err := s.Save(ctx, r); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	got, err := s.Get(ctx, "exp-upsert")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != schema.StatusPassed {
		t.Errorf("Status = %q, want passed", got.Status)
	}
}

func TestSQLiteStoreGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}
