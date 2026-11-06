package domain

import (
	"errors"
	"testing"
)

func TestLineEndings(t *testing.T) {
	t.Parallel()
	got, err := Normalise("\n\na  \r\nb\t\rc\n\n", IgnoreRules{})
	if err != nil || got != "a\nb\nc" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestIgnoreRules(t *testing.T) {
	t.Parallel()
	rules := IgnoreRules{
		ExcludePatterns:    []string{`Delivery by \w+`, `(?m)^Session \d+$`},
		CaseInsensitive:    true,
		CollapseWhitespace: true,
	}
	got, err := Normalise("  Title   Here \n\nDelivery by Tuesday\nSession 42\n In Stock ", rules)
	if err != nil || got != "title here\nin stock" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := Normalise("x", IgnoreRules{ExcludePatterns: []string{"("}}); !errors.Is(err, ErrInvalidPattern) {
		t.Errorf("bad pattern error = %v", err)
	}
}

func TestNormaliseContent(t *testing.T) {
	t.Parallel()
	c := Content{Text: " a ", Fields: []FieldValue{{Name: "t", Value: " X "}}}
	got, err := NormaliseContent(c, IgnoreRules{CollapseWhitespace: true, CaseInsensitive: true})
	if err != nil || got.Text != "a" || got.Fields[0].Value != "x" || got.Fields[0].Name != "t" {
		t.Errorf("got %+v, %v", got, err)
	}
	bad := IgnoreRules{ExcludePatterns: []string{"("}}
	if _, err := NormaliseContent(Content{}, bad); err == nil {
		t.Error("text pattern error not reported")
	}
}
