package domain

import "testing"

// TestReadingForKnowsAmazonProductPagesOnly: every storefront's product address gets the Amazon
// reading, in either path form and after a title slug; other Amazon pages and other hosts get
// none.
func TestReadingForKnowsAmazonProductPagesOnly(t *testing.T) {
	t.Parallel()
	for _, u := range []string{
		"https://www.amazon.com/dp/0123456789",
		"https://amazon.de/dp/B012345678/",
		"https://www.amazon.com/Some-Book-Title/dp/0123456789/ref=sr_1_1?keywords=x",
		"https://www.amazon.co.jp/gp/product/B012345678",
		"  HTTPS://WWW.AMAZON.FR/DP/0123456789  ",
	} {
		r, ok := ReadingFor(u)
		if !ok || r.Mode != ModeFields || r.Name == "" {
			t.Errorf("%q: got %+v, %v; want the Amazon reading", u, r, ok)
		}
	}
	for _, u := range []string{
		"https://www.amazon.com/",
		"https://www.amazon.com/gp/bestsellers/books",
		"https://www.amazon.com/dp/012345",
		"https://www.amazon.com/dp/01234567890",
		"https://amazon./dp/0123456789",
		"https://notamazon.com/dp/0123456789",
		"https://example.com/dp/0123456789",
		"https://example.com/?u=amazon.com/dp/0123456789",
		"http://[::1",
	} {
		if r, ok := ReadingFor(u); ok {
			t.Errorf("%q: got %+v; want no reading", u, r)
		}
	}
}

// TestAmazonReadingRequiresOnlyTheTitle: a robot-check page has no title, so the title alone is
// required and that page reads as a failed check; every other part may be absent (a book with no
// reviews yet has no rating).
func TestAmazonReadingRequiresOnlyTheTitle(t *testing.T) {
	t.Parallel()
	r := amazonProduct()
	for i, f := range r.Fields {
		if f.Optional == (i == 0) {
			t.Errorf("field %q: optional = %v", f.Name, f.Optional)
		}
	}
	if r.Fields[0].Name != "Title" {
		t.Errorf("first field = %q, want Title", r.Fields[0].Name)
	}
}

// TestReadingForHandsOutAFreshCopy: a dialog that edits the rules it was given cannot change the
// next site's.
func TestReadingForHandsOutAFreshCopy(t *testing.T) {
	t.Parallel()
	const u = "https://www.amazon.com/dp/0123456789"
	a, _ := ReadingFor(u)
	a.Fields[0].Selector = "changed"
	a.ExcludeSelectors[0] = "changed"
	b, _ := ReadingFor(u)
	if b.Fields[0].Selector == "changed" || b.ExcludeSelectors[0] == "changed" {
		t.Error("a caller's edit reached the next reading")
	}
}
