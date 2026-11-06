package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"webvigil/internal/domain"
)

const siteURL = "https://example.com/"

type checkRig struct {
	store   *memStore
	fetch   *fakeFetcher
	clock   *fakeClock
	alerts  *alertLog
	checker *Checker
	id      domain.SiteID
}

func newCheckRig(t *testing.T, mutate ...func(*domain.Site)) *checkRig {
	t.Helper()
	r := &checkRig{store: newMemStore(), fetch: &fakeFetcher{}, clock: &fakeClock{now: t0}, alerts: &alertLog{}}
	r.checker = NewChecker(r.store, r.store, r.fetch, textExtractor{}, r.clock, r.alerts)
	site, _ := domain.NewSite("Example", siteURL).Normalised()
	for _, m := range mutate {
		m(&site)
	}
	r.id, _ = r.store.AddSite(context.Background(), site, domain.SiteState{NextDueAt: t0})
	return r
}

func (r *checkRig) check(t *testing.T, body string) CheckResult {
	t.Helper()
	r.fetch.set(siteURL, body)
	r.clock.advance(time.Hour)
	res, err := r.checker.Check(context.Background(), r.id, false)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (r *checkRig) state() domain.SiteState { return r.store.states[r.id] }

func TestFirstSuccessIsBaseline(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	res := r.check(t, "hello")
	st := r.state()
	if res.Change != domain.ChangeBaseline || st.Light() != domain.LightGreen || len(r.alerts.changes) != 0 ||
		st.BaselineSnapshotID == 0 || st.LastDetail != detailBaseline {
		t.Errorf("baseline wrong: %+v %+v", res, st)
	}
}

func TestUnchangedRaisesNothing(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "hello")
	res := r.check(t, "hello")
	if res.Change != domain.ChangeUnchanged || len(r.alerts.changes) != 0 || r.state().LastDetail != detailUnchanged {
		t.Errorf("unchanged wrong: %+v", res)
	}
}

func TestChangeRaisesAlarm(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "hello")
	before := r.state().BaselineSnapshotID
	res := r.check(t, "hello\nworld")
	st := r.state()
	if res.Change != domain.ChangeChanged || len(r.alerts.changes) != 1 || st.Light() != domain.LightAmber {
		t.Fatalf("change wrong: %+v %+v", res, st)
	}
	ch := r.alerts.changes[0]
	if ch.ID == 0 || ch.PrevSnapshotID != before || ch.NewSnapshotID != st.BaselineSnapshotID ||
		ch.PrevHash == ch.NewHash || ch.Summary != "1 line added, 0 removed" || st.PendingChangeID != ch.ID {
		t.Errorf("change event wrong: %+v", ch)
	}
}

func TestChangeWithoutAcknowledgementLeavesNoPendingAlert(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t, func(s *domain.Site) { s.Notify.RequireAck = false })
	r.check(t, "a")
	r.check(t, "b")
	if r.state().HasPendingAlert() || len(r.alerts.changes) != 1 || r.state().Light() != domain.LightGreen {
		t.Errorf("state = %+v", r.state())
	}
}

func TestRepeatedContentRaisesOnce(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "a")
	r.check(t, "b")
	r.check(t, "b")
	r.fetch.set(siteURL, "b")
	if _, err := r.checker.Check(context.Background(), r.id, true); err != nil {
		t.Fatal(err)
	}
	if len(r.alerts.changes) != 1 {
		t.Errorf("alarms = %d, want 1", len(r.alerts.changes))
	}
}

func TestRevertIsAChange(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "a")
	r.check(t, "b")
	if res := r.check(t, "a"); res.Change != domain.ChangeChanged || len(r.alerts.changes) != 2 {
		t.Errorf("revert not a change: %+v", res)
	}
}

func TestFailureKeepsBaseline(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "a")
	base := r.state()
	r.fetch.resp[siteURL] = FetchResponse{HTTPStatus: http.StatusServiceUnavailable, Body: []byte("down"), RetryAfter: 2 * time.Hour}
	r.clock.advance(time.Hour)
	res, err := r.checker.Check(context.Background(), r.id, false)
	st := r.state()
	if err != nil || res.Category != domain.CategoryHTTP503 || st.BaselineHash != base.BaselineHash ||
		st.Light() != domain.LightRed || st.LastHTTPStatus != 503 || st.LastDetail != "HTTP 503: The server is unavailable" ||
		!st.NextDueAt.Equal(r.clock.Now().Add(2*time.Hour)) {
		t.Errorf("failure state wrong: %+v %v", st, err)
	}
	if res := r.check(t, "a"); res.Change != domain.ChangeUnchanged {
		t.Errorf("the failure was treated as a change: %+v", res)
	}
}

func TestClassification(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		resp FetchResponse
		ext  error
		want domain.Category
	}{
		"network":   {FetchResponse{Failure: domain.CategoryDNS, FailureDetail: "no such host"}, nil, domain.CategoryDNS},
		"challenge": {FetchResponse{HTTPStatus: 200, Body: []byte("/errors/validateCaptcha")}, nil, domain.CategoryChallenge},
		"chal 503":  {FetchResponse{HTTPStatus: 503, Body: []byte("Just a moment...</title>")}, nil, domain.CategoryHTTP503},
		"404":       {FetchResponse{HTTPStatus: 404}, nil, domain.CategoryHTTP404},
		"selector": {FetchResponse{HTTPStatus: 200, Body: []byte("x")},
			fmt.Errorf("%w: Title (#t)", domain.ErrSelectorMissing), domain.CategorySelectorMissing},
		"parse": {FetchResponse{HTTPStatus: 200, Body: []byte("x")},
			fmt.Errorf("%w: bad", domain.ErrParseFailure), domain.CategoryParseFailure},
		"other": {FetchResponse{HTTPStatus: 200, Body: []byte("x")}, errBoom, domain.CategoryParseFailure},
	}
	for name, c := range cases {
		r := newCheckRig(t)
		r.checker.extractor = textExtractor{err: c.ext}
		r.fetch.resp = map[string]FetchResponse{siteURL: c.resp}
		res, err := r.checker.Check(context.Background(), r.id, false)
		if err != nil || res.Category != c.want {
			t.Errorf("%s: category = %q, %v; want %q", name, res.Category, err, c.want)
		}
	}
	r := newCheckRig(t)
	r.checker.extractor = textExtractor{err: fmt.Errorf("%w: Title (#t)", domain.ErrSelectorMissing)}
	r.fetch.set(siteURL, "x")
	_, _ = r.checker.Check(context.Background(), r.id, false)
	if got := r.state().LastDetail; got != "A required selector matched nothing: Title (#t)" {
		t.Errorf("detail = %q", got)
	}
}

func TestChallengeOnOrdinaryStatus(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.fetch.resp = map[string]FetchResponse{siteURL: {HTTPStatus: 503, Body: []byte(`<form action="/errors/validateCaptcha">`)}}
	res, _ := r.checker.Check(context.Background(), r.id, false)
	if res.Category != domain.CategoryChallenge || r.state().LastHTTPStatus != 503 {
		t.Errorf("challenge on 503 = %+v", res)
	}
}

func TestEmptyContentIsInvalid(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	if res := r.check(t, "   \n "); res.Category != domain.CategoryInvalidContent {
		t.Errorf("empty content = %+v", res)
	}
	bad := newCheckRig(t, func(s *domain.Site) { s.Ignore.ExcludePatterns = []string{"("} })
	if res := bad.check(t, "x"); res.Category != domain.CategoryInvalidContent {
		t.Errorf("stored bad pattern = %+v", res)
	}
}

func TestNotModifiedWithBaselineIsUnchanged(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.fetch.resp = map[string]FetchResponse{siteURL: {HTTPStatus: 200, Body: []byte("a"), ETag: `"v1"`, LastModified: "Fri"}}
	if _, err := r.checker.Check(context.Background(), r.id, false); err != nil {
		t.Fatal(err)
	}
	r.fetch.resp[siteURL] = FetchResponse{HTTPStatus: http.StatusNotModified}
	res, _ := r.checker.Check(context.Background(), r.id, false)
	last := r.fetch.reqs[len(r.fetch.reqs)-1]
	if last.ETag != `"v1"` || last.LastModified != "Fri" || res.Change != domain.ChangeUnchanged ||
		r.state().LastDetail != detailNotMod || len(r.alerts.changes) != 0 {
		t.Errorf("304 handling wrong: req %+v res %+v", last, res)
	}
	if r.fetch.reqs[0].ETag != "" {
		t.Error("validators sent without a baseline")
	}
}

func TestNotModifiedWithoutBaselineFails(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.fetch.resp = map[string]FetchResponse{siteURL: {HTTPStatus: http.StatusNotModified}}
	if res, _ := r.checker.Check(context.Background(), r.id, false); res.Category != domain.CategoryUnexpectedStructure {
		t.Errorf("304 without baseline = %+v", res)
	}
}

func TestFailureThresholdWarnsOnce(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t, func(s *domain.Site) { s.FailureThreshold = 2 })
	r.fetch.resp = map[string]FetchResponse{siteURL: {HTTPStatus: 500}}
	for range 4 {
		_, _ = r.checker.Check(context.Background(), r.id, false)
	}
	if len(r.alerts.failures) != 1 || r.alerts.failures[0].ConsecutiveFailures != 2 {
		t.Errorf("failure warnings = %+v", r.alerts.failures)
	}
	r.check(t, "ok")
	r.fetch.resp[siteURL] = FetchResponse{HTTPStatus: 500}
	for range 2 {
		_, _ = r.checker.Check(context.Background(), r.id, false)
	}
	if len(r.alerts.failures) != 2 {
		t.Errorf("a new run of failures did not warn again: %d", len(r.alerts.failures))
	}
}

func TestCheckCancelledRecordsNothing(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	ctx, cancel := context.WithCancel(context.Background())
	r.fetch.hook = func(context.Context, FetchRequest) FetchResponse {
		cancel()
		return FetchResponse{Failure: domain.CategoryCancelled}
	}
	if _, err := r.checker.Check(ctx, r.id, false); !errors.Is(err, ErrCancelled) || len(r.store.checks) != 0 {
		t.Errorf("cancel: %v, %d records", err, len(r.store.checks))
	}
}

func TestCheckStoreErrors(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"GetSite", "GetState", "RecordCheck"} {
		r := newCheckRig(t)
		r.fetch.set(siteURL, "x")
		r.store.fail[method] = true
		if _, err := r.checker.Check(context.Background(), r.id, false); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", method, err)
		}
	}
	r := newCheckRig(t)
	r.fetch.resp = map[string]FetchResponse{siteURL: {HTTPStatus: 500}}
	r.store.fail["RecordCheck"] = true
	if _, err := r.checker.Check(context.Background(), r.id, false); !errors.Is(err, errBoom) {
		t.Errorf("failure record: %v", err)
	}
	r = newCheckRig(t)
	r.check(t, "a")
	r.store.fail["GetSnapshot"] = true
	r.fetch.set(siteURL, "b")
	if _, err := r.checker.Check(context.Background(), r.id, false); !errors.Is(err, errBoom) {
		t.Errorf("snapshot read: %v", err)
	}
}

func TestPreviewStoresNothing(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.fetch.set("https://example.org/", " Hello  World ")
	p, err := r.checker.Preview(context.Background(), domain.NewSite("", "https://EXAMPLE.org"))
	if err != nil || p.Content.Text != "Hello World" || p.HTTPStatus != 200 || len(r.store.checks) != 0 {
		t.Errorf("preview = %+v, %v", p, err)
	}
	if _, err := r.checker.Preview(context.Background(), domain.NewSite("", "ftp://x")); err == nil {
		t.Error("bad URL previewed")
	}
	r.checker.extractor = textExtractor{validateErr: errBoom}
	if _, err := r.checker.Preview(context.Background(), domain.NewSite("", siteURL)); !errors.Is(err, errBoom) {
		t.Errorf("selector validation: %v", err)
	}
}

func TestTrimSentinel(t *testing.T) {
	t.Parallel()
	if got := trimSentinel(domain.ErrParseFailure, domain.ErrParseFailure); got != domain.ErrParseFailure.Error() {
		t.Errorf("bare sentinel = %q", got)
	}
}
