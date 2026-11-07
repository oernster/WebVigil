package main

import (
	"encoding/json"
	"strings"
	"testing"

	"webvigil/internal/domain"
)

// TestSuggestedReadingCarriesTheWholeReading: the dialog receives every field and ignore rule of
// the reading for a product page; for any other page it receives an answer whose lists are empty
// rather than null, so the page never meets a null where it expects a list.
func TestSuggestedReadingCarriesTheWholeReading(t *testing.T) {
	t.Parallel()
	const product = "https://www.amazon.com/dp/0123456789"
	want, _ := domain.ReadingFor(product)
	got := (&App{}).SuggestedReading(product)
	if !got.Found || got.Mode != string(domain.ModeFields) || got.Name != want.Name || len(got.Fields) != len(want.Fields) ||
		len(got.ExcludeSelectors) != len(want.ExcludeSelectors) || len(got.ExcludePatterns) != len(want.ExcludePatterns) {
		t.Fatalf("product page: %+v", got)
	}
	for i, f := range want.Fields {
		if got.Fields[i] != FieldRuleDTO(f) {
			t.Errorf("field %d = %+v, want %+v", i, got.Fields[i], f)
		}
	}
	none, err := json.Marshal((&App{}).SuggestedReading("https://example.com/"))
	if err != nil {
		t.Fatal(err)
	}
	if s := string(none); !strings.Contains(s, `"found":false`) || strings.Contains(s, "null") {
		t.Errorf("other page: %s", s)
	}
}
