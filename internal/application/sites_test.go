package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"webvigil/internal/domain"
)

type fakeCanceller struct {
	cancelled []domain.SiteID
	running   map[domain.SiteID]bool
}

func (f *fakeCanceller) Cancel(id domain.SiteID)       { f.cancelled = append(f.cancelled, id) }
func (f *fakeCanceller) Running(id domain.SiteID) bool { return f.running[id] }

type sitesRig struct {
	store  *memStore
	clock  *fakeClock
	engine *fakeCanceller
	events *eventLog
	sites  *Sites
}

func newSitesRig() *sitesRig {
	r := &sitesRig{store: newMemStore(), clock: &fakeClock{now: t0}, engine: &fakeCanceller{running: map[domain.SiteID]bool{}}, events: &eventLog{}}
	r.sites = NewSites(r.store, textExtractor{}, r.clock, r.engine, r.events)
	return r
}

var bg = context.Background()

func TestAddSiteStoresAndIsDueNow(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	id, err := r.sites.Add(bg, domain.NewSite("", "https://Example.com/a"), false)
	if err != nil {
		t.Fatal(err)
	}
	r.engine.running[id] = true
	views, err := r.sites.List(bg)
	if err != nil || len(views) != 1 {
		t.Fatalf("list: %v %v", views, err)
	}
	v := views[0]
	if v.Light != domain.LightGrey || !v.State.NextDueAt.Equal(t0) || v.Site.URL != "https://example.com/a" ||
		v.Site.Name != "example.com" || !v.Running {
		t.Errorf("view = %+v", v)
	}
	if got, _ := r.sites.Get(bg, id); got.ID != id {
		t.Error("Get wrong")
	}
}

func TestAddRefusals(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	if _, err := r.sites.Add(bg, domain.NewSite("", "ftp://x"), false); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("bad url: %v", err)
	}
	r.sites.extractor = textExtractor{validateErr: errBoom}
	if _, err := r.sites.Add(bg, domain.NewSite("", siteURL), false); !errors.Is(err, errBoom) {
		t.Errorf("bad selector: %v", err)
	}
	r = newSitesRig()
	r.store.fail["ListSites"] = true
	if _, err := r.sites.Add(bg, domain.NewSite("", siteURL), false); !errors.Is(err, errBoom) {
		t.Errorf("list error: %v", err)
	}
	r = newSitesRig()
	r.store.fail["AddSite"] = true
	if _, err := r.sites.Add(bg, domain.NewSite("", siteURL), false); !errors.Is(err, errBoom) {
		t.Errorf("add error: %v", err)
	}
}

func TestAddDuplicateNeedsConfirmation(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	_, _ = r.sites.Add(bg, domain.NewSite("One", siteURL), false)
	if _, err := r.sites.Add(bg, domain.NewSite("Two", "https://EXAMPLE.com:443/#x"), false); !errors.Is(err, ErrDuplicate) {
		t.Errorf("duplicate = %v", err)
	}
	if _, err := r.sites.Add(bg, domain.NewSite("Two", siteURL), true); err != nil {
		t.Errorf("confirmed duplicate refused: %v", err)
	}
}

func TestEditKeepsBaselineForCosmeticChange(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	id, _ := r.sites.Add(bg, domain.NewSite("One", siteURL), false)
	r.store.states[id] = domain.SiteState{SiteID: id, BaselineHash: "h", PendingChangeID: 4, LastCheckedAt: t0}
	site, _ := r.sites.Get(bg, id)
	site.Name, site.Interval, site.Notify.Sound = "Renamed", 2*time.Hour, false
	if err := r.sites.Update(bg, site, false, false); err != nil {
		t.Fatal(err)
	}
	st := r.store.states[id]
	if st.BaselineHash != "h" || st.PendingChangeID != 4 || !st.NextDueAt.Equal(t0.Add(2*time.Hour)) ||
		r.store.sites[id].Name != "Renamed" {
		t.Errorf("cosmetic edit altered state: %+v", st)
	}
}

func TestEditBaselineKeyResetsBaseline(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	id, _ := r.sites.Add(bg, domain.NewSite("One", siteURL), false)
	r.store.states[id] = domain.SiteState{SiteID: id, BaselineHash: "h", LastCheckedAt: t0}
	site, _ := r.sites.Get(bg, id)
	site.Mode = domain.ModeRaw
	if err := r.sites.Update(bg, site, false, false); !errors.Is(err, ErrRebaselineRequired) {
		t.Errorf("unconfirmed rebaseline = %v", err)
	}
	if r.store.states[id].BaselineHash != "h" {
		t.Error("refused edit changed state")
	}
	r.clock.advance(time.Minute)
	if err := r.sites.Update(bg, site, true, false); err != nil {
		t.Fatal(err)
	}
	st := r.store.states[id]
	if st.HasBaseline() || !st.NextDueAt.Equal(r.clock.Now()) || len(r.engine.cancelled) != 1 {
		t.Errorf("rebaseline state = %+v", st)
	}
}

func TestUpdateErrors(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"GetSite", "GetState", "UpdateSite", "ListSites"} {
		r := newSitesRig()
		id, _ := r.sites.Add(bg, domain.NewSite("One", siteURL), false)
		site, _ := r.sites.Get(bg, id)
		r.store.fail[method] = true
		if err := r.sites.Update(bg, site, false, false); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", method, err)
		}
	}
	r := newSitesRig()
	a, _ := r.sites.Add(bg, domain.NewSite("A", siteURL), false)
	_, _ = r.sites.Add(bg, domain.NewSite("B", "https://example.org/"), false)
	site, _ := r.sites.Get(bg, a)
	site.URL = "https://example.org/"
	if err := r.sites.Update(bg, site, true, false); !errors.Is(err, ErrDuplicate) {
		t.Errorf("edit into a duplicate = %v", err)
	}
	site.URL = "ftp://bad"
	if err := r.sites.Update(bg, site, true, false); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("bad url = %v", err)
	}
}

func TestDeleteSite(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	id, _ := r.sites.Add(bg, domain.NewSite("One", siteURL), false)
	if err := r.sites.Delete(bg, id); err != nil {
		t.Fatal(err)
	}
	if len(r.store.sites) != 0 || len(r.engine.cancelled) != 1 || r.engine.cancelled[0] != id {
		t.Error("delete did not cancel and remove")
	}
	r.store.fail["DeleteSite"] = true
	if err := r.sites.Delete(bg, id); !errors.Is(err, errBoom) {
		t.Errorf("delete error = %v", err)
	}
}

func TestSetEnabled(t *testing.T) {
	t.Parallel()
	r := newSitesRig()
	id, _ := r.sites.Add(bg, domain.NewSite("One", siteURL), false)
	if err := r.sites.SetEnabled(bg, id, false); err != nil || r.store.sites[id].Enabled {
		t.Errorf("disable: %v", err)
	}
	r.clock.advance(time.Hour)
	if err := r.sites.SetEnabled(bg, id, true); err != nil || !r.store.states[id].NextDueAt.Equal(r.clock.Now()) {
		t.Errorf("enable: %v %+v", err, r.store.states[id])
	}
	for _, method := range []string{"GetSite", "GetState", "UpdateSite"} {
		r.store.fail = map[string]bool{method: true}
		if err := r.sites.SetEnabled(bg, id, true); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", method, err)
		}
	}
}

func TestListErrors(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"ListSites", "ListStates"} {
		r := newSitesRig()
		r.store.fail[method] = true
		if _, err := r.sites.List(bg); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", method, err)
		}
	}
}
