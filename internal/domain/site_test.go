package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewSiteDefaults(t *testing.T) {
	t.Parallel()
	s := NewSite("n", "https://example.com")
	if !s.Enabled || s.Interval != time.Hour || s.Timeout != 30*time.Second || s.Mode != ModeText ||
		!s.Notify.Sound || !s.Notify.Visual || !s.Notify.RequireAck || !s.Ignore.CollapseWhitespace ||
		s.FailureThreshold != DefaultFailureThreshold {
		t.Errorf("defaults wrong: %+v", s)
	}
}

func TestNormalisedTrimsAndNames(t *testing.T) {
	t.Parallel()
	s := NewSite("  ", "HTTPS://Example.com")
	s.Ignore.ExcludeSelectors = []string{" .ad ", ""}
	s.Ignore.ExcludePatterns = []string{"", " x "}
	s.Mode = ModeFields
	s.Extraction.Fields = []FieldRule{{Name: " Title ", Selector: " h1 ", Attribute: " "}, {}}
	got, err := s.Normalised()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "example.com" || got.URL != "https://example.com/" {
		t.Errorf("name/url = %q %q", got.Name, got.URL)
	}
	if len(got.Ignore.ExcludeSelectors) != 1 || got.Ignore.ExcludeSelectors[0] != ".ad" {
		t.Errorf("selectors = %q", got.Ignore.ExcludeSelectors)
	}
	if len(got.Ignore.ExcludePatterns) != 1 || got.Ignore.ExcludePatterns[0] != " x " {
		t.Errorf("patterns = %q", got.Ignore.ExcludePatterns)
	}
	if len(got.Extraction.Fields) != 1 || got.Extraction.Fields[0] != (FieldRule{Name: "Title", Selector: "h1"}) {
		t.Errorf("fields = %+v", got.Extraction.Fields)
	}
}

func TestNormalisedRefuses(t *testing.T) {
	t.Parallel()
	base := NewSite("n", "https://example.com/")
	many := make([]string, MaxRules+1)
	for i := range many {
		many[i] = "x"
	}
	cases := map[string]func(*Site){
		"long name":      func(s *Site) { s.Name = strings.Repeat("n", MaxNameLength+1) },
		"short interval": func(s *Site) { s.Interval = time.Minute },
		"long interval":  func(s *Site) { s.Interval = MaxInterval + time.Second },
		"short timeout":  func(s *Site) { s.Timeout = time.Second },
		"long timeout":   func(s *Site) { s.Timeout = MaxTimeout + time.Second },
		"zero threshold": func(s *Site) { s.FailureThreshold = 0 },
		"huge threshold": func(s *Site) { s.FailureThreshold = MaxFailureThreshold + 1 },
		"too many rules": func(s *Site) { s.Ignore.ExcludeSelectors = many },
		"bad mode":       func(s *Site) { s.Mode = "telepathy" },
		"no selectors":   func(s *Site) { s.Mode = ModeSelector },
		"no fields":      func(s *Site) { s.Mode = ModeFields },
		"field no sel": func(s *Site) {
			s.Mode = ModeFields
			s.Extraction.Fields = []FieldRule{{Name: "a"}}
		},
		"dup field": func(s *Site) {
			s.Mode = ModeFields
			s.Extraction.Fields = []FieldRule{{Name: "a", Selector: "h1"}, {Name: "a", Selector: "h2"}}
		},
	}
	for name, mutate := range cases {
		s := base
		mutate(&s)
		if _, err := s.Normalised(); !errors.Is(err, ErrInvalidSite) {
			t.Errorf("%s: error = %v, want ErrInvalidSite", name, err)
		}
	}
	bad := base
	bad.Ignore.ExcludePatterns = []string{"("}
	if _, err := bad.Normalised(); !errors.Is(err, ErrInvalidPattern) {
		t.Errorf("bad pattern: %v", err)
	}
	badURL := base
	badURL.URL = "ftp://x"
	if _, err := badURL.Normalised(); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("bad url: %v", err)
	}
}

func TestNormalisedAcceptsEachMode(t *testing.T) {
	t.Parallel()
	for _, m := range Modes {
		s := NewSite("n", "https://example.com/")
		s.Mode = m
		s.Extraction.Selectors = []string{"main"}
		s.Extraction.Fields = []FieldRule{{Name: "t", Selector: "h1"}}
		if _, err := s.Normalised(); err != nil {
			t.Errorf("mode %s: %v", m, err)
		}
	}
}

func TestBaselineKey(t *testing.T) {
	t.Parallel()
	a := NewSite("a", "https://example.com/")
	b := a
	b.Name, b.Interval, b.Notify.Sound, b.Enabled = "b", 2*time.Hour, false, false
	if a.BaselineKey() != b.BaselineKey() {
		t.Error("cosmetic change altered the baseline key")
	}
	empty := a
	empty.Extraction.Selectors = []string{}
	empty.Extraction.Fields = []FieldRule{}
	empty.Ignore.ExcludeSelectors = []string{}
	empty.Ignore.ExcludePatterns = []string{}
	if a.BaselineKey() != empty.BaselineKey() {
		t.Error("nil and empty rule lists must give the same key")
	}
	for name, mutate := range map[string]func(*Site){
		"url":      func(s *Site) { s.URL = "https://example.com/x" },
		"mode":     func(s *Site) { s.Mode = ModeRaw },
		"selector": func(s *Site) { s.Extraction.Selectors = []string{"main"} },
		"pattern":  func(s *Site) { s.Ignore.ExcludePatterns = []string{"x"} },
		"fold":     func(s *Site) { s.Ignore.CaseInsensitive = true },
	} {
		c := a
		mutate(&c)
		if c.BaselineKey() == a.BaselineKey() {
			t.Errorf("%s change did not alter the baseline key", name)
		}
	}
}
