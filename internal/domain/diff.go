package domain

import (
	"fmt"
	"strings"
)

// DiffOp marks a line in a diff.
type DiffOp string

const (
	DiffEqual   DiffOp = "equal"
	DiffAdded   DiffOp = "added"
	DiffRemoved DiffOp = "removed"
)

// DiffLine is one line of a line diff.
type DiffLine struct {
	Op   DiffOp
	Text string
}

// MaxDiffEdits bounds the Myers search. The trace keeps one V array per edit step, so memory
// grows with the square of this bound: at 1000 the worst case is about 16 MB. Beyond it the
// middle section is shown as removed then added in full; still correct, just coarser.
const MaxDiffEdits = 1000

// DiffLines returns a minimal line diff from a to b (Myers), after trimming the common prefix
// and suffix so the search runs over the changed middle only.
func DiffLines(a, b []string) []DiffLine {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	out := make([]DiffLine, 0, len(a)+len(b))
	for _, l := range a[:pre] {
		out = append(out, DiffLine{DiffEqual, l})
	}
	out = append(out, myers(a[pre:len(a)-suf], b[pre:len(b)-suf])...)
	for _, l := range a[len(a)-suf:] {
		out = append(out, DiffLine{DiffEqual, l})
	}
	return out
}

func myers(a, b []string) []DiffLine {
	n, m := len(a), len(b)
	maxD := min(n+m, MaxDiffEdits)
	offset := maxD + 1
	v := make([]int, 2*maxD+3)
	var trace [][]int
	for d := 0; d <= maxD; d++ {
		trace = append(trace, append([]int(nil), v...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return backtrack(trace, a, b, offset, d)
			}
		}
	}
	return replaceAll(a, b)
}

func backtrack(trace [][]int, a, b []string, offset, d int) []DiffLine {
	x, y := len(a), len(b)
	var rev []DiffLine
	for ; d > 0; d-- {
		v := trace[d]
		k := x - y
		var prevK int
		if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := v[offset+prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x, y = x-1, y-1
			rev = append(rev, DiffLine{DiffEqual, a[x]})
		}
		if x == prevX {
			y--
			rev = append(rev, DiffLine{DiffAdded, b[y]})
		} else {
			x--
			rev = append(rev, DiffLine{DiffRemoved, a[x]})
		}
	}
	// No leading snake remains at d = 0: DiffLines trims the common prefix before calling myers,
	// so the first lines of a and b always differ here.
	out := make([]DiffLine, len(rev))
	for i, l := range rev {
		out[len(rev)-1-i] = l
	}
	return out
}

func replaceAll(a, b []string) []DiffLine {
	out := make([]DiffLine, 0, len(a)+len(b))
	for _, l := range a {
		out = append(out, DiffLine{DiffRemoved, l})
	}
	for _, l := range b {
		out = append(out, DiffLine{DiffAdded, l})
	}
	return out
}

// SplitLines splits normalised text into lines; empty text has no lines.
func SplitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// FieldChange describes one structured field whose value differs.
type FieldChange struct {
	Name string
	Old  string
	New  string
}

// DiffFields lists the fields whose value differs, in the order of the newer content, followed
// by fields that disappeared.
func DiffFields(old, cur []FieldValue) []FieldChange {
	prev := map[string]string{}
	for _, f := range old {
		prev[f.Name] = f.Value
	}
	var out []FieldChange
	seen := map[string]bool{}
	for _, f := range cur {
		seen[f.Name] = true
		if p, ok := prev[f.Name]; !ok || p != f.Value {
			out = append(out, FieldChange{Name: f.Name, Old: prev[f.Name], New: f.Value})
		}
	}
	for _, f := range old {
		if !seen[f.Name] {
			out = append(out, FieldChange{Name: f.Name, Old: f.Value})
		}
	}
	return out
}

// Summarise gives a one-line description of a change for a notification.
func Summarise(old, cur Content) string {
	if len(old.Fields) > 0 || len(cur.Fields) > 0 {
		changes := DiffFields(old.Fields, cur.Fields)
		names := make([]string, len(changes))
		for i, c := range changes {
			names[i] = c.Name
		}
		return "Changed: " + strings.Join(names, ", ")
	}
	added, removed := 0, 0
	for _, l := range DiffLines(SplitLines(old.Text), SplitLines(cur.Text)) {
		switch l.Op {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		}
	}
	return fmt.Sprintf("%d %s added, %d removed", added, plural(added, "line", "lines"), removed)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
