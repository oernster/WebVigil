package application

import (
	"errors"
	"testing"

	"webvigil/internal/domain"
)

func TestHistoryDetail(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "a\nb")
	r.check(t, "a\nc")
	h := NewHistory(r.store)
	changes, err := h.Changes(bg, r.id, 10)
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v %v", changes, err)
	}
	checks, err := h.Checks(bg, r.id, 1)
	if err != nil || len(checks) != 1 || checks[0].Change != domain.ChangeChanged {
		t.Errorf("checks = %+v %v", checks, err)
	}
	d, err := h.Detail(bg, changes[0].ID)
	if err != nil || len(d.Lines) != 3 || d.Lines[1].Op != domain.DiffRemoved || d.Lines[2].Op != domain.DiffAdded {
		t.Errorf("detail = %+v %v", d, err)
	}
}

func TestHistoryFieldDetail(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t, func(s *domain.Site) {
		s.Mode = domain.ModeFields
		s.Extraction.Fields = []domain.FieldRule{{Name: "Title", Selector: "h1"}}
	})
	r.check(t, "Title A")
	r.check(t, "Title B")
	h := NewHistory(r.store)
	changes, _ := h.Changes(bg, r.id, 10)
	d, err := h.Detail(bg, changes[0].ID)
	if err != nil || len(d.Fields) != 1 || d.Fields[0] != (domain.FieldChange{Name: "Title", Old: "A", New: "B"}) {
		t.Errorf("field detail = %+v %v", d, err)
	}
}

func TestHistoryDetailErrors(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.check(t, "a")
	r.check(t, "b")
	h := NewHistory(r.store)
	changes, _ := h.Changes(bg, r.id, 10)
	if _, err := h.Detail(bg, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing change: %v", err)
	}
	r.store.failAfter["GetSnapshot"] = r.store.calls["GetSnapshot"]
	if _, err := h.Detail(bg, changes[0].ID); !errors.Is(err, errBoom) {
		t.Errorf("first snapshot: %v", err)
	}
	r.store.failAfter["GetSnapshot"] = r.store.calls["GetSnapshot"] + 1
	if _, err := h.Detail(bg, changes[0].ID); !errors.Is(err, errBoom) {
		t.Errorf("second snapshot: %v", err)
	}
}
