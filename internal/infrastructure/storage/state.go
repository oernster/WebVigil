package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"webvigil/internal/domain"
)

const stateColumns = `site_id, last_outcome, last_category, last_detail, last_http_status, last_checked_at,
	last_success_at, last_change_at, baseline_snapshot_id, baseline_hash, pending_change_id,
	consecutive_failures, failure_warned, etag, last_modified, next_due_at`

func scanState(r scanner) (domain.SiteState, error) {
	var st domain.SiteState
	var outcome, category string
	var checked, success, change, due int64
	var warned int
	err := r.Scan(&st.SiteID, &outcome, &category, &st.LastDetail, &st.LastHTTPStatus, &checked, &success, &change,
		&st.BaselineSnapshotID, &st.BaselineHash, &st.PendingChangeID, &st.ConsecutiveFailures, &warned,
		&st.ETag, &st.LastModified, &due)
	if err != nil {
		return domain.SiteState{}, err
	}
	st.LastOutcome, st.LastCategory = domain.Outcome(outcome), domain.Category(category)
	st.LastCheckedAt, st.LastSuccessAt, st.LastChangeAt = fromNanos(checked), fromNanos(success), fromNanos(change)
	st.NextDueAt, st.FailureWarned = fromNanos(due), warned == 1
	return st, nil
}

func saveState(ctx context.Context, tx *sql.Tx, st domain.SiteState) error {
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO site_state (`+stateColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.SiteID, string(st.LastOutcome), string(st.LastCategory), st.LastDetail, st.LastHTTPStatus,
		toNanos(st.LastCheckedAt), toNanos(st.LastSuccessAt), toNanos(st.LastChangeAt), st.BaselineSnapshotID,
		st.BaselineHash, st.PendingChangeID, st.ConsecutiveFailures, boolInt(st.FailureWarned), st.ETag,
		st.LastModified, toNanos(st.NextDueAt))
	return err
}

// ListStates returns the state of every site.
func (s *Store) ListStates(ctx context.Context) ([]domain.SiteState, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+stateColumns+" FROM site_state ORDER BY site_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SiteState
	for rows.Next() {
		st, err := scanState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// GetState returns the state of one site.
func (s *Store) GetState(ctx context.Context, id domain.SiteID) (domain.SiteState, error) {
	st, err := scanState(s.db.QueryRowContext(ctx, "SELECT "+stateColumns+" FROM site_state WHERE site_id = ?", id))
	if err != nil {
		return domain.SiteState{}, notFound(err, fmt.Sprintf("state of site %d", id))
	}
	return st, nil
}

const settingsKey = "settings"

type settingsRow struct {
	Theme            string `json:"theme"`
	Paused           bool   `json:"paused"`
	StartAtSignIn    bool   `json:"startAtSignIn"`
	SoundPath        string `json:"soundPath"`
	RepeatAlarm      bool   `json:"repeatAlarm"`
	ForegroundDialog bool   `json:"foregroundDialog"`
}

// LoadSettings returns the stored settings; the defaults on a first run.
func (s *Store) LoadSettings(ctx context.Context) (domain.Settings, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM application_settings WHERE key = ?", settingsKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return domain.DefaultSettings(), nil
	}
	if err != nil {
		return domain.Settings{}, err
	}
	var row settingsRow
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		return domain.Settings{}, fmt.Errorf("stored settings: %w", err)
	}
	out := domain.Settings{Theme: domain.Theme(row.Theme), Paused: row.Paused, StartAtSignIn: row.StartAtSignIn,
		SoundPath: row.SoundPath, RepeatAlarm: row.RepeatAlarm, ForegroundDialog: row.ForegroundDialog}
	if out.Validate() != nil {
		out.Theme = domain.ThemeSystem
	}
	return out, nil
}

// SaveSettings stores the settings.
func (s *Store) SaveSettings(ctx context.Context, set domain.Settings) error {
	raw, _ := json.Marshal(settingsRow{Theme: string(set.Theme), Paused: set.Paused, StartAtSignIn: set.StartAtSignIn,
		SoundPath: set.SoundPath, RepeatAlarm: set.RepeatAlarm, ForegroundDialog: set.ForegroundDialog})
	_, err := s.db.ExecContext(ctx, "INSERT OR REPLACE INTO application_settings (key, value) VALUES (?, ?)",
		settingsKey, string(raw))
	return err
}
