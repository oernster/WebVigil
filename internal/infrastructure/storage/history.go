package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

type fieldValueRow struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// RecordCheck writes one check: its history row, a snapshot and change event when there are
// any, the new state and the retention pass, all in one transaction.
func (s *Store) RecordCheck(ctx context.Context, w application.CheckWrite) (application.CheckIDs, error) {
	var ids application.CheckIDs
	err := s.tx(ctx, func(tx *sql.Tx) error {
		st := w.State
		if w.Snapshot != nil {
			id, err := insertSnapshot(ctx, tx, *w.Snapshot)
			if err != nil {
				return err
			}
			ids.SnapshotID, st.BaselineSnapshotID = id, id
		}
		if w.Change != nil {
			c := w.Change
			res, err := tx.ExecContext(ctx, `INSERT INTO change_events (site_id, prev_snapshot_id, new_snapshot_id,
				prev_hash, new_hash, summary, detected_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				c.SiteID, c.PrevSnapshotID, ids.SnapshotID, c.PrevHash, c.NewHash, c.Summary, toNanos(c.DetectedAt))
			if err != nil {
				return err
			}
			ids.ChangeID, _ = res.LastInsertId()
			if w.SetPending {
				st.PendingChangeID = ids.ChangeID
			}
		}
		r := w.Record
		if _, err := tx.ExecContext(ctx, `INSERT INTO check_history (site_id, at, request_url, final_url, http_status,
			duration_ms, content_type, bytes, outcome, category, detail, hash, change_kind, manual)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.SiteID, toNanos(r.At), r.RequestURL, r.FinalURL, r.HTTPStatus, r.Duration.Milliseconds(), r.ContentType,
			r.Bytes, string(r.Outcome), string(r.Category), r.Detail, r.Hash, string(r.Change), boolInt(r.Manual)); err != nil {
			return err
		}
		if err := saveState(ctx, tx, st); err != nil {
			return err
		}
		return s.prune(ctx, tx, st)
	})
	return ids, err
}

func insertSnapshot(ctx context.Context, tx *sql.Tx, snap domain.Snapshot) (int64, error) {
	fields := make([]fieldValueRow, 0, len(snap.Content.Fields))
	for _, f := range snap.Content.Fields {
		fields = append(fields, fieldValueRow(f))
	}
	fieldsJSON, _ := json.Marshal(fields) // strings always marshal
	res, err := tx.ExecContext(ctx, `INSERT INTO snapshots (site_id, hash, text, fields_json, captured_at)
		VALUES (?, ?, ?, ?, ?)`, snap.SiteID, snap.Hash, snap.Content.Text, string(fieldsJSON), toNanos(snap.CapturedAt))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// prune keeps the newest checks, the newest acknowledged changes and every pending one; it keeps the
// newest snapshots plus the baseline and any snapshot a kept change refers to (FR-103).
func (s *Store) prune(ctx context.Context, tx *sql.Tx, st domain.SiteState) error {
	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM check_history WHERE site_id = ?1 AND id NOT IN
			(SELECT id FROM check_history WHERE site_id = ?1 ORDER BY id DESC LIMIT ?2)`,
			[]any{st.SiteID, s.retention.Checks}},
		{`DELETE FROM change_events WHERE site_id = ?1 AND acknowledged_at != 0 AND id NOT IN
			(SELECT id FROM change_events WHERE site_id = ?1 ORDER BY id DESC LIMIT ?2)`,
			[]any{st.SiteID, s.retention.Snapshots}},
		{`DELETE FROM snapshots WHERE site_id = ?1 AND id != ?2
			AND id NOT IN (SELECT id FROM snapshots WHERE site_id = ?1 ORDER BY id DESC LIMIT ?3)
			AND id NOT IN (SELECT prev_snapshot_id FROM change_events WHERE site_id = ?1)
			AND id NOT IN (SELECT new_snapshot_id FROM change_events WHERE site_id = ?1)`,
			[]any{st.SiteID, st.BaselineSnapshotID, s.retention.Snapshots}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.sql, step.args...); err != nil {
			return fmt.Errorf("retention: %w", err)
		}
	}
	return nil
}

// Acknowledge stamps every pending change of a site and clears its pending alert.
func (s *Store) Acknowledge(ctx context.Context, id domain.SiteID, at time.Time) (int, error) {
	var n int64
	err := s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE change_events SET acknowledged_at = ? WHERE site_id = ? AND acknowledged_at = 0",
			toNanos(at), id)
		if err != nil {
			return err
		}
		n, _ = res.RowsAffected()
		_, err = tx.ExecContext(ctx, "UPDATE site_state SET pending_change_id = 0 WHERE site_id = ?", id)
		return err
	})
	return int(n), err
}

// PendingCount returns how many sites have an unacknowledged change.
func (s *Store) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM site_state WHERE pending_change_id != 0").Scan(&n)
	return n, err
}

// History returns the newest checks of a site, newest first.
func (s *Store) History(ctx context.Context, id domain.SiteID, limit int) ([]domain.CheckRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, site_id, at, request_url, final_url, http_status, duration_ms,
		content_type, bytes, outcome, category, detail, hash, change_kind, manual
		FROM check_history WHERE site_id = ? ORDER BY id DESC LIMIT ?`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CheckRecord
	for rows.Next() {
		var r domain.CheckRecord
		var at, ms int64
		var outcome, category, change string
		var manual int
		if err := rows.Scan(&r.ID, &r.SiteID, &at, &r.RequestURL, &r.FinalURL, &r.HTTPStatus, &ms, &r.ContentType,
			&r.Bytes, &outcome, &category, &r.Detail, &r.Hash, &change, &manual); err != nil {
			return nil, err
		}
		r.At, r.Duration, r.Manual = fromNanos(at), time.Duration(ms)*time.Millisecond, manual == 1
		r.Outcome, r.Category, r.Change = domain.Outcome(outcome), domain.Category(category), domain.ChangeKind(change)
		out = append(out, r)
	}
	return out, rows.Err()
}

const changeColumns = `id, site_id, prev_snapshot_id, new_snapshot_id, prev_hash, new_hash, summary, detected_at, acknowledged_at`

func scanChange(r scanner) (domain.ChangeEvent, error) {
	var c domain.ChangeEvent
	var detected, acked int64
	if err := r.Scan(&c.ID, &c.SiteID, &c.PrevSnapshotID, &c.NewSnapshotID, &c.PrevHash, &c.NewHash, &c.Summary,
		&detected, &acked); err != nil {
		return domain.ChangeEvent{}, err
	}
	c.DetectedAt, c.AcknowledgedAt = fromNanos(detected), fromNanos(acked)
	return c, nil
}

// Changes returns the newest change events of a site, newest first.
func (s *Store) Changes(ctx context.Context, id domain.SiteID, limit int) ([]domain.ChangeEvent, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+changeColumns+" FROM change_events WHERE site_id = ? ORDER BY id DESC LIMIT ?", id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ChangeEvent
	for rows.Next() {
		c, err := scanChange(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChange returns one change event.
func (s *Store) GetChange(ctx context.Context, id int64) (domain.ChangeEvent, error) {
	c, err := scanChange(s.db.QueryRowContext(ctx, "SELECT "+changeColumns+" FROM change_events WHERE id = ?", id))
	if err != nil {
		return domain.ChangeEvent{}, notFound(err, fmt.Sprintf("change %d", id))
	}
	return c, nil
}

// GetSnapshot returns one snapshot.
func (s *Store) GetSnapshot(ctx context.Context, id int64) (domain.Snapshot, error) {
	var snap domain.Snapshot
	var fieldsJSON string
	var captured int64
	err := s.db.QueryRowContext(ctx, "SELECT id, site_id, hash, text, fields_json, captured_at FROM snapshots WHERE id = ?", id).
		Scan(&snap.ID, &snap.SiteID, &snap.Hash, &snap.Content.Text, &fieldsJSON, &captured)
	if err != nil {
		return domain.Snapshot{}, notFound(err, fmt.Sprintf("snapshot %d", id))
	}
	var fields []fieldValueRow
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return domain.Snapshot{}, fmt.Errorf("snapshot %d fields: %w", id, err)
	}
	for _, f := range fields {
		snap.Content.Fields = append(snap.Content.Fields, domain.FieldValue(f))
	}
	snap.CapturedAt = fromNanos(captured)
	return snap, nil
}
