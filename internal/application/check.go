package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"webvigil/internal/domain"
)

// Detail texts recorded on a successful check.
const (
	detailBaseline  = "Baseline established"
	detailUnchanged = "No change"
	detailNotMod    = "No change (not modified)"
)

// AlertSink receives what a check found that the user must be told about.
type AlertSink interface {
	ChangeDetected(site domain.Site, change domain.ChangeEvent)
	FailuresRepeated(site domain.Site, state domain.SiteState)
}

// Checker runs one check of one site: fetch, validate, extract, normalise, hash, compare,
// record and alert. Manual and scheduled checks both come here (FR-080).
type Checker struct {
	sites     SiteStore
	history   HistoryStore
	fetcher   Fetcher
	extractor Extractor
	clock     Clock
	alerts    AlertSink
}

// NewChecker wires a checker.
func NewChecker(s SiteStore, h HistoryStore, f Fetcher, e Extractor, c Clock, a AlertSink) *Checker {
	return &Checker{sites: s, history: h, fetcher: f, extractor: e, clock: c, alerts: a}
}

// CheckResult reports what one check concluded.
type CheckResult struct {
	Outcome  domain.Outcome
	Category domain.Category
	Change   domain.ChangeKind
}

// observation is the classified result of fetching and extracting, before comparison.
type observation struct {
	resp     FetchResponse
	category domain.Category
	detail   string
	content  domain.Content
	notMod   bool
}

// Check runs one check of the site. A cancelled check records nothing (FR-027).
func (c *Checker) Check(ctx context.Context, id domain.SiteID, manual bool) (CheckResult, error) {
	site, err := c.sites.GetSite(ctx, id)
	if err != nil {
		return CheckResult{}, err
	}
	state, err := c.sites.GetState(ctx, id)
	if err != nil {
		return CheckResult{}, err
	}
	at := c.clock.Now()
	obs := c.observe(ctx, site, state)
	if ctx.Err() != nil {
		return CheckResult{}, ErrCancelled
	}
	rec := domain.CheckRecord{
		SiteID: id, At: at, RequestURL: site.URL, FinalURL: obs.resp.FinalURL,
		HTTPStatus: obs.resp.HTTPStatus, Duration: obs.resp.Duration, ContentType: obs.resp.ContentType,
		Bytes: obs.resp.Bytes, Manual: manual,
	}
	facts := domain.CheckFacts{
		At: at, HTTPStatus: obs.resp.HTTPStatus, Category: obs.category, Detail: obs.detail,
		ETag: obs.resp.ETag, LastModified: obs.resp.LastModified, RetryAfter: obs.resp.RetryAfter,
	}
	if obs.category != domain.CategoryNone {
		return c.recordFailure(ctx, site, state, rec, facts)
	}
	return c.recordSuccess(ctx, site, state, rec, facts, obs)
}

// observe fetches and classifies. Validators are sent only when a baseline exists, so a 304
// can only ever confirm content WebVigil already holds (FR-024, FR-025).
func (c *Checker) observe(ctx context.Context, site domain.Site, state domain.SiteState) observation {
	req := FetchRequest{URL: site.URL, Timeout: site.Timeout}
	if state.HasBaseline() {
		req.ETag, req.LastModified = state.ETag, state.LastModified
	}
	resp := c.fetcher.Fetch(ctx, req)
	return c.classify(resp, site, state.HasBaseline())
}

func (c *Checker) classify(resp FetchResponse, site domain.Site, hasBaseline bool) observation {
	obs := observation{resp: resp}
	fail := func(cat domain.Category, extra string) observation {
		obs.category = cat
		obs.detail = domain.Describe(cat, resp.HTTPStatus)
		if extra != "" {
			obs.detail += ": " + extra
		}
		return obs
	}
	if resp.Failure != domain.CategoryNone {
		return fail(resp.Failure, resp.FailureDetail)
	}
	if resp.HTTPStatus == http.StatusNotModified {
		if !hasBaseline {
			return fail(domain.CategoryUnexpectedStructure, "304 Not Modified without a baseline")
		}
		obs.notMod = true
		obs.detail = detailNotMod
		return obs
	}
	if name, ok := domain.DetectChallenge(string(resp.Body)); ok {
		return fail(domain.CategoryChallenge, name)
	}
	if !domain.IsSuccessStatus(resp.HTTPStatus) {
		return fail(domain.CategoryForStatus(resp.HTTPStatus), "")
	}
	content, err := c.extractor.Extract(resp.Body, resp.ContentType, site)
	switch {
	case errors.Is(err, domain.ErrSelectorMissing):
		return fail(domain.CategorySelectorMissing, trimSentinel(err, domain.ErrSelectorMissing))
	case err != nil:
		return fail(domain.CategoryParseFailure, trimSentinel(err, domain.ErrParseFailure))
	}
	content, err = domain.NormaliseContent(content, site.Ignore)
	if err != nil {
		return fail(domain.CategoryInvalidContent, err.Error())
	}
	if content.IsEmpty() {
		return fail(domain.CategoryInvalidContent, "")
	}
	obs.content = content
	return obs
}

// trimSentinel removes the sentinel's own text from a wrapped error, leaving the specific part.
func trimSentinel(err, sentinel error) string {
	msg := err.Error()
	prefix := sentinel.Error() + ": "
	if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
		return msg[len(prefix):]
	}
	return msg
}

func (c *Checker) recordFailure(ctx context.Context, site domain.Site, state domain.SiteState,
	rec domain.CheckRecord, facts domain.CheckFacts) (CheckResult, error) {
	next := state.AfterFailure(facts, site.Interval)
	warn := next.ShouldWarnFailures(site.FailureThreshold)
	next.FailureWarned = next.FailureWarned || warn
	rec.Outcome, rec.Category, rec.Detail = domain.OutcomeFailure, facts.Category, facts.Detail
	if _, err := c.history.RecordCheck(ctx, CheckWrite{Record: rec, State: next}); err != nil {
		return CheckResult{}, err
	}
	if warn {
		c.alerts.FailuresRepeated(site, next)
	}
	return CheckResult{Outcome: domain.OutcomeFailure, Category: facts.Category}, nil
}

func (c *Checker) recordSuccess(ctx context.Context, site domain.Site, state domain.SiteState,
	rec domain.CheckRecord, facts domain.CheckFacts, obs observation) (CheckResult, error) {
	hash := state.BaselineHash
	kind := domain.ChangeUnchanged
	if !obs.notMod {
		hash = obs.content.Hash()
		kind = domain.Compare(state, hash)
	}
	w := CheckWrite{Record: rec}
	switch kind {
	case domain.ChangeBaseline:
		facts.Detail = detailBaseline
		w.Snapshot = &domain.Snapshot{SiteID: site.ID, Hash: hash, Content: obs.content, CapturedAt: facts.At}
	case domain.ChangeChanged:
		prev, err := c.history.GetSnapshot(ctx, state.BaselineSnapshotID)
		if err != nil {
			return CheckResult{}, fmt.Errorf("read baseline snapshot: %w", err)
		}
		summary := domain.Summarise(prev.Content, obs.content)
		facts.Detail = summary
		w.Snapshot = &domain.Snapshot{SiteID: site.ID, Hash: hash, Content: obs.content, CapturedAt: facts.At}
		w.Change = &domain.ChangeEvent{
			SiteID: site.ID, PrevSnapshotID: state.BaselineSnapshotID, PrevHash: state.BaselineHash,
			NewHash: hash, Summary: summary, DetectedAt: facts.At,
		}
		w.SetPending = site.Notify.RequireAck
	default:
		if facts.Detail == "" {
			facts.Detail = detailUnchanged
		}
	}
	w.State = state.AfterSuccess(facts, site.Interval, kind, hash)
	w.Record.Outcome, w.Record.Detail, w.Record.Hash, w.Record.Change = domain.OutcomeSuccess, facts.Detail, hash, kind
	ids, err := c.history.RecordCheck(ctx, w)
	if err != nil {
		return CheckResult{}, err
	}
	if w.Change != nil {
		w.Change.ID, w.Change.NewSnapshotID = ids.ChangeID, ids.SnapshotID
		c.alerts.ChangeDetected(site, *w.Change)
	}
	return CheckResult{Outcome: domain.OutcomeSuccess, Change: kind}, nil
}

// Preview is the result of extracting a site once without storing anything (FR-057).
type Preview struct {
	HTTPStatus int
	Category   domain.Category
	Detail     string
	Content    domain.Content
}

// Preview fetches and extracts a definition that may not be saved yet.
func (c *Checker) Preview(ctx context.Context, site domain.Site) (Preview, error) {
	site, err := site.Normalised()
	if err != nil {
		return Preview{}, err
	}
	if err := c.extractor.Validate(site); err != nil {
		return Preview{}, err
	}
	obs := c.observe(ctx, site, domain.SiteState{})
	return Preview{HTTPStatus: obs.resp.HTTPStatus, Category: obs.category, Detail: obs.detail, Content: obs.content}, nil
}
