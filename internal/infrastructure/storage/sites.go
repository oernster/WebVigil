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

// The JSON columns have their own wire shapes so the database format never follows a change to
// a domain type by accident.
type extractionRow struct {
	Selectors []string   `json:"selectors"`
	Fields    []fieldRow `json:"fields"`
}

type fieldRow struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Optional  bool   `json:"optional"`
}

type ignoreRow struct {
	ExcludeSelectors   []string `json:"excludeSelectors"`
	ExcludePatterns    []string `json:"excludePatterns"`
	CaseInsensitive    bool     `json:"caseInsensitive"`
	CollapseWhitespace bool     `json:"collapseWhitespace"`
}

const siteColumns = `id, name, url, enabled, interval_seconds, timeout_seconds, detection_mode,
	extraction_json, ignore_json, notify_sound, notify_visual, require_ack, failure_threshold`

type scanner interface{ Scan(dest ...any) error }

func scanSite(r scanner) (domain.Site, error) {
	var s domain.Site
	var interval, timeout int64
	var enabled, sound, visual, ack int
	var mode, extJSON, ignJSON string
	if err := r.Scan(&s.ID, &s.Name, &s.URL, &enabled, &interval, &timeout, &mode, &extJSON, &ignJSON,
		&sound, &visual, &ack, &s.FailureThreshold); err != nil {
		return domain.Site{}, err
	}
	var ext extractionRow
	var ign ignoreRow
	if err := json.Unmarshal([]byte(extJSON), &ext); err != nil {
		return domain.Site{}, fmt.Errorf("site %d extraction rules: %w", s.ID, err)
	}
	if err := json.Unmarshal([]byte(ignJSON), &ign); err != nil {
		return domain.Site{}, fmt.Errorf("site %d ignore rules: %w", s.ID, err)
	}
	s.Enabled, s.Notify = enabled == 1, domain.NotificationPrefs{Sound: sound == 1, Visual: visual == 1, RequireAck: ack == 1}
	s.Interval, s.Timeout = time.Duration(interval)*time.Second, time.Duration(timeout)*time.Second
	s.Mode = domain.DetectionMode(mode)
	s.Extraction.Selectors = ext.Selectors
	for _, f := range ext.Fields {
		s.Extraction.Fields = append(s.Extraction.Fields, domain.FieldRule(f))
	}
	s.Ignore = domain.IgnoreRules(ign)
	return s, nil
}

func siteArgs(s domain.Site) []any {
	ext := extractionRow{Selectors: s.Extraction.Selectors}
	for _, f := range s.Extraction.Fields {
		ext.Fields = append(ext.Fields, fieldRow(f))
	}
	extJSON, _ := json.Marshal(ext)                 // strings and bools always marshal
	ignJSON, _ := json.Marshal(ignoreRow(s.Ignore)) // likewise
	return []any{s.Name, s.URL, boolInt(s.Enabled), int64(s.Interval / time.Second), int64(s.Timeout / time.Second),
		string(s.Mode), string(extJSON), string(ignJSON), boolInt(s.Notify.Sound), boolInt(s.Notify.Visual),
		boolInt(s.Notify.RequireAck), s.FailureThreshold}
}

// ListSites returns every site in creation order.
func (s *Store) ListSites(ctx context.Context) ([]domain.Site, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+siteColumns+" FROM sites ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, rows.Err()
}

// GetSite returns one site.
func (s *Store) GetSite(ctx context.Context, id domain.SiteID) (domain.Site, error) {
	site, err := scanSite(s.db.QueryRowContext(ctx, "SELECT "+siteColumns+" FROM sites WHERE id = ?", id))
	if err != nil {
		return domain.Site{}, notFound(err, fmt.Sprintf("site %d", id))
	}
	return site, nil
}

const insertSite = `INSERT INTO sites (name, url, enabled, interval_seconds, timeout_seconds, detection_mode,
	extraction_json, ignore_json, notify_sound, notify_visual, require_ack, failure_threshold)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateSite = `UPDATE sites SET name = ?, url = ?, enabled = ?, interval_seconds = ?, timeout_seconds = ?,
	detection_mode = ?, extraction_json = ?, ignore_json = ?, notify_sound = ?, notify_visual = ?,
	require_ack = ?, failure_threshold = ? WHERE id = ?`

func addSite(ctx context.Context, tx *sql.Tx, site domain.Site, st domain.SiteState) (domain.SiteID, error) {
	res, err := tx.ExecContext(ctx, insertSite, siteArgs(site)...)
	if err != nil {
		return 0, err
	}
	n, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	st.SiteID = domain.SiteID(n)
	return st.SiteID, saveState(ctx, tx, st)
}

func replaceSite(ctx context.Context, tx *sql.Tx, site domain.Site, st domain.SiteState) error {
	res, err := tx.ExecContext(ctx, updateSite, append(siteArgs(site), site.ID)...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("site %d: %w", site.ID, errNotFound)
	}
	st.SiteID = site.ID
	return saveState(ctx, tx, st)
}

// AddSite stores a new site with its initial state.
func (s *Store) AddSite(ctx context.Context, site domain.Site, st domain.SiteState) (domain.SiteID, error) {
	var id domain.SiteID
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		id, err = addSite(ctx, tx, site, st)
		return err
	})
	return id, err
}

// UpdateSite replaces a site's definition and state together.
func (s *Store) UpdateSite(ctx context.Context, site domain.Site, st domain.SiteState) error {
	return s.tx(ctx, func(tx *sql.Tx) error { return replaceSite(ctx, tx, site, st) })
}

// DeleteSite removes a site; its state, history, snapshots and changes go with it by cascade.
func (s *Store) DeleteSite(ctx context.Context, id domain.SiteID) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM sites WHERE id = ?", id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("site %d: %w", id, errNotFound)
		}
		return nil
	})
}

// ApplyImport adds or replaces every site of a confirmed import in one transaction (FR-015).
func (s *Store) ApplyImport(ctx context.Context, writes []application.ImportWrite) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		for _, w := range writes {
			if w.Site.ID == 0 {
				if _, err := addSite(ctx, tx, w.Site, w.State); err != nil {
					return err
				}
				continue
			}
			if err := replaceSite(ctx, tx, w.Site, w.State); err != nil {
				return err
			}
		}
		return nil
	})
}
