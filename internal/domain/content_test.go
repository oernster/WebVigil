package domain

import "testing"

func TestCanonicalHashIsStable(t *testing.T) {
	t.Parallel()
	const helloSHA256 = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := (Content{Text: "hello"}).Hash(); got != helloSHA256 {
		t.Errorf("hash = %s", got)
	}
	a := Content{Fields: []FieldValue{{"a", "b c"}}}
	b := Content{Fields: []FieldValue{{"a b", "c"}}}
	if a.Hash() == b.Hash() {
		t.Error("field boundary is forgeable")
	}
	if a.Canonical() == "" {
		t.Error("empty canonical for fields")
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	if !(Content{Text: " \n", Fields: []FieldValue{{"a", " "}}}).IsEmpty() {
		t.Error("blank content not empty")
	}
	if (Content{Text: "x"}).IsEmpty() || (Content{Fields: []FieldValue{{"a", "x"}}}).IsEmpty() {
		t.Error("content reported empty")
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	if Compare(SiteState{}, "h") != ChangeBaseline {
		t.Error("no baseline should give a baseline")
	}
	s := SiteState{BaselineHash: "h"}
	if Compare(s, "h") != ChangeUnchanged || Compare(s, "g") != ChangeChanged {
		t.Error("compare wrong")
	}
}
