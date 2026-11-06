package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"webvigil/internal/domain"
)

// fakeFetcher answers each URL from a table; else with a function.
type fakeFetcher struct {
	mu    sync.Mutex
	resp  map[string]FetchResponse
	reqs  []FetchRequest
	hook  func(ctx context.Context, req FetchRequest) FetchResponse
	calls int
}

func (f *fakeFetcher) Fetch(ctx context.Context, req FetchRequest) FetchResponse {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	f.calls++
	hook, r := f.hook, f.resp[req.URL]
	f.mu.Unlock()
	if hook != nil {
		return hook(ctx, req)
	}
	return r
}

func (f *fakeFetcher) set(url, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resp == nil {
		f.resp = map[string]FetchResponse{}
	}
	f.resp[url] = FetchResponse{FinalURL: url, HTTPStatus: 200, ContentType: "text/html", Body: []byte(body), Bytes: int64(len(body))}
}

// textExtractor returns the body as text; "FIELDS:" bodies become one field per line "k=v".
type textExtractor struct {
	err         error
	validateErr error
}

func (e textExtractor) Extract(body []byte, _ string, site domain.Site) (domain.Content, error) {
	if e.err != nil {
		return domain.Content{}, e.err
	}
	if site.Mode == domain.ModeFields {
		var fields []domain.FieldValue
		for _, line := range domain.SplitLines(string(body)) {
			var k, v string
			fmt.Sscanf(line, "%s %s", &k, &v)
			fields = append(fields, domain.FieldValue{Name: k, Value: v})
		}
		return domain.Content{Fields: fields}, nil
	}
	return domain.Content{Text: string(body)}, nil
}

func (e textExtractor) Validate(domain.Site) error { return e.validateErr }

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }

func (c *fakeClock) advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }

type eventLog struct {
	mu     sync.Mutex
	events []EventKind
}

func (e *eventLog) Emit(k EventKind, _ domain.SiteID) {
	e.mu.Lock()
	e.events = append(e.events, k)
	e.mu.Unlock()
}

type logSink struct {
	mu    sync.Mutex
	lines []string
}

func (l *logSink) Printf(f string, a ...any) {
	l.mu.Lock()
	l.lines = append(l.lines, fmt.Sprintf(f, a...))
	l.mu.Unlock()
}

func (l *logSink) count() int { l.mu.Lock(); defer l.mu.Unlock(); return len(l.lines) }

type alertLog struct {
	mu       sync.Mutex
	changes  []domain.ChangeEvent
	failures []domain.SiteState
}

func (a *alertLog) ChangeDetected(_ domain.Site, c domain.ChangeEvent) {
	a.mu.Lock()
	a.changes = append(a.changes, c)
	a.mu.Unlock()
}

func (a *alertLog) FailuresRepeated(_ domain.Site, s domain.SiteState) {
	a.mu.Lock()
	a.failures = append(a.failures, s)
	a.mu.Unlock()
}

type fixedSettings struct{ s domain.Settings }

func (f *fixedSettings) Current() domain.Settings { return f.s }
