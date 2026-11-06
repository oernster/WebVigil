package domain

import "time"

// Snapshot is one accepted version of a site's content.
type Snapshot struct {
	ID         int64
	SiteID     SiteID
	Hash       string
	Content    Content
	CapturedAt time.Time
}

// ChangeEvent records that content moved from one snapshot to another.
type ChangeEvent struct {
	ID             int64
	SiteID         SiteID
	PrevSnapshotID int64
	NewSnapshotID  int64
	PrevHash       string
	NewHash        string
	Summary        string
	DetectedAt     time.Time
	AcknowledgedAt time.Time // zero while pending
}

// CheckRecord is one row of check history.
type CheckRecord struct {
	ID          int64
	SiteID      SiteID
	At          time.Time
	RequestURL  string
	FinalURL    string
	HTTPStatus  int
	Duration    time.Duration
	ContentType string
	Bytes       int64
	Outcome     Outcome
	Category    Category
	Detail      string
	Hash        string
	Change      ChangeKind // empty on failure
	Manual      bool
}
