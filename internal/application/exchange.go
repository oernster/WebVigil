package application

import (
	"context"
	"fmt"
	"time"

	"webvigil/internal/domain"
)

// Exchange exports and imports site definitions (FR-010 to FR-018).
type Exchange struct {
	store  SiteStore
	files  Files
	clock  Clock
	events Events
}

// NewExchange wires the exchange service.
func NewExchange(store SiteStore, files Files, clock Clock, events Events) *Exchange {
	return &Exchange{store: store, files: files, clock: clock, events: events}
}

// Export writes the chosen sites (every site when ids is empty) to path and returns how many.
func (x *Exchange) Export(ctx context.Context, path string, ids []domain.SiteID) (int, error) {
	sites, err := x.store.ListSites(ctx)
	if err != nil {
		return 0, err
	}
	if len(ids) > 0 {
		want := make(map[domain.SiteID]bool, len(ids))
		for _, id := range ids {
			want[id] = true
		}
		chosen := sites[:0:0]
		for _, s := range sites {
			if want[s.ID] {
				chosen = append(chosen, s)
			}
		}
		sites = chosen
	}
	data := domain.MarshalExport(domain.BuildExport(sites, x.clock.Now()))
	if err := x.files.WriteAtomic(path, data); err != nil {
		return 0, fmt.Errorf("write %s: %w", path, err)
	}
	return len(sites), nil
}

// ImportPreview is what an import would do, shown before anything is written.
type ImportPreview struct {
	Plan    []domain.ImportPlanItem
	Invalid []domain.InvalidEntry
}

// PreviewImport reads and validates path and plans each entry against the stored sites.
func (x *Exchange) PreviewImport(ctx context.Context, path string) (ImportPreview, error) {
	data, err := x.files.ReadLimited(path, domain.MaxImportBytes+1)
	if err != nil {
		return ImportPreview{}, fmt.Errorf("read %s: %w", path, err)
	}
	file, err := domain.ParseImport(data)
	if err != nil {
		return ImportPreview{}, err
	}
	existing, err := x.store.ListSites(ctx)
	if err != nil {
		return ImportPreview{}, err
	}
	return ImportPreview{Plan: domain.PlanImport(file.Candidates, existing), Invalid: file.Invalid}, nil
}

// ConfirmImport applies the chosen action for each planned entry in one transaction. Every
// imported or updated site whose meaning changed starts without a baseline, so its first check
// establishes one locally and raises no alarm (FR-015, FR-018).
func (x *Exchange) ConfirmImport(ctx context.Context, plan []domain.ImportPlanItem, invalid int) (domain.ImportSummary, error) {
	summary := domain.ImportSummary{Invalid: invalid}
	now := x.clock.Now()
	writes := make([]ImportWrite, 0, len(plan))
	for _, item := range plan {
		if err := item.Allowed(item.Action); err != nil {
			return domain.ImportSummary{}, err
		}
		site := item.Site
		switch item.Action {
		case domain.ImportSkip:
			summary.Skipped++
			continue
		case domain.ImportUpdate:
			w, err := x.updateWrite(ctx, item, now)
			if err != nil {
				return domain.ImportSummary{}, err
			}
			writes = append(writes, w)
			summary.Updated++
			continue
		}
		site.ID = 0
		writes = append(writes, ImportWrite{Site: site, State: domain.SiteState{NextDueAt: now}})
		summary.Imported++
	}
	if err := x.store.ApplyImport(ctx, writes); err != nil {
		return domain.ImportSummary{}, err
	}
	x.events.Emit(EventSitesChanged, 0)
	return summary, nil
}

// updateWrite replaces a stored definition, keeping its identity. Its baseline survives only when
// the imported definition compares content the same way.
func (x *Exchange) updateWrite(ctx context.Context, item domain.ImportPlanItem, now time.Time) (ImportWrite, error) {
	old, err := x.store.GetSite(ctx, item.ExistingID)
	if err != nil {
		return ImportWrite{}, err
	}
	state, err := x.store.GetState(ctx, item.ExistingID)
	if err != nil {
		return ImportWrite{}, err
	}
	site := item.Site
	site.ID = old.ID
	if old.BaselineKey() != site.BaselineKey() {
		state = state.WithoutBaseline(now)
	}
	return ImportWrite{Site: site, State: state}, nil
}
