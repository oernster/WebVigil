package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SiteID identifies a stored site. Zero means not yet stored.
type SiteID int64

// DetectionMode chooses how a response becomes comparable content.
type DetectionMode string

const (
	ModeRaw      DetectionMode = "raw"
	ModeText     DetectionMode = "extracted_text"
	ModeSelector DetectionMode = "css_selector"
	ModeFields   DetectionMode = "structured_fields"
)

// Modes lists every detection mode in the order the interface offers them.
var Modes = []DetectionMode{ModeText, ModeSelector, ModeFields, ModeRaw}

// Defaults and bounds for a site definition (REQUIREMENTS.md section 1.4).
const (
	DefaultInterval         = 60 * time.Minute
	MinInterval             = 5 * time.Minute
	MaxInterval             = 7 * 24 * time.Hour
	DefaultTimeout          = 30 * time.Second
	MinTimeout              = 5 * time.Second
	MaxTimeout              = 120 * time.Second
	DefaultFailureThreshold = 3
	MaxFailureThreshold     = 100
	MaxNameLength           = 200
	MaxRules                = 50
)

// FieldRule extracts one named value. With Attribute empty the element's text is used.
type FieldRule struct {
	Name      string
	Selector  string
	Attribute string
	Optional  bool
}

// ExtractionConfig holds the rules for the selector and field modes.
type ExtractionConfig struct {
	Selectors []string
	Fields    []FieldRule
}

// IgnoreRules remove volatile content before comparison.
type IgnoreRules struct {
	ExcludeSelectors   []string
	ExcludePatterns    []string
	CaseInsensitive    bool
	CollapseWhitespace bool
}

// NotificationPrefs says how a change on this site is announced.
type NotificationPrefs struct {
	Sound      bool
	Visual     bool
	RequireAck bool
}

// Site is one monitored definition.
type Site struct {
	ID               SiteID
	Name             string
	URL              string
	Enabled          bool
	Interval         time.Duration
	Timeout          time.Duration
	Mode             DetectionMode
	Extraction       ExtractionConfig
	Ignore           IgnoreRules
	Notify           NotificationPrefs
	FailureThreshold int
}

// NewSite returns a definition carrying every default from the specification.
func NewSite(name, rawURL string) Site {
	return Site{
		Name:             name,
		URL:              rawURL,
		Enabled:          true,
		Interval:         DefaultInterval,
		Timeout:          DefaultTimeout,
		Mode:             ModeText,
		Ignore:           IgnoreRules{CollapseWhitespace: true},
		Notify:           NotificationPrefs{Sound: true, Visual: true, RequireAck: true},
		FailureThreshold: DefaultFailureThreshold,
	}
}

// Normalised validates the definition and returns it with a canonical URL, trimmed text and
// no empty rules. An empty name takes the host name.
func (s Site) Normalised() (Site, error) {
	u, err := NormaliseURL(s.URL)
	if err != nil {
		return Site{}, err
	}
	s.URL = u
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		s.Name = HostOf(u)
	}
	s.Extraction.Selectors = trimmed(s.Extraction.Selectors)
	s.Ignore.ExcludeSelectors = trimmed(s.Ignore.ExcludeSelectors)
	s.Ignore.ExcludePatterns = nonEmpty(s.Ignore.ExcludePatterns)
	fields := make([]FieldRule, 0, len(s.Extraction.Fields))
	for _, f := range s.Extraction.Fields {
		f.Name, f.Selector, f.Attribute = strings.TrimSpace(f.Name), strings.TrimSpace(f.Selector), strings.TrimSpace(f.Attribute)
		if f.Name != "" || f.Selector != "" {
			fields = append(fields, f)
		}
	}
	s.Extraction.Fields = fields
	return s, s.validate()
}

func (s Site) validate() error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidSite}, a...)...)
	}
	switch {
	case len(s.Name) > MaxNameLength:
		return bad("the name is longer than %d characters", MaxNameLength)
	case s.Interval < MinInterval || s.Interval > MaxInterval:
		return bad("the interval must be between %s and %s", MinInterval, MaxInterval)
	case s.Timeout < MinTimeout || s.Timeout > MaxTimeout:
		return bad("the timeout must be between %s and %s", MinTimeout, MaxTimeout)
	case s.FailureThreshold < 1 || s.FailureThreshold > MaxFailureThreshold:
		return bad("the failure threshold must be between 1 and %d", MaxFailureThreshold)
	case len(s.Extraction.Selectors) > MaxRules || len(s.Extraction.Fields) > MaxRules ||
		len(s.Ignore.ExcludeSelectors) > MaxRules || len(s.Ignore.ExcludePatterns) > MaxRules:
		return bad("no rule list may hold more than %d entries", MaxRules)
	}
	if err := s.validateMode(); err != nil {
		return err
	}
	for _, p := range s.Ignore.ExcludePatterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("%w: %q: %v", ErrInvalidPattern, p, err)
		}
	}
	return nil
}

func (s Site) validateMode() error {
	switch s.Mode {
	case ModeRaw, ModeText:
		return nil
	case ModeSelector:
		if len(s.Extraction.Selectors) == 0 {
			return fmt.Errorf("%w: the CSS selector mode needs at least one selector", ErrInvalidSite)
		}
		return nil
	case ModeFields:
		if len(s.Extraction.Fields) == 0 {
			return fmt.Errorf("%w: the structured fields mode needs at least one field", ErrInvalidSite)
		}
		seen := map[string]bool{}
		for _, f := range s.Extraction.Fields {
			if f.Name == "" || f.Selector == "" {
				return fmt.Errorf("%w: every field needs a name and a selector", ErrInvalidSite)
			}
			if seen[f.Name] {
				return fmt.Errorf("%w: the field name %q is used twice", ErrInvalidSite, f.Name)
			}
			seen[f.Name] = true
		}
		return nil
	}
	return fmt.Errorf("%w: unknown detection mode %q", ErrInvalidSite, s.Mode)
}

// BaselineKey is a canonical rendering of the settings that decide what content means. Two
// definitions with different keys produce content that must not be compared.
func (s Site) BaselineKey() string {
	key := struct {
		URL      string
		Mode     DetectionMode
		Sel      []string
		Fields   []FieldRule
		ExSel    []string
		ExPat    []string
		Fold     bool
		Collapse bool
	}{s.URL, s.Mode, nonNil(s.Extraction.Selectors), nonNilFields(s.Extraction.Fields),
		nonNil(s.Ignore.ExcludeSelectors), nonNil(s.Ignore.ExcludePatterns),
		s.Ignore.CaseInsensitive, s.Ignore.CollapseWhitespace}
	b, _ := json.Marshal(key) // a struct of strings, bools and slices always marshals
	return string(b)
}

func trimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// nonEmpty drops empty patterns but keeps surrounding space, which can be meaningful in a regex.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func nonNilFields(in []FieldRule) []FieldRule {
	if in == nil {
		return []FieldRule{}
	}
	return in
}
