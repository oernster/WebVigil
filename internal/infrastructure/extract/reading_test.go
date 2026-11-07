package extract

import (
	"errors"
	"strings"
	"testing"

	"webvigil/internal/domain"
)

// amazonPage is a cut-down product page in the shape Amazon serves (the element ids and the
// details list measured on two live book pages), with the parts that differ on every load.
const amazonPage = `<!doctype html><html><head><title>A Book</title></head><body>
<div id="nav">Menu {{MENU}}</div>
<span id="productTitle"> A Book </span>
<div id="bylineInfo"><span>by</span> <a>Some Author</a> <span>(Author)</span></div>
<div id="tmmSwatches"><ul><li>Kindle Edition <span>£4.99</span></li><li>Paperback <span>{{PRICE}}</span></li></ul></div>
<span id="acrPopover" title="4.5 out of 5 stars"><i>4.5</i></span>
<span id="acrCustomerReviewText">(1,234)</span>
<div id="bookDescription_feature_div"><p>What the book is about.</p></div>
<img id="landingImage" src="small.jpg" data-old-hires="https://img.example/cover.jpg">
<div>Sponsored {{AD}}</div>
<div>{"buybox":"{{TOKEN}}"}</div>
<div id="detailBullets_feature_div"><ul class="a-unordered-list detail-bullet-list">
<li><span class="a-list-item"><span>Publisher &rlm; : &lrm;</span><span>A Press</span></span></li>
<li><span class="a-list-item"><span>Print length &rlm; : &lrm;</span><span>300 pages</span></span></li>
<li><span class="a-list-item"><span>Best Sellers Rank:</span> {{RANK}} in Books <ul class="zg_hrsr"><li>3 in Fiction</li></ul></span></li>
<li><span class="a-list-item"><span>Customer reviews:</span><div id="detailBullets_averageCustomerReviews">4.5 out of 5 stars</div></span></li>
</ul></div>
</body></html>`

func amazonBody(menu, price, ad, token, rank string) []byte {
	return []byte(strings.NewReplacer("{{MENU}}", menu, "{{PRICE}}", price, "{{AD}}", ad, "{{TOKEN}}", token,
		"{{RANK}}", rank).Replace(amazonPage))
}

// amazonSite is a definition filled in from the Amazon reading, as the site dialog fills it.
func amazonSite(t *testing.T) domain.Site {
	t.Helper()
	r, ok := domain.ReadingFor("https://www.amazon.com/dp/0123456789")
	if !ok {
		t.Fatal("no Amazon reading")
	}
	s := domain.NewSite("book", "https://www.amazon.com/dp/0123456789")
	s.Mode = r.Mode
	s.Extraction.Fields = r.Fields
	s.Ignore.ExcludeSelectors = r.ExcludeSelectors
	s.Ignore.ExcludePatterns = r.ExcludePatterns
	if err := New().Validate(s); err != nil {
		t.Fatalf("the reading's selectors do not compile: %v", err)
	}
	return s
}

func readAmazon(t *testing.T, s domain.Site, body []byte) domain.Content {
	t.Helper()
	c, err := New().Extract(body, "text/html", s)
	if err != nil {
		t.Fatal(err)
	}
	n, err := domain.NormaliseContent(c, s.Ignore)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestAmazonReadingReadsTheProduct: each field takes the product part it names, the details keep
// their facts without the sales rank or the repeated rating and the direction marks are gone.
func TestAmazonReadingReadsTheProduct(t *testing.T) {
	t.Parallel()
	s := amazonSite(t)
	got := map[string]string{}
	for _, f := range readAmazon(t, s, amazonBody("a", "£9.99", "x", "1", "540")).Fields {
		got[f.Name] = f.Value
	}
	want := map[string]string{
		"Title": "A Book", "Author": "by Some Author (Author)", "Formats and prices": "Kindle Edition £4.99\nPaperback £9.99",
		"Rating": "4.5 out of 5 stars", "Reviews": "(1,234)", "Description": "What the book is about.",
		"Cover": "https://img.example/cover.jpg", "Details": "Publisher : A Press\nPrint length : 300 pages",
	}
	for name, v := range want {
		if got[name] != v {
			t.Errorf("%s = %q, want %q", name, got[name], v)
		}
	}
}

// TestAmazonReadingIgnoresWhatMovesOnEveryLoad: menus, sponsored slots, the buy-box blob and the
// sales rank change nothing; a price does.
func TestAmazonReadingIgnoresWhatMovesOnEveryLoad(t *testing.T) {
	t.Parallel()
	s := amazonSite(t)
	base := readAmazon(t, s, amazonBody("a", "£9.99", "x", "1", "540")).Hash()
	if h := readAmazon(t, s, amazonBody("b", "£9.99", "y", "2", "612")).Hash(); h != base {
		t.Error("page clutter changed the comparison")
	}
	if h := readAmazon(t, s, amazonBody("a", "£7.99", "x", "1", "540")).Hash(); h == base {
		t.Error("a price change went unseen")
	}
}

// TestAmazonReadingFailsOnARobotCheck: the page Amazon serves in place of the product has no
// title, so it is a failed check rather than a change.
func TestAmazonReadingFailsOnARobotCheck(t *testing.T) {
	t.Parallel()
	robot := []byte(`<html><body><p>Click the button below to continue shopping</p><button>Continue shopping</button></body></html>`)
	if _, err := New().Extract(robot, "text/html", amazonSite(t)); !errors.Is(err, domain.ErrSelectorMissing) {
		t.Errorf("robot check: %v, want a missing-selector failure", err)
	}
}
