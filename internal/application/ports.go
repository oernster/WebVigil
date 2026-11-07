// Package application holds WebVigil's use cases: one entry point per user-visible action, each
// depending only on the domain and on the interfaces declared here. Infrastructure implements
// the interfaces; the UI calls the services.
package application

import (
	"context"
	"errors"
	"time"

	"webvigil/internal/domain"
)

// Sentinel errors returned by the services.
var (
	ErrNotFound           = errors.New("not found")
	ErrDuplicate          = errors.New("a site with this address already exists")
	ErrRebaselineRequired = errors.New("this change needs a new baseline")
	ErrAlreadyRunning     = errors.New("a check of this site is already running")
	ErrCancelled          = errors.New("the check was cancelled")
	ErrStopped            = errors.New("monitoring has stopped")
)

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// SiteStore persists definitions and their state.
type SiteStore interface {
	ListSites(ctx context.Context) ([]domain.Site, error)
	GetSite(ctx context.Context, id domain.SiteID) (domain.Site, error)
	AddSite(ctx context.Context, site domain.Site, state domain.SiteState) (domain.SiteID, error)
	UpdateSite(ctx context.Context, site domain.Site, state domain.SiteState) error
	DeleteSite(ctx context.Context, id domain.SiteID) error
	ListStates(ctx context.Context) ([]domain.SiteState, error)
	GetState(ctx context.Context, id domain.SiteID) (domain.SiteState, error)
	ApplyImport(ctx context.Context, writes []ImportWrite) error
}

// ImportWrite is one change of a confirmed import. ID is zero for an addition.
type ImportWrite struct {
	Site  domain.Site
	State domain.SiteState
}

// CheckWrite is everything one check stores, written in a single transaction. When Snapshot is
// set the store inserts it and makes it the baseline; when Change is set its NewSnapshotID is
// that snapshot; when SetPending is set the state's pending alert becomes the new change.
type CheckWrite struct {
	Record     domain.CheckRecord
	State      domain.SiteState
	Snapshot   *domain.Snapshot
	Change     *domain.ChangeEvent
	SetPending bool
}

// CheckIDs are the identifiers assigned by RecordCheck.
type CheckIDs struct {
	SnapshotID int64
	ChangeID   int64
}

// HistoryStore persists checks, snapshots and change events.
type HistoryStore interface {
	RecordCheck(ctx context.Context, w CheckWrite) (CheckIDs, error)
	Acknowledge(ctx context.Context, id domain.SiteID, at time.Time) (int, error)
	History(ctx context.Context, id domain.SiteID, limit int) ([]domain.CheckRecord, error)
	Changes(ctx context.Context, id domain.SiteID, limit int) ([]domain.ChangeEvent, error)
	GetChange(ctx context.Context, id int64) (domain.ChangeEvent, error)
	GetSnapshot(ctx context.Context, id int64) (domain.Snapshot, error)
	PendingCount(ctx context.Context) (int, error)
}

// SettingsStore persists application settings.
type SettingsStore interface {
	LoadSettings(ctx context.Context) (domain.Settings, error)
	SaveSettings(ctx context.Context, s domain.Settings) error
}

// FetchRequest is one HTTP retrieval.
type FetchRequest struct {
	URL          string
	Timeout      time.Duration
	ETag         string
	LastModified string
}

// FetchResponse is what the fetcher observed. Failure is set when no usable response arrived
// (network, TLS, redirect, size or content-type problems); HTTPStatus is zero when no HTTP
// response arrived at all.
type FetchResponse struct {
	FinalURL      string
	HTTPStatus    int
	ContentType   string
	Body          []byte
	Bytes         int64
	Duration      time.Duration
	ETag          string
	LastModified  string
	RetryAfter    time.Duration
	Failure       domain.Category
	FailureDetail string
}

// Fetcher retrieves a URL.
type Fetcher interface {
	Fetch(ctx context.Context, req FetchRequest) FetchResponse
}

// Extractor turns a response body into content. Its errors wrap domain.ErrSelectorMissing or
// domain.ErrParseFailure.
type Extractor interface {
	Extract(body []byte, contentType string, site domain.Site) (domain.Content, error)
	Validate(site domain.Site) error
}

// Files reads and writes exchange files.
type Files interface {
	ReadLimited(path string, limit int64) ([]byte, error)
	WriteAtomic(path string, data []byte) error
}

// Sounder plays an alarm and returns when it has finished. An empty path means the built-in chime.
type Sounder interface {
	Play(path string) error
}

// ChangeNotice is what a change notification shows.
type ChangeNotice struct {
	SiteID   domain.SiteID
	ChangeID int64
	Name     string
	URL      string
	At       time.Time
	Summary  string
}

// FailureNotice is what a repeated-failure warning shows.
type FailureNotice struct {
	SiteID   domain.SiteID
	Name     string
	Failures int
	Detail   string
}

// Notifier shows visual notifications.
type Notifier interface {
	NotifyChange(n ChangeNotice)
	NotifyFailures(n FailureNotice)
}

// AutoStart controls the sign-in start entry. Enabled reports whether the entry is there now,
// whoever wrote it: the setup program writes the same entry.
type AutoStart interface {
	Set(enabled bool) error
	Enabled() (bool, error)
}

// EventKind names a state change the interface should react to.
type EventKind string

const (
	EventSitesChanged  EventKind = "sites"
	EventCheckStarted  EventKind = "check-started"
	EventCheckFinished EventKind = "check-finished"
	EventSettings      EventKind = "settings"
)

// Events tells the interface that something changed; it re-reads what it shows.
type Events interface {
	Emit(kind EventKind, id domain.SiteID)
}

// Logger records failures that have no other surface.
type Logger interface {
	Printf(format string, args ...any)
}
