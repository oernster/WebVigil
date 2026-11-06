package application

import (
	"context"

	"webvigil/internal/domain"
)

// History reads check and change history and builds diffs (FR-100 to FR-102).
type History struct {
	store HistoryStore
}

// NewHistory wires the history service.
func NewHistory(store HistoryStore) *History {
	return &History{store: store}
}

// Checks returns the newest checks of a site, newest first.
func (h *History) Checks(ctx context.Context, id domain.SiteID, limit int) ([]domain.CheckRecord, error) {
	return h.store.History(ctx, id, limit)
}

// Changes returns the newest change events of a site, newest first.
func (h *History) Changes(ctx context.Context, id domain.SiteID, limit int) ([]domain.ChangeEvent, error) {
	return h.store.Changes(ctx, id, limit)
}

// ChangeDetail is one change with both versions and their differences.
type ChangeDetail struct {
	Change domain.ChangeEvent
	Prev   domain.Snapshot
	New    domain.Snapshot
	Lines  []domain.DiffLine
	Fields []domain.FieldChange
}

// Detail loads a change with its snapshots and computes the diff.
func (h *History) Detail(ctx context.Context, changeID int64) (ChangeDetail, error) {
	ch, err := h.store.GetChange(ctx, changeID)
	if err != nil {
		return ChangeDetail{}, err
	}
	prev, err := h.store.GetSnapshot(ctx, ch.PrevSnapshotID)
	if err != nil {
		return ChangeDetail{}, err
	}
	cur, err := h.store.GetSnapshot(ctx, ch.NewSnapshotID)
	if err != nil {
		return ChangeDetail{}, err
	}
	d := ChangeDetail{Change: ch, Prev: prev, New: cur}
	if len(prev.Content.Fields) > 0 || len(cur.Content.Fields) > 0 {
		d.Fields = domain.DiffFields(prev.Content.Fields, cur.Content.Fields)
	} else {
		d.Lines = domain.DiffLines(domain.SplitLines(prev.Content.Text), domain.SplitLines(cur.Content.Text))
	}
	return d, nil
}
