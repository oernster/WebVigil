package extract

import (
	"errors"
	"strings"
	"testing"

	"webvigil/internal/domain"
)

const page = `<!doctype html><html><head><title>T</title><style>.x{}</style></head><body>
<header class="ad">Buy now</header>
<h1 id="productTitle">  Evergreen Book </h1>
<div id="availability"><span>In stock</span></div>
<img id="cover" src="/img/cover.jpg">
<p>First <b>para</b></p><script>var s = "secret";</script>
<div hidden>hidden one</div><div aria-hidden="true">hidden two</div><div style="display: none">hidden three</div>
<ul><li>one</li><li>two</li></ul><!-- comment -->
</body></html>`

func site(mode domain.DetectionMode) domain.Site {
	s := domain.NewSite("x", "https://example.com/")
	s.Mode = mode
	return s
}

func norm(t *testing.T, c domain.Content) string {
	t.Helper()
	n, err := domain.NormaliseContent(c, domain.IgnoreRules{CollapseWhitespace: true})
	if err != nil {
		t.Fatal(err)
	}
	return n.Text
}

func TestRawMode(t *testing.T) {
	t.Parallel()
	c, err := New().Extract([]byte(page), "text/html", site(domain.ModeRaw))
	if err != nil || c.Text != page {
		t.Errorf("raw = %v", err)
	}
}

func TestTextMode(t *testing.T) {
	t.Parallel()
	c, err := New().Extract([]byte(page), "text/html", site(domain.ModeText))
	if err != nil {
		t.Fatal(err)
	}
	got := norm(t, c)
	want := "Buy now\nEvergreen Book\nIn stock\nFirst para\none\ntwo"
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	for _, leak := range []string{"secret", "hidden", "comment", ".x{}"} {
		if strings.Contains(got, leak) {
			t.Errorf("text leaks %q", leak)
		}
	}
	plain, _ := New().Extract([]byte("<b>not html</b>"), "text/plain", site(domain.ModeText))
	if plain.Text != "<b>not html</b>" {
		t.Errorf("plain text was parsed as HTML: %q", plain.Text)
	}
}

func TestExcludedSelectors(t *testing.T) {
	t.Parallel()
	s := site(domain.ModeText)
	s.Ignore.ExcludeSelectors = []string{".ad", "ul"}
	c, _ := New().Extract([]byte(page), "", s)
	if got := norm(t, c); strings.Contains(got, "Buy now") || strings.Contains(got, "one") {
		t.Errorf("excluded content survived: %q", got)
	}
	s.Ignore.ExcludeSelectors = []string{"[["}
	if _, err := New().Extract([]byte(page), "", s); !errors.Is(err, domain.ErrParseFailure) {
		t.Errorf("bad excluded selector: %v", err)
	}
}

func TestSelectorMode(t *testing.T) {
	t.Parallel()
	s := site(domain.ModeSelector)
	s.Extraction.Selectors = []string{"#productTitle", "li"}
	c, err := New().Extract([]byte(page), "text/html", s)
	if err != nil || norm(t, c) != "Evergreen Book\none\ntwo" {
		t.Errorf("selector = %q, %v", c.Text, err)
	}
	s.Extraction.Selectors = []string{"#missing"}
	if _, err := New().Extract([]byte(page), "text/html", s); !errors.Is(err, domain.ErrSelectorMissing) {
		t.Errorf("missing selector: %v", err)
	}
	s.Extraction.Selectors = []string{"[["}
	if _, err := New().Extract([]byte(page), "text/html", s); !errors.Is(err, domain.ErrParseFailure) {
		t.Errorf("bad selector: %v", err)
	}
}

func TestFieldMode(t *testing.T) {
	t.Parallel()
	s := site(domain.ModeFields)
	s.Extraction.Fields = []domain.FieldRule{
		{Name: "Title", Selector: "#productTitle"},
		{Name: "Cover", Selector: "#cover", Attribute: "SRC"},
		{Name: "Stock", Selector: "#availability"},
		{Name: "Edition", Selector: "#edition", Optional: true},
		{Name: "Alt", Selector: "#cover", Attribute: "alt"},
	}
	c, err := New().Extract([]byte(page), "text/html", s)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := domain.NormaliseContent(c, domain.IgnoreRules{CollapseWhitespace: true})
	want := []domain.FieldValue{{Name: "Title", Value: "Evergreen Book"}, {Name: "Cover", Value: "/img/cover.jpg"},
		{Name: "Stock", Value: "In stock"}, {Name: "Edition"}, {Name: "Alt"}}
	for i, f := range want {
		if n.Fields[i] != f {
			t.Errorf("field %d = %+v, want %+v", i, n.Fields[i], f)
		}
	}
}

func TestMissingSelector(t *testing.T) {
	t.Parallel()
	s := site(domain.ModeFields)
	s.Extraction.Fields = []domain.FieldRule{{Name: "Price", Selector: "#price"}}
	_, err := New().Extract([]byte(page), "text/html", s)
	if !errors.Is(err, domain.ErrSelectorMissing) || !strings.Contains(err.Error(), "Price (#price)") {
		t.Errorf("missing field: %v", err)
	}
	s.Extraction.Fields = []domain.FieldRule{{Name: "Bad", Selector: "[["}}
	if _, err := New().Extract([]byte(page), "text/html", s); !errors.Is(err, domain.ErrParseFailure) {
		t.Errorf("bad field selector: %v", err)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	s := site(domain.ModeFields)
	s.Extraction.Fields = []domain.FieldRule{{Name: "a", Selector: "#ok"}}
	s.Ignore.ExcludeSelectors = []string{".ad"}
	if err := New().Validate(s); err != nil {
		t.Errorf("valid selectors refused: %v", err)
	}
	s.Extraction.Fields[0].Selector = "div[[["
	if err := New().Validate(s); !errors.Is(err, domain.ErrInvalidSite) {
		t.Errorf("bad selector accepted: %v", err)
	}
}
