package main

import (
	"time"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

// The wire between Go and the page. Every type here has a twin interface in
// frontend/src/wire.ts; tests/structural/wire_test.go compares the two field by field.

// SiteDTO is one row of the main table.
type SiteDTO struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Enabled             bool   `json:"enabled"`
	Light               string `json:"light"`
	Running             bool   `json:"running"`
	Outcome             string `json:"outcome"`
	Category            string `json:"category"`
	HTTPStatus          int    `json:"httpStatus"`
	LastChecked         string `json:"lastChecked"`
	LastSuccess         string `json:"lastSuccess"`
	LastChange          string `json:"lastChange"`
	NextDue             string `json:"nextDue"`
	Details             string `json:"details"`
	HasBaseline         bool   `json:"hasBaseline"`
	PendingChangeID     int64  `json:"pendingChangeId"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	Mode                string `json:"mode"`
	IntervalMinutes     int    `json:"intervalMinutes"`
}

// FieldRuleDTO is one structured field rule.
type FieldRuleDTO struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Optional  bool   `json:"optional"`
}

// SiteFormDTO is a full definition as the Add and Edit dialogs hold it.
type SiteFormDTO struct {
	ID                 int64          `json:"id"`
	Name               string         `json:"name"`
	URL                string         `json:"url"`
	Enabled            bool           `json:"enabled"`
	IntervalMinutes    int            `json:"intervalMinutes"`
	TimeoutSeconds     int            `json:"timeoutSeconds"`
	Mode               string         `json:"mode"`
	Selectors          []string       `json:"selectors"`
	Fields             []FieldRuleDTO `json:"fields"`
	ExcludeSelectors   []string       `json:"excludeSelectors"`
	ExcludePatterns    []string       `json:"excludePatterns"`
	CaseInsensitive    bool           `json:"caseInsensitive"`
	CollapseWhitespace bool           `json:"collapseWhitespace"`
	Sound              bool           `json:"sound"`
	Visual             bool           `json:"visual"`
	RequireAck         bool           `json:"requireAck"`
	FailureThreshold   int            `json:"failureThreshold"`
}

// SettingsDTO carries the application settings.
type SettingsDTO struct {
	Theme            string `json:"theme"`
	Paused           bool   `json:"paused"`
	StartAtSignIn    bool   `json:"startAtSignIn"`
	SoundPath        string `json:"soundPath"`
	RepeatAlarm      bool   `json:"repeatAlarm"`
	ForegroundDialog bool   `json:"foregroundDialog"`
}

// AppStateDTO is what the page needs on load.
type AppStateDTO struct {
	Version  string      `json:"version"`
	Problem  string      `json:"problem"`
	LogPath  string      `json:"logPath"`
	Settings SettingsDTO `json:"settings"`
	Toasts   bool        `json:"toasts"`
	// Platform is "windows", "macos" or "linux"; only Windows has a tray to keep running in.
	Platform string `json:"platform"`
}

// CreditDTO names one open-source component and its licence for Help > About.
type CreditDTO struct {
	Name    string `json:"name"`
	Licence string `json:"licence"`
}

// AboutDTO is what Help > About shows. The icon itself is a bundled front-end asset.
type AboutDTO struct {
	Name      string      `json:"name"`
	Tagline   string      `json:"tagline"`
	Version   string      `json:"version"`
	Author    string      `json:"author"`
	Copyright string      `json:"copyright"`
	Licence   string      `json:"licence"`
	LogPath   string      `json:"logPath"`
	Credits   []CreditDTO `json:"credits"`
}

// UpdateStatusDTO is the outcome of one update check. Latest is empty when the release source could
// not be reached, in which case UpdateAvailable is always false. DownloadURL is the release asset
// for this OS (empty when the release carries none); PageURL is the release page the page falls
// back to.
type UpdateStatusDTO struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl"`
	PageURL         string `json:"pageUrl"`
}

// CheckDTO is one row of check history.
type CheckDTO struct {
	At          string `json:"at"`
	HTTPStatus  int    `json:"httpStatus"`
	DurationMs  int64  `json:"durationMs"`
	Outcome     string `json:"outcome"`
	Category    string `json:"category"`
	Detail      string `json:"detail"`
	Hash        string `json:"hash"`
	Change      string `json:"change"`
	Manual      bool   `json:"manual"`
	FinalURL    string `json:"finalUrl"`
	ContentType string `json:"contentType"`
	Bytes       int64  `json:"bytes"`
}

// ChangeDTO is one change event.
type ChangeDTO struct {
	ID             int64  `json:"id"`
	DetectedAt     string `json:"detectedAt"`
	AcknowledgedAt string `json:"acknowledgedAt"`
	Summary        string `json:"summary"`
	PrevHash       string `json:"prevHash"`
	NewHash        string `json:"newHash"`
}

// HistoryDTO is a site's recent checks and changes.
type HistoryDTO struct {
	Checks  []CheckDTO  `json:"checks"`
	Changes []ChangeDTO `json:"changes"`
}

// DiffLineDTO is one line of a diff.
type DiffLineDTO struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// FieldChangeDTO is one changed structured field.
type FieldChangeDTO struct {
	Name string `json:"name"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// ChangeDetailDTO is a change with its diff.
type ChangeDetailDTO struct {
	Change ChangeDTO        `json:"change"`
	Lines  []DiffLineDTO    `json:"lines"`
	Fields []FieldChangeDTO `json:"fields"`
}

// FieldValueDTO is one extracted field.
type FieldValueDTO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ReadingDTO is a ready-made reading the site dialog fills a definition in from.
type ReadingDTO struct {
	Found            bool           `json:"found"`
	Name             string         `json:"name"`
	Mode             string         `json:"mode"`
	Fields           []FieldRuleDTO `json:"fields"`
	ExcludeSelectors []string       `json:"excludeSelectors"`
	ExcludePatterns  []string       `json:"excludePatterns"`
}

func readingDTO(r domain.Reading) ReadingDTO {
	out := ReadingDTO{Found: true, Name: r.Name, Mode: string(r.Mode), Fields: []FieldRuleDTO{},
		ExcludeSelectors: orEmpty(r.ExcludeSelectors), ExcludePatterns: orEmpty(r.ExcludePatterns)}
	for _, f := range r.Fields {
		out.Fields = append(out.Fields, FieldRuleDTO(f))
	}
	return out
}

// PreviewDTO is a one-off extraction shown while editing.
type PreviewDTO struct {
	HTTPStatus int             `json:"httpStatus"`
	Category   string          `json:"category"`
	Detail     string          `json:"detail"`
	Text       string          `json:"text"`
	Fields     []FieldValueDTO `json:"fields"`
}

// ImportItemDTO is one planned import entry.
type ImportItemDTO struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Duplicate bool   `json:"duplicate"`
	CanUpdate bool   `json:"canUpdate"`
	Action    string `json:"action"`
}

// InvalidEntryDTO is a refused import entry.
type InvalidEntryDTO struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// ImportPreviewDTO is the preview shown before an import is confirmed.
type ImportPreviewDTO struct {
	Path    string            `json:"path"`
	Items   []ImportItemDTO   `json:"items"`
	Invalid []InvalidEntryDTO `json:"invalid"`
}

// ImportSummaryDTO counts a confirmed import.
type ImportSummaryDTO struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Invalid  int `json:"invalid"`
}

// ExportResultDTO reports an export.
type ExportResultDTO struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// stamp renders a time for the wire; the zero time is the empty string.
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func siteDTO(v application.SiteView) SiteDTO {
	st := v.State
	return SiteDTO{
		ID: int64(v.Site.ID), Name: v.Site.Name, URL: v.Site.URL, Enabled: v.Site.Enabled, Light: string(v.Light),
		Running: v.Running, Outcome: string(st.LastOutcome), Category: string(st.LastCategory),
		HTTPStatus: st.LastHTTPStatus, LastChecked: stamp(st.LastCheckedAt), LastSuccess: stamp(st.LastSuccessAt),
		LastChange: stamp(st.LastChangeAt), NextDue: stamp(st.NextDueAt), Details: st.LastDetail,
		HasBaseline: st.HasBaseline(), PendingChangeID: st.PendingChangeID, ConsecutiveFailures: st.ConsecutiveFailures,
		Mode: string(v.Site.Mode), IntervalMinutes: int(v.Site.Interval / time.Minute),
	}
}

func formDTO(s domain.Site) SiteFormDTO {
	f := SiteFormDTO{
		ID: int64(s.ID), Name: s.Name, URL: s.URL, Enabled: s.Enabled, IntervalMinutes: int(s.Interval / time.Minute),
		TimeoutSeconds: int(s.Timeout / time.Second), Mode: string(s.Mode), Selectors: orEmpty(s.Extraction.Selectors),
		Fields: []FieldRuleDTO{}, ExcludeSelectors: orEmpty(s.Ignore.ExcludeSelectors),
		ExcludePatterns: orEmpty(s.Ignore.ExcludePatterns), CaseInsensitive: s.Ignore.CaseInsensitive,
		CollapseWhitespace: s.Ignore.CollapseWhitespace, Sound: s.Notify.Sound, Visual: s.Notify.Visual,
		RequireAck: s.Notify.RequireAck, FailureThreshold: s.FailureThreshold,
	}
	for _, r := range s.Extraction.Fields {
		f.Fields = append(f.Fields, FieldRuleDTO(r))
	}
	return f
}

func (f SiteFormDTO) site() domain.Site {
	s := domain.Site{
		ID: domain.SiteID(f.ID), Name: f.Name, URL: f.URL, Enabled: f.Enabled,
		Interval: time.Duration(f.IntervalMinutes) * time.Minute, Timeout: time.Duration(f.TimeoutSeconds) * time.Second,
		Mode: domain.DetectionMode(f.Mode), Extraction: domain.ExtractionConfig{Selectors: f.Selectors},
		Ignore: domain.IgnoreRules{ExcludeSelectors: f.ExcludeSelectors, ExcludePatterns: f.ExcludePatterns,
			CaseInsensitive: f.CaseInsensitive, CollapseWhitespace: f.CollapseWhitespace},
		Notify:           domain.NotificationPrefs{Sound: f.Sound, Visual: f.Visual, RequireAck: f.RequireAck},
		FailureThreshold: f.FailureThreshold,
	}
	for _, r := range f.Fields {
		s.Extraction.Fields = append(s.Extraction.Fields, domain.FieldRule(r))
	}
	return s
}

func settingsDTO(s domain.Settings) SettingsDTO {
	return SettingsDTO{Theme: string(s.Theme), Paused: s.Paused, StartAtSignIn: s.StartAtSignIn,
		SoundPath: s.SoundPath, RepeatAlarm: s.RepeatAlarm, ForegroundDialog: s.ForegroundDialog}
}

func (d SettingsDTO) settings() domain.Settings {
	return domain.Settings{Theme: domain.Theme(d.Theme), Paused: d.Paused, StartAtSignIn: d.StartAtSignIn,
		SoundPath: d.SoundPath, RepeatAlarm: d.RepeatAlarm, ForegroundDialog: d.ForegroundDialog}
}

func changeDTO(c domain.ChangeEvent) ChangeDTO {
	return ChangeDTO{ID: c.ID, DetectedAt: stamp(c.DetectedAt), AcknowledgedAt: stamp(c.AcknowledgedAt),
		Summary: c.Summary, PrevHash: c.PrevHash, NewHash: c.NewHash}
}

func orEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
