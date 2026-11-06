package domain

import "fmt"

// ImportAction is what a confirmed import does with one entry.
type ImportAction string

const (
	ImportAdd      ImportAction = "add"
	ImportSkip     ImportAction = "skip"
	ImportUpdate   ImportAction = "update"
	ImportSeparate ImportAction = "separate"
)

// ImportPlanItem is one entry of an import preview. A duplicate defaults to Skip, so nothing
// existing is overwritten unless the user chooses Update (FR-014).
type ImportPlanItem struct {
	Index      int
	Site       Site
	Duplicate  bool
	ExistingID SiteID // the stored site with the same URL; zero for a repeat within the file
	Action     ImportAction
}

// PlanImport marks each candidate as new or duplicate against the stored sites and against
// earlier entries in the same file.
func PlanImport(candidates []ImportCandidate, existing []Site) []ImportPlanItem {
	stored := make(map[string]SiteID, len(existing))
	for _, s := range existing {
		if _, ok := stored[s.URL]; !ok {
			stored[s.URL] = s.ID
		}
	}
	inFile := map[string]bool{}
	plan := make([]ImportPlanItem, 0, len(candidates))
	for _, c := range candidates {
		item := ImportPlanItem{Index: c.Index, Site: c.Site, Action: ImportAdd}
		if id, ok := stored[c.Site.URL]; ok {
			item.Duplicate, item.ExistingID, item.Action = true, id, ImportSkip
		} else if inFile[c.Site.URL] {
			item.Duplicate, item.Action = true, ImportSkip
		}
		inFile[c.Site.URL] = true
		plan = append(plan, item)
	}
	return plan
}

// Allowed reports whether action may be chosen for this item: a new entry may be added or
// skipped; a duplicate may be skipped or imported separately; it may be updated only when a stored
// site exists to update.
func (i ImportPlanItem) Allowed(action ImportAction) error {
	ok := false
	switch action {
	case ImportSkip:
		ok = true
	case ImportAdd:
		ok = !i.Duplicate
	case ImportSeparate:
		ok = i.Duplicate
	case ImportUpdate:
		ok = i.ExistingID != 0
	}
	if !ok {
		return fmt.Errorf("%w: entry %d cannot be %q", ErrInvalidSite, i.Index, action)
	}
	return nil
}

// ImportSummary counts the result of a confirmed import (FR-016).
type ImportSummary struct {
	Imported int
	Updated  int
	Skipped  int
	Invalid  int
}
