package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// FieldValue is one extracted structured field.
type FieldValue struct {
	Name  string
	Value string
}

// Content is what a check compares: free text; else a list of named fields.
type Content struct {
	Text   string
	Fields []FieldValue
}

// Separators for the canonical field rendering. They are control characters that cannot occur
// in normalised text, so no field value can forge a boundary.
const (
	fieldNameSep = "\x1f"
	fieldSep     = "\x1e"
)

// Canonical renders the content as one string for hashing.
func (c Content) Canonical() string {
	if len(c.Fields) == 0 {
		return c.Text
	}
	var b strings.Builder
	for _, f := range c.Fields {
		b.WriteString(f.Name)
		b.WriteString(fieldNameSep)
		b.WriteString(f.Value)
		b.WriteString(fieldSep)
	}
	return b.String()
}

// IsEmpty reports whether there is nothing to compare.
func (c Content) IsEmpty() bool {
	if strings.TrimSpace(c.Text) != "" {
		return false
	}
	for _, f := range c.Fields {
		if strings.TrimSpace(f.Value) != "" {
			return false
		}
	}
	return true
}

// Hash returns the hex SHA-256 of the canonical content.
func (c Content) Hash() string {
	sum := sha256.Sum256([]byte(c.Canonical()))
	return hex.EncodeToString(sum[:])
}

// ChangeKind is the comparison verdict of a successful check.
type ChangeKind string

const (
	ChangeBaseline  ChangeKind = "baseline"
	ChangeUnchanged ChangeKind = "unchanged"
	ChangeChanged   ChangeKind = "changed"
)

// Compare decides what a successful check means. The baseline always moves to the latest
// accepted content, so the same changed content seen twice compares unchanged the second time,
// while a return to any earlier version differs from the baseline and is a change.
func Compare(state SiteState, newHash string) ChangeKind {
	switch {
	case !state.HasBaseline():
		return ChangeBaseline
	case state.BaselineHash == newHash:
		return ChangeUnchanged
	default:
		return ChangeChanged
	}
}
