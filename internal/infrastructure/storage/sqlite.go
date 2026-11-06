// Package storage implements the application SiteStore, HistoryStore and SettingsStore on a local
// pure-Go SQLite database (modernc.org/sqlite, no CGO).
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"webvigil/internal/domain"
)

const driverName = "sqlite"

// busyTimeoutMillis is how long a connection blocked by another writer waits before SQLITE_BUSY.
const busyTimeoutMillis = 5000

// Retention says how much history each site keeps (REQUIREMENTS.md FR-103).
type Retention struct {
	Checks    int
	Snapshots int
}

// DefaultRetention is the retention the application uses.
var DefaultRetention = Retention{Checks: domain.KeepChecksPerSite, Snapshots: domain.KeepSnapshotsPerSite}

// Store is the SQLite implementation of the application storage ports.
type Store struct {
	db        *sql.DB
	retention Retention
}

// Open opens or creates the database at path and applies pending migrations.
func Open(ctx context.Context, path string, retention Retention) (*Store, error) {
	// The pragmas are on the DSN so every pooled connection receives them, not just the first:
	// busy_timeout and foreign_keys are per-connection settings. _txlock=immediate takes the write
	// lock at BEGIN, so two writers serialise instead of deadlocking on a lock upgrade.
	dsn := path + "?" + strings.Join([]string{
		fmt.Sprintf("_pragma=busy_timeout(%d)", busyTimeoutMillis),
		"_pragma=journal_mode(WAL)",
		"_pragma=synchronous(NORMAL)",
		"_pragma=foreign_keys(ON)",
		"_txlock=immediate",
	}, "&")
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", path, err)
	}
	s := &Store{db: db, retention: retention}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

// migrate applies each pending step in its own transaction together with the version bump, so
// a crash leaves the database at a whole step, never half way through one.
func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version > len(migrations) {
		return fmt.Errorf("the database is schema version %d, newer than this WebVigil (%d)", version, len(migrations))
	}
	for step := version; step < len(migrations); step++ {
		err := s.tx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, migrations[step]); err != nil {
				return err
			}
			// PRAGMA user_version takes no bound parameter; the value is a trusted integer.
			_, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", step+1))
			return err
		})
		if err != nil {
			return fmt.Errorf("apply schema step %d: %w", step+1, err)
		}
	}
	return nil
}

// tx runs fn in a transaction, committing on success and rolling back on error.
func (s *Store) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// notFound maps sql.ErrNoRows to the application sentinel.
func notFound(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", what, errNotFound)
	}
	return err
}

// Times are stored as Unix nanoseconds; zero stands for the zero time.
func toNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func fromNanos(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n).UTC()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
