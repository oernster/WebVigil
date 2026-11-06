package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/domain"
)

// historyLimit is how many checks and changes the details panel shows.
const historyLimit = 200

// exchangeFilter offers the WebVigil sites format in the file dialogs.
var exchangeFilter = []runtime.FileFilter{
	{DisplayName: "WebVigil sites (*" + domain.ExchangeExtension + ")", Pattern: "*" + domain.ExchangeExtension},
	{DisplayName: "JSON files (*.json)", Pattern: "*.json"},
}

// History returns a site's recent checks and changes.
func (a *App) History(id int64) (HistoryDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return HistoryDTO{}, err
	}
	ctx := context.Background()
	checks, err := svc.history.Checks(ctx, domain.SiteID(id), historyLimit)
	if err != nil {
		return HistoryDTO{}, err
	}
	changes, err := svc.history.Changes(ctx, domain.SiteID(id), historyLimit)
	if err != nil {
		return HistoryDTO{}, err
	}
	out := HistoryDTO{Checks: []CheckDTO{}, Changes: []ChangeDTO{}}
	for _, c := range checks {
		out.Checks = append(out.Checks, CheckDTO{At: stamp(c.At), HTTPStatus: c.HTTPStatus, DurationMs: c.Duration.Milliseconds(),
			Outcome: string(c.Outcome), Category: string(c.Category), Detail: c.Detail, Hash: c.Hash, Change: string(c.Change),
			Manual: c.Manual, FinalURL: c.FinalURL, ContentType: c.ContentType, Bytes: c.Bytes})
	}
	for _, c := range changes {
		out.Changes = append(out.Changes, changeDTO(c))
	}
	return out, nil
}

// ChangeDetail returns one change with its diff.
func (a *App) ChangeDetail(changeID int64) (ChangeDetailDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return ChangeDetailDTO{}, err
	}
	d, err := svc.history.Detail(context.Background(), changeID)
	if err != nil {
		return ChangeDetailDTO{}, err
	}
	out := ChangeDetailDTO{Change: changeDTO(d.Change), Lines: []DiffLineDTO{}, Fields: []FieldChangeDTO{}}
	for _, l := range d.Lines {
		out.Lines = append(out.Lines, DiffLineDTO{Op: string(l.Op), Text: l.Text})
	}
	for _, f := range d.Fields {
		out.Fields = append(out.Fields, FieldChangeDTO(f))
	}
	return out, nil
}

// ExportSites asks where to save, then writes the chosen sites (all when ids is empty). An empty
// path in the result means the user cancelled.
func (a *App) ExportSites(ids []int64) (ExportResultDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return ExportResultDTO{}, err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Export sites", DefaultFilename: "sites" + domain.ExchangeExtension, Filters: exchangeFilter,
	})
	if err != nil || path == "" {
		return ExportResultDTO{}, err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += domain.ExchangeExtension
	}
	chosen := make([]domain.SiteID, 0, len(ids))
	for _, id := range ids {
		chosen = append(chosen, domain.SiteID(id))
	}
	n, err := svc.exchange.Export(context.Background(), path, chosen)
	return ExportResultDTO{Path: path, Count: n}, err
}

// PreviewImport asks for a file, validates it and plans each entry. Nothing is written yet. An
// empty path in the result means the user cancelled.
func (a *App) PreviewImport() (ImportPreviewDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return ImportPreviewDTO{}, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Import sites", Filters: exchangeFilter})
	if err != nil || path == "" {
		return ImportPreviewDTO{}, err
	}
	p, err := svc.exchange.PreviewImport(context.Background(), path)
	if err != nil {
		return ImportPreviewDTO{}, err
	}
	a.mu.Lock()
	a.pending[path] = p
	a.mu.Unlock()
	out := ImportPreviewDTO{Path: path, Items: []ImportItemDTO{}, Invalid: []InvalidEntryDTO{}}
	for _, it := range p.Plan {
		out.Items = append(out.Items, ImportItemDTO{Index: it.Index, Name: it.Site.Name, URL: it.Site.URL,
			Duplicate: it.Duplicate, CanUpdate: it.ExistingID != 0, Action: string(it.Action)})
	}
	for _, inv := range p.Invalid {
		out.Invalid = append(out.Invalid, InvalidEntryDTO(inv))
	}
	return out, nil
}

// ConfirmImport applies the previewed import with the chosen action per entry index.
func (a *App) ConfirmImport(path string, actions map[int]string) (ImportSummaryDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return ImportSummaryDTO{}, err
	}
	a.mu.Lock()
	p, ok := a.pending[path]
	delete(a.pending, path)
	a.mu.Unlock()
	if !ok {
		return ImportSummaryDTO{}, fmt.Errorf("the import preview for %s has expired; please import again", path)
	}
	plan := make([]domain.ImportPlanItem, len(p.Plan))
	for i, it := range p.Plan {
		if act, ok := actions[it.Index]; ok {
			it.Action = domain.ImportAction(act)
		}
		plan[i] = it
	}
	s, err := svc.exchange.ConfirmImport(context.Background(), plan, len(p.Invalid))
	if err != nil {
		return ImportSummaryDTO{}, err
	}
	_, _ = svc.scheduler.Tick(context.Background()) // imported sites are due now
	return ImportSummaryDTO(s), nil
}

// SaveSettings stores the settings.
func (a *App) SaveSettings(s SettingsDTO) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.settings.Save(context.Background(), s.settings())
}

// SetPaused pauses or resumes automatic monitoring.
func (a *App) SetPaused(paused bool) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.settings.SetPaused(context.Background(), paused)
}

// SetTheme stores the appearance preference.
func (a *App) SetTheme(theme string) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.settings.SetTheme(context.Background(), domain.Theme(theme))
}

// ChooseSound asks for a WAV file; an empty result means the user cancelled.
func (a *App) ChooseSound() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose alarm sound",
		Filters: []runtime.FileFilter{{DisplayName: "WAV sound (*.wav)", Pattern: "*.wav"}}})
}

// TestSound plays path; the built-in alarm when it is empty.
func (a *App) TestSound(path string) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.alerts.TestSound(path)
}
