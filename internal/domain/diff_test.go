package domain

import (
	"fmt"
	"reflect"
	"testing"
)

func render(d []DiffLine) []string {
	out := make([]string, len(d))
	for i, l := range d {
		out[i] = map[DiffOp]string{DiffEqual: " ", DiffAdded: "+", DiffRemoved: "-"}[l.Op] + l.Text
	}
	return out
}

func TestDiffLines(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b []string
		want []string
	}{
		{[]string{"a", "b", "c"}, []string{"a", "x", "c"}, []string{" a", "-b", "+x", " c"}},
		{nil, []string{"a"}, []string{"+a"}},
		{[]string{"a"}, nil, []string{"-a"}},
		{[]string{"a", "b"}, []string{"a", "b"}, []string{" a", " b"}},
		{[]string{"x", "a", "b", "y"}, []string{"a", "c", "b"}, []string{"-x", " a", "+c", " b", "-y"}},
		{[]string{"a", "b", "c", "a", "b", "b", "a"}, []string{"c", "b", "a", "b", "a", "c"},
			[]string{"-a", "-b", " c", "+b", " a", " b", "-b", " a", "+c"}},
	}
	for _, c := range cases {
		if got := render(DiffLines(c.a, c.b)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("DiffLines(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestDiffLinesFallsBackBeyondBound(t *testing.T) {
	t.Parallel()
	a := make([]string, MaxDiffEdits)
	b := make([]string, MaxDiffEdits)
	for i := range a {
		a[i], b[i] = fmt.Sprint("a", i), fmt.Sprint("b", i)
	}
	got := DiffLines(a, b)
	if len(got) != 2*MaxDiffEdits || got[0].Op != DiffRemoved || got[len(got)-1].Op != DiffAdded {
		t.Errorf("fallback wrong: %d lines", len(got))
	}
}

func TestSplitLines(t *testing.T) {
	t.Parallel()
	if SplitLines("") != nil || len(SplitLines("a\nb")) != 2 {
		t.Error("SplitLines wrong")
	}
}

func TestDiffFields(t *testing.T) {
	t.Parallel()
	old := []FieldValue{{"Title", "A"}, {"Stock", "In"}, {"Gone", "x"}}
	cur := []FieldValue{{"Title", "A"}, {"Stock", "Out"}, {"New", "y"}}
	want := []FieldChange{{"Stock", "In", "Out"}, {"New", "", "y"}, {"Gone", "x", ""}}
	if got := DiffFields(old, cur); !reflect.DeepEqual(got, want) {
		t.Errorf("DiffFields = %+v", got)
	}
}

func TestSummarise(t *testing.T) {
	t.Parallel()
	if got := Summarise(Content{Text: "a\nb"}, Content{Text: "a\nc\nd"}); got != "2 lines added, 1 removed" {
		t.Errorf("text summary = %q", got)
	}
	if got := Summarise(Content{Text: "a"}, Content{Text: "b"}); got != "1 line added, 1 removed" {
		t.Errorf("singular summary = %q", got)
	}
	f := Summarise(Content{Fields: []FieldValue{{"T", "a"}}}, Content{Fields: []FieldValue{{"T", "b"}}})
	if f != "Changed: T" {
		t.Errorf("field summary = %q", f)
	}
}
