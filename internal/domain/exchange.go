package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Exchange format identity and limits (REQUIREMENTS.md FR-010 to FR-018).
const (
	ExchangeFormat        = "WebVigilSites"
	ExchangeSchemaVersion = 1
	ExchangeExtension     = ".webvigil.json"
	MaxImportBytes        = 1 << 20
	MaxImportSites        = 500
)

// ExchangeDocument is the portable sites file. It carries definitions only: no snapshots,
// history, hashes, validators or state.
type ExchangeDocument struct {
	Format        string         `json:"format"`
	SchemaVersion int            `json:"schemaVersion"`
	ExportedAt    string         `json:"exportedAt"`
	Sites         []ExchangeSite `json:"sites"`
}

// ExchangeSite is one definition on the wire. Optional members are pointers so that an absent
// value takes the specification default rather than Go's zero value.
type ExchangeSite struct {
	Name             string                 `json:"name"`
	URL              string                 `json:"url"`
	Enabled          *bool                  `json:"enabled,omitempty"`
	IntervalSeconds  *int64                 `json:"intervalSeconds,omitempty"`
	TimeoutSeconds   *int64                 `json:"timeoutSeconds,omitempty"`
	DetectionMode    string                 `json:"detectionMode,omitempty"`
	ExtractionConfig ExchangeExtraction     `json:"extractionConfig"`
	IgnoreRules      ExchangeIgnore         `json:"ignoreRules"`
	Notifications    *ExchangeNotifications `json:"notifications,omitempty"`
	FailureThreshold *int                   `json:"failureThreshold,omitempty"`
}

// ExchangeExtraction carries selectors and field rules.
type ExchangeExtraction struct {
	Selectors []string        `json:"selectors"`
	Fields    []ExchangeField `json:"fields"`
}

// ExchangeField is one structured field rule.
type ExchangeField struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
}

// ExchangeIgnore carries the ignore rules.
type ExchangeIgnore struct {
	ExcludeSelectors   []string `json:"excludeSelectors"`
	ExcludePatterns    []string `json:"excludePatterns"`
	CaseInsensitive    bool     `json:"caseInsensitive"`
	CollapseWhitespace *bool    `json:"collapseWhitespace,omitempty"`
}

// ExchangeNotifications carries the notification preferences.
type ExchangeNotifications struct {
	Sound                  *bool `json:"sound,omitempty"`
	Visual                 *bool `json:"visual,omitempty"`
	RequireAcknowledgement *bool `json:"requireAcknowledgement,omitempty"`
}

// BuildExport renders sites as an exchange document stamped with at.
func BuildExport(sites []Site, at time.Time) ExchangeDocument {
	doc := ExchangeDocument{
		Format:        ExchangeFormat,
		SchemaVersion: ExchangeSchemaVersion,
		ExportedAt:    at.UTC().Format(time.RFC3339),
		Sites:         make([]ExchangeSite, 0, len(sites)),
	}
	for _, s := range sites {
		fields := make([]ExchangeField, 0, len(s.Extraction.Fields))
		for _, f := range s.Extraction.Fields {
			fields = append(fields, ExchangeField(f))
		}
		doc.Sites = append(doc.Sites, ExchangeSite{
			Name:             s.Name,
			URL:              s.URL,
			Enabled:          ptr(s.Enabled),
			IntervalSeconds:  ptr(int64(s.Interval / time.Second)),
			TimeoutSeconds:   ptr(int64(s.Timeout / time.Second)),
			DetectionMode:    string(s.Mode),
			ExtractionConfig: ExchangeExtraction{Selectors: nonNil(s.Extraction.Selectors), Fields: fields},
			IgnoreRules: ExchangeIgnore{
				ExcludeSelectors:   nonNil(s.Ignore.ExcludeSelectors),
				ExcludePatterns:    nonNil(s.Ignore.ExcludePatterns),
				CaseInsensitive:    s.Ignore.CaseInsensitive,
				CollapseWhitespace: ptr(s.Ignore.CollapseWhitespace),
			},
			Notifications: &ExchangeNotifications{
				Sound:                  ptr(s.Notify.Sound),
				Visual:                 ptr(s.Notify.Visual),
				RequireAcknowledgement: ptr(s.Notify.RequireAck),
			},
			FailureThreshold: ptr(s.FailureThreshold),
		})
	}
	return doc
}

// MarshalExport serialises a document as indented JSON with a trailing newline. The document
// holds only strings, numbers, booleans and slices of them, which always marshal.
func MarshalExport(doc ExchangeDocument) []byte {
	b, _ := json.MarshalIndent(doc, "", "  ")
	return append(b, '\n')
}

// ImportCandidate is a valid definition from an import file with its position in the file.
type ImportCandidate struct {
	Index int
	Site  Site
}

// InvalidEntry is a definition that failed validation, with its position and reason.
type InvalidEntry struct {
	Index  int
	Name   string
	URL    string
	Reason string
}

// ImportFile is a parsed import file: the valid candidates and the refused entries.
type ImportFile struct {
	Candidates []ImportCandidate
	Invalid    []InvalidEntry
}

// ParseImport validates an import file as a whole and then entry by entry. A whole-file problem
// refuses everything; an entry problem refuses only that entry.
func ParseImport(data []byte) (ImportFile, error) {
	if len(data) > MaxImportBytes {
		return ImportFile{}, fmt.Errorf("%w: %d bytes, the limit is %d", ErrImportTooLarge, len(data), MaxImportBytes)
	}
	var doc ExchangeDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return ImportFile{}, fmt.Errorf("%w: %v", ErrImportMalformed, err)
	}
	if doc.Format != ExchangeFormat {
		return ImportFile{}, fmt.Errorf("%w: the format is %q, expected %q", ErrImportFormat, doc.Format, ExchangeFormat)
	}
	if doc.SchemaVersion != ExchangeSchemaVersion {
		return ImportFile{}, fmt.Errorf("%w: the file is version %d, this WebVigil reads version %d",
			ErrImportSchema, doc.SchemaVersion, ExchangeSchemaVersion)
	}
	if len(doc.Sites) > MaxImportSites {
		return ImportFile{}, fmt.Errorf("%w: %d sites, the limit is %d", ErrImportTooMany, len(doc.Sites), MaxImportSites)
	}
	var out ImportFile
	for i, es := range doc.Sites {
		site, err := es.toSite().Normalised()
		if err != nil {
			out.Invalid = append(out.Invalid, InvalidEntry{Index: i + 1, Name: es.Name, URL: es.URL, Reason: err.Error()})
			continue
		}
		out.Candidates = append(out.Candidates, ImportCandidate{Index: i + 1, Site: site})
	}
	return out, nil
}

func (es ExchangeSite) toSite() Site {
	s := NewSite(es.Name, es.URL)
	s.Enabled = deref(es.Enabled, s.Enabled)
	if es.IntervalSeconds != nil {
		s.Interval = time.Duration(*es.IntervalSeconds) * time.Second
	}
	if es.TimeoutSeconds != nil {
		s.Timeout = time.Duration(*es.TimeoutSeconds) * time.Second
	}
	if es.DetectionMode != "" {
		s.Mode = DetectionMode(es.DetectionMode)
	}
	s.Extraction.Selectors = es.ExtractionConfig.Selectors
	for _, f := range es.ExtractionConfig.Fields {
		s.Extraction.Fields = append(s.Extraction.Fields, FieldRule(f))
	}
	s.Ignore.ExcludeSelectors = es.IgnoreRules.ExcludeSelectors
	s.Ignore.ExcludePatterns = es.IgnoreRules.ExcludePatterns
	s.Ignore.CaseInsensitive = es.IgnoreRules.CaseInsensitive
	s.Ignore.CollapseWhitespace = deref(es.IgnoreRules.CollapseWhitespace, s.Ignore.CollapseWhitespace)
	if n := es.Notifications; n != nil {
		s.Notify.Sound = deref(n.Sound, s.Notify.Sound)
		s.Notify.Visual = deref(n.Visual, s.Notify.Visual)
		s.Notify.RequireAck = deref(n.RequireAcknowledgement, s.Notify.RequireAck)
	}
	s.FailureThreshold = deref(es.FailureThreshold, s.FailureThreshold)
	return s
}

func ptr[T any](v T) *T { return &v }

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
