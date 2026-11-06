package domain

import "time"

// Light is the traffic-light summary of a site.
type Light string

const (
	LightGrey  Light = "grey"
	LightGreen Light = "green"
	LightAmber Light = "amber"
	LightRed   Light = "red"
)

// SiteState is the persisted monitoring state of one site. Every fact is its own field so that,
// for example, a pending change survives a later failure (REQUIREMENTS.md FR-065, FR-070).
type SiteState struct {
	SiteID              SiteID
	LastOutcome         Outcome
	LastCategory        Category
	LastDetail          string
	LastHTTPStatus      int // zero when no HTTP response arrived
	LastCheckedAt       time.Time
	LastSuccessAt       time.Time
	LastChangeAt        time.Time
	BaselineSnapshotID  int64
	BaselineHash        string
	PendingChangeID     int64 // zero when no alert awaits acknowledgement
	ConsecutiveFailures int
	FailureWarned       bool
	ETag                string
	LastModified        string
	NextDueAt           time.Time
}

// HasBaseline reports whether accepted content exists to compare against.
func (s SiteState) HasBaseline() bool { return s.BaselineHash != "" }

// HasPendingAlert reports whether a change awaits acknowledgement.
func (s SiteState) HasPendingAlert() bool { return s.PendingChangeID != 0 }

// Light derives the status colour: Red for a current failure (it outranks everything), Grey
// with no baseline yet, Amber for an unacknowledged change and Green otherwise. The pending
// change is a separate field, so a Red site can still show its change badge.
func (s SiteState) Light() Light {
	switch {
	case s.LastOutcome == OutcomeFailure:
		return LightRed
	case !s.HasBaseline():
		return LightGrey
	case s.HasPendingAlert():
		return LightAmber
	default:
		return LightGreen
	}
}

// CheckFacts is what a check observed, independent of how it compared.
type CheckFacts struct {
	At           time.Time
	HTTPStatus   int
	Category     Category
	Detail       string
	ETag         string
	LastModified string
	RetryAfter   time.Duration
}

// AfterFailure returns the state following a failed check. The baseline and any pending
// change are kept; the failure count and the next due time advance.
func (s SiteState) AfterFailure(f CheckFacts, interval time.Duration) SiteState {
	s.LastOutcome = OutcomeFailure
	s.LastCategory = f.Category
	s.LastDetail = f.Detail
	s.LastHTTPStatus = f.HTTPStatus
	s.LastCheckedAt = f.At
	s.ConsecutiveFailures++
	s.NextDueAt = NextDue(f.At, interval, s.ConsecutiveFailures, f.RetryAfter)
	return s
}

// ShouldWarnFailures reports whether the failure count has just reached the site's threshold
// and no warning has been given for this run of failures.
func (s SiteState) ShouldWarnFailures(threshold int) bool {
	return !s.FailureWarned && s.ConsecutiveFailures >= threshold
}

// AfterSuccess returns the state following a successful check. For a baseline or a change the
// baseline hash moves to hash; the snapshot and change identifiers are assigned by storage in
// the same transaction that writes them.
func (s SiteState) AfterSuccess(f CheckFacts, interval time.Duration, kind ChangeKind, hash string) SiteState {
	if kind != ChangeUnchanged {
		s.BaselineHash = hash
	}
	s.LastOutcome = OutcomeSuccess
	s.LastCategory = CategoryNone
	s.LastDetail = f.Detail
	s.LastHTTPStatus = f.HTTPStatus
	s.LastCheckedAt = f.At
	s.LastSuccessAt = f.At
	s.ConsecutiveFailures = 0
	s.FailureWarned = false
	if f.ETag != "" || f.LastModified != "" {
		s.ETag, s.LastModified = f.ETag, f.LastModified
	}
	if kind == ChangeChanged {
		s.LastChangeAt = f.At
	}
	s.NextDueAt = NextDue(f.At, interval, 0, f.RetryAfter)
	return s
}

// WithoutBaseline returns the state with the baseline, validators and pending alert cleared,
// used when an edit makes the old content incomparable. History is not touched.
func (s SiteState) WithoutBaseline(now time.Time) SiteState {
	s.BaselineHash = ""
	s.BaselineSnapshotID = 0
	s.PendingChangeID = 0
	s.ETag, s.LastModified = "", ""
	s.LastOutcome, s.LastCategory, s.LastDetail = OutcomeNone, CategoryNone, ""
	s.ConsecutiveFailures, s.FailureWarned = 0, false
	s.NextDueAt = now
	return s
}
