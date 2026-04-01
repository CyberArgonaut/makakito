// Package store provides SQLite-backed persistence for experiment results.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/CyberArgonaut/makakito/pkg/schema"
	_ "modernc.org/sqlite" // registers the "sqlite" driver for database/sql
)

// SQLiteStore is a Store backed by SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at the given path.
// Use ":memory:" for an in-memory database.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %q: %w", dbPath, err)
	}

	// Enable WAL mode for concurrent reads.
	if _, err := db.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.Init(context.Background()); err != nil {
		db.Close() //nolint:errcheck
		return nil, err
	}
	return s, nil
}

// Init creates the schema if it does not exist.
func (s *SQLiteStore) Init(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS experiments (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL,
    started_at  DATETIME NOT NULL,
    finished_at DATETIME,
    data        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_experiments_status     ON experiments(status);
CREATE INDEX IF NOT EXISTS idx_experiments_started_at ON experiments(started_at);
`
	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// Save upserts an experiment result.
func (s *SQLiteStore) Save(ctx context.Context, result *schema.ExperimentResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	finishedAt := sql.NullString{}
	if result.FinishedAt != nil {
		finishedAt = sql.NullString{String: result.FinishedAt.UTC().Format("2006-01-02T15:04:05.999999999Z"), Valid: true}
	}

	const q = `INSERT OR REPLACE INTO experiments (id, name, status, started_at, finished_at, data)
               VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, q,
		result.ID,
		result.Name,
		string(result.Status),
		result.StartedAt.UTC().Format("2006-01-02T15:04:05.999999999Z"),
		finishedAt,
		string(data),
	)
	if err != nil {
		return fmt.Errorf("saving experiment %q: %w", result.ID, err)
	}
	return nil
}

// Get retrieves a single experiment result by ID.
func (s *SQLiteStore) Get(ctx context.Context, id string) (*schema.ExperimentResult, error) {
	row := s.db.QueryRowContext(ctx, "SELECT data FROM experiments WHERE id = ?", id)
	var data string
	if err := row.Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("experiment %q not found", id)
		}
		return nil, fmt.Errorf("querying experiment %q: %w", id, err)
	}
	var result schema.ExperimentResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("unmarshaling experiment %q: %w", id, err)
	}
	return &result, nil
}

// List returns recent experiments up to the given limit.
func (s *SQLiteStore) List(ctx context.Context, limit int) ([]schema.ExperimentResult, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT data FROM experiments ORDER BY started_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("listing experiments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanResults(rows)
}

// ListByStatus returns experiments with the given status up to the given limit.
func (s *SQLiteStore) ListByStatus(ctx context.Context, status schema.ExperimentStatus, limit int) ([]schema.ExperimentResult, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT data FROM experiments WHERE status = ? ORDER BY started_at DESC LIMIT ?",
		string(status), limit)
	if err != nil {
		return nil, fmt.Errorf("listing experiments by status %q: %w", status, err)
	}
	defer func() { _ = rows.Close() }()
	return scanResults(rows)
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func scanResults(rows *sql.Rows) ([]schema.ExperimentResult, error) {
	var results []schema.ExperimentResult
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		var r schema.ExperimentResult
		if err := json.Unmarshal([]byte(data), &r); err != nil {
			return nil, fmt.Errorf("unmarshaling row: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return results, nil
}
