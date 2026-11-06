package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildExportCarriesDefinitionsOnly(t *testing.T) {
	t.Parallel()
	s := NewSite("Shop", "https://example.com/")
	s.Mode = ModeFields
	s.Extraction.Fields = []FieldRule{{Name: "Cover", Selector: "#img", Attribute: "src", Optional: true}}
	data := MarshalExport(BuildExport([]Site{s}, t0))
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatal(err)
	}
	if generic["format"] != ExchangeFormat || generic["schemaVersion"] != float64(1) ||
		generic["exportedAt"] != "2026-11-06T12:00:00Z" {
		t.Errorf("header wrong: %v", generic)
	}
	text := string(data)
	for _, forbidden := range []string{"hash", "Hash", "snapshot", "etag", "ETag", "history", "lastChecked"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("export carries %q", forbidden)
		}
	}
	if !strings.HasSuffix(text, "}\n") {
		t.Error("export lacks a trailing newline")
	}
}

func doc(sites string) []byte {
	return []byte(fmt.Sprintf(`{"format":"WebVigilSites","schemaVersion":1,"sites":[%s]}`, sites))
}

func TestParseImportRefuses(t *testing.T) {
	t.Parallel()
	many := strings.Repeat(`{"url":"https://example.com/"},`, MaxImportSites) + `{"url":"https://example.com/"}`
	cases := map[string]struct {
		data []byte
		want error
	}{
		"too large": {make([]byte, MaxImportBytes+1), ErrImportTooLarge},
		"malformed": {[]byte(`{"format":`), ErrImportMalformed},
		"format":    {[]byte(`{"format":"Other","schemaVersion":1}`), ErrImportFormat},
		"schema":    {[]byte(`{"format":"WebVigilSites","schemaVersion":2}`), ErrImportSchema},
		"too many":  {doc(many), ErrImportTooMany},
	}
	for name, c := range cases {
		if _, err := ParseImport(c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestParseImportListsInvalidEntries(t *testing.T) {
	t.Parallel()
	f, err := ParseImport(doc(`{"name":"ok","url":"https://example.com/"},{"name":"bad","url":"ftp://x"},` +
		`{"name":"slow","url":"https://example.org/","intervalSeconds":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Candidates) != 1 || f.Candidates[0].Index != 1 {
		t.Errorf("candidates = %+v", f.Candidates)
	}
	if len(f.Invalid) != 2 || f.Invalid[0].Index != 2 || f.Invalid[0].Name != "bad" ||
		!strings.Contains(f.Invalid[0].Reason, "http") || f.Invalid[1].Index != 3 {
		t.Errorf("invalid = %+v", f.Invalid)
	}
}

func TestParseImportAppliesDefaults(t *testing.T) {
	t.Parallel()
	f, err := ParseImport(doc(`{"url":"https://example.com/"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := f.Candidates[0].Site
	want, _ := NewSite("", "https://example.com/").Normalised()
	if got.BaselineKey() != want.BaselineKey() || got.Interval != want.Interval || got.Notify != want.Notify ||
		got.Enabled != want.Enabled || got.FailureThreshold != want.FailureThreshold {
		t.Errorf("defaults not applied: %+v", got)
	}
}

func TestExchangeRoundTrip(t *testing.T) {
	t.Parallel()
	s := NewSite("Shop", "https://example.com/item")
	s.Enabled = false
	s.Interval, s.Timeout, s.FailureThreshold = 2*time.Hour, 45*time.Second, 7
	s.Mode = ModeFields
	s.Extraction.Selectors = []string{"main"}
	s.Extraction.Fields = []FieldRule{{Name: "Cover", Selector: "#img", Attribute: "src", Optional: true}}
	s.Ignore = IgnoreRules{ExcludeSelectors: []string{".ad"}, ExcludePatterns: []string{`\d+ left`}, CaseInsensitive: true}
	s.Notify = NotificationPrefs{Sound: false, Visual: true, RequireAck: false}
	f, err := ParseImport(MarshalExport(BuildExport([]Site{s}, t0)))
	if err != nil || len(f.Candidates) != 1 {
		t.Fatalf("parse: %v %+v", err, f)
	}
	got := f.Candidates[0].Site
	if got.Name != s.Name || got.URL != s.URL || got.Enabled != s.Enabled || got.Interval != s.Interval ||
		got.Timeout != s.Timeout || got.FailureThreshold != s.FailureThreshold || got.Notify != s.Notify ||
		got.BaselineKey() != s.BaselineKey() {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, s)
	}
}

func TestPlanImportDefaultsToSkip(t *testing.T) {
	t.Parallel()
	existing := []Site{{ID: 9, URL: "https://example.com/"}, {ID: 10, URL: "https://example.com/"}}
	cands := []ImportCandidate{
		{Index: 1, Site: Site{URL: "https://example.com/"}},
		{Index: 2, Site: Site{URL: "https://new.example/"}},
		{Index: 3, Site: Site{URL: "https://new.example/"}},
	}
	plan := PlanImport(cands, existing)
	if !plan[0].Duplicate || plan[0].ExistingID != 9 || plan[0].Action != ImportSkip {
		t.Errorf("existing duplicate = %+v", plan[0])
	}
	if plan[1].Duplicate || plan[1].Action != ImportAdd {
		t.Errorf("new entry = %+v", plan[1])
	}
	if !plan[2].Duplicate || plan[2].ExistingID != 0 || plan[2].Action != ImportSkip {
		t.Errorf("in-file repeat = %+v", plan[2])
	}
}

func TestImportPlanItemAllowed(t *testing.T) {
	t.Parallel()
	fresh := ImportPlanItem{Index: 1}
	stored := ImportPlanItem{Index: 2, Duplicate: true, ExistingID: 4}
	repeat := ImportPlanItem{Index: 3, Duplicate: true}
	allowed := []struct {
		item   ImportPlanItem
		action ImportAction
		ok     bool
	}{
		{fresh, ImportAdd, true}, {fresh, ImportSkip, true}, {fresh, ImportSeparate, false}, {fresh, ImportUpdate, false},
		{stored, ImportAdd, false}, {stored, ImportUpdate, true}, {stored, ImportSeparate, true},
		{repeat, ImportUpdate, false}, {repeat, ImportSeparate, true}, {fresh, "explode", false},
	}
	for _, c := range allowed {
		if err := c.item.Allowed(c.action); (err == nil) != c.ok {
			t.Errorf("item %d %s: err = %v, want ok=%v", c.item.Index, c.action, err, c.ok)
		}
	}
}
