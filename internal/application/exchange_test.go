package application

import (
	"errors"
	"strings"
	"testing"

	"webvigil/internal/domain"
)

// memFiles is an in-memory Files.
type memFiles struct {
	data     map[string][]byte
	writeErr error
	readErr  error
	limits   []int64
}

func (f *memFiles) ReadLimited(path string, limit int64) ([]byte, error) {
	f.limits = append(f.limits, limit)
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.data[path], nil
}

func (f *memFiles) WriteAtomic(path string, data []byte) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.data[path] = data
	return nil
}

func newExchangeRig() (*Exchange, *memStore, *memFiles, *Sites) {
	store := newMemStore()
	files := &memFiles{data: map[string][]byte{}}
	clock := &fakeClock{now: t0}
	sites := NewSites(store, textExtractor{}, clock, &fakeCanceller{}, &eventLog{})
	return NewExchange(store, files, clock, &eventLog{}), store, files, sites
}

func TestExportImportRoundTrip(t *testing.T) {
	t.Parallel()
	x, _, files, sites := newExchangeRig()
	s := domain.NewSite("Shop", "https://example.com/item")
	s.Mode = domain.ModeSelector
	s.Extraction.Selectors = []string{"#main"}
	s.Ignore.ExcludePatterns = []string{`\d+ left`}
	_, _ = sites.Add(bg, s, false)
	if n, err := x.Export(bg, "out.webvigil.json", nil); n != 1 || err != nil {
		t.Fatalf("export: %d %v", n, err)
	}
	y, store2, files2, _ := newExchangeRig()
	files2.data["in"] = files.data["out.webvigil.json"]
	p, err := y.PreviewImport(bg, "in")
	if err != nil || len(p.Plan) != 1 || p.Plan[0].Action != domain.ImportAdd {
		t.Fatalf("preview: %+v %v", p, err)
	}
	if _, err := y.ConfirmImport(bg, p.Plan, 0); err != nil {
		t.Fatal(err)
	}
	got := store2.sites[store2.order[0]]
	want, _ := s.Normalised()
	if got.BaselineKey() != want.BaselineKey() || got.Name != want.Name || got.Interval != want.Interval {
		t.Errorf("round trip: got %+v", got)
	}
	if files2.limits[0] != domain.MaxImportBytes+1 {
		t.Errorf("read limit = %d", files2.limits[0])
	}
}

func TestExportSelected(t *testing.T) {
	t.Parallel()
	x, _, files, sites := newExchangeRig()
	a, _ := sites.Add(bg, domain.NewSite("A", "https://a.example/"), false)
	_, _ = sites.Add(bg, domain.NewSite("B", "https://b.example/"), false)
	if n, err := x.Export(bg, "p", []domain.SiteID{a}); n != 1 || err != nil || strings.Contains(string(files.data["p"]), "b.example") {
		t.Errorf("selected export: %d %v", n, err)
	}
}

func TestExportErrors(t *testing.T) {
	t.Parallel()
	x, store, files, _ := newExchangeRig()
	files.writeErr = errBoom
	if _, err := x.Export(bg, "p", nil); !errors.Is(err, errBoom) {
		t.Errorf("write: %v", err)
	}
	store.fail["ListSites"] = true
	if _, err := x.Export(bg, "p", nil); !errors.Is(err, errBoom) {
		t.Errorf("list: %v", err)
	}
}

func TestPreviewImportErrors(t *testing.T) {
	t.Parallel()
	x, store, files, _ := newExchangeRig()
	files.readErr = errBoom
	if _, err := x.PreviewImport(bg, "p"); !errors.Is(err, errBoom) {
		t.Errorf("read: %v", err)
	}
	files.readErr = nil
	files.data["p"] = []byte(`{"format":"nope"}`)
	if _, err := x.PreviewImport(bg, "p"); !errors.Is(err, domain.ErrImportFormat) {
		t.Errorf("format: %v", err)
	}
	files.data["p"] = []byte(`{"format":"WebVigilSites","schemaVersion":1,"sites":[]}`)
	store.fail["ListSites"] = true
	if _, err := x.PreviewImport(bg, "p"); !errors.Is(err, errBoom) {
		t.Errorf("list: %v", err)
	}
}

func TestConfirmImportSummary(t *testing.T) {
	t.Parallel()
	x, store, _, sites := newExchangeRig()
	keep, _ := sites.Add(bg, domain.NewSite("Keep", "https://keep.example/"), false)
	upd, _ := sites.Add(bg, domain.NewSite("Upd", "https://upd.example/"), false)
	store.states[upd] = domain.SiteState{SiteID: upd, BaselineHash: "h"}
	changed := domain.NewSite("Upd2", "https://upd.example/")
	changed.Mode = domain.ModeRaw
	same := domain.NewSite("Keep2", "https://keep.example/")
	store.states[keep] = domain.SiteState{SiteID: keep, BaselineHash: "k"}
	plan := []domain.ImportPlanItem{
		{Index: 1, Site: domain.NewSite("New", "https://new.example/"), Action: domain.ImportAdd},
		{Index: 2, Site: changed, Duplicate: true, ExistingID: upd, Action: domain.ImportUpdate},
		{Index: 3, Site: same, Duplicate: true, ExistingID: keep, Action: domain.ImportUpdate},
		{Index: 4, Site: same, Duplicate: true, ExistingID: keep, Action: domain.ImportSkip},
		{Index: 5, Site: same, Duplicate: true, ExistingID: keep, Action: domain.ImportSeparate},
	}
	sum, err := x.ConfirmImport(bg, plan, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sum != (domain.ImportSummary{Imported: 2, Updated: 2, Skipped: 1, Invalid: 2}) {
		t.Errorf("summary = %+v", sum)
	}
	if store.states[upd].HasBaseline() || store.states[keep].BaselineHash != "k" || store.sites[upd].Name != "Upd2" {
		t.Errorf("update baselines wrong: %+v %+v", store.states[upd], store.states[keep])
	}
	if len(store.sites) != 4 {
		t.Errorf("sites = %d", len(store.sites))
	}
}

func TestConfirmImportRefusals(t *testing.T) {
	t.Parallel()
	x, store, _, _ := newExchangeRig()
	bad := []domain.ImportPlanItem{{Index: 1, Action: domain.ImportUpdate}}
	if _, err := x.ConfirmImport(bg, bad, 0); !errors.Is(err, domain.ErrInvalidSite) {
		t.Errorf("disallowed action: %v", err)
	}
	upd := []domain.ImportPlanItem{{Index: 1, Duplicate: true, ExistingID: 99, Action: domain.ImportUpdate}}
	if _, err := x.ConfirmImport(bg, upd, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing site: %v", err)
	}
	store.states[1] = domain.SiteState{}
	store.sites[1] = domain.Site{ID: 1}
	store.fail["GetState"] = true
	upd[0].ExistingID = 1
	if _, err := x.ConfirmImport(bg, upd, 0); !errors.Is(err, errBoom) {
		t.Errorf("state: %v", err)
	}
	store.fail = map[string]bool{"ApplyImport": true}
	if _, err := x.ConfirmImport(bg, nil, 0); !errors.Is(err, errBoom) {
		t.Errorf("apply: %v", err)
	}
}

func TestImportedSiteBaselinesWithoutAlarm(t *testing.T) {
	t.Parallel()
	x, store, _, _ := newExchangeRig()
	plan := []domain.ImportPlanItem{{Index: 1, Site: domain.NewSite("Imp", siteURL), Action: domain.ImportAdd}}
	if _, err := x.ConfirmImport(bg, plan, 0); err != nil {
		t.Fatal(err)
	}
	id := store.order[0]
	fetch := &fakeFetcher{}
	fetch.set(siteURL, "content")
	alerts := &alertLog{}
	c := NewChecker(store, store, fetch, textExtractor{}, &fakeClock{now: t0}, alerts)
	res, err := c.Check(bg, id, false)
	if err != nil || res.Change != domain.ChangeBaseline || len(alerts.changes) != 0 {
		t.Errorf("imported first check = %+v %v", res, err)
	}
}
