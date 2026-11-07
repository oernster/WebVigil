package domain

import (
	"net/url"
	"regexp"
	"strings"
)

// Reading is a ready-made way to read one kind of page: the mode, fields and ignore rules that
// compare what a reader cares about on it and nothing else. The Add and Edit dialogs fill a
// definition in from it, where it stays visible and editable (FR-058).
type Reading struct {
	Name             string
	Mode             DetectionMode
	Fields           []FieldRule
	ExcludeSelectors []string
	ExcludePatterns  []string
}

// amazonHostLabel is the first label of every Amazon storefront's host (amazon.com, amazon.de,
// amazon.co.uk and the rest), after an optional "www.".
const amazonHostLabel = "amazon."

// amazonProductPath matches a product page: /dp/<ASIN> or /gp/product/<ASIN>, optionally after a
// title slug. An ASIN, which for a book is its ISBN-10, is ten letters and digits.
var amazonProductPath = regexp.MustCompile(`(?i)/(?:dp|gp/product)/[a-z0-9]{10}(?:/|$)`)

// amazonProduct reads an Amazon product page by the element ids Amazon gives each part of the
// product. The whole page differs on every load (sponsored slots, a buy-box data blob, footer
// links), so comparing its text raises an alarm on every check; these parts do not move unless
// the product does. Only the title is required: a robot-check page served in place of the
// product has none, so it reads as a failed check rather than as a change.
func amazonProduct() Reading {
	return Reading{
		Name: "Amazon product page",
		Mode: ModeFields,
		Fields: []FieldRule{
			{Name: "Title", Selector: "#productTitle"},
			{Name: "Author", Selector: "#bylineInfo", Optional: true},
			{Name: "Formats and prices", Selector: "#tmmSwatches", Optional: true},
			{Name: "Rating", Selector: "#acrPopover", Attribute: "title", Optional: true},
			{Name: "Reviews", Selector: "#acrCustomerReviewText", Optional: true},
			{Name: "Description", Selector: "#bookDescription_feature_div", Optional: true},
			{Name: "Cover", Selector: "#landingImage", Attribute: "data-old-hires", Optional: true},
			{Name: "Details", Selector: "#detailBullets_feature_div ul.detail-bullet-list", Optional: true},
		},
		// The details list also carries the sales rank, which moves by the hour, plus a second copy of
		// the rating, which the Rating and Reviews fields already compare.
		ExcludeSelectors: []string{
			"#detailBullets_feature_div li:has(.zg_hrsr)",
			"#detailBullets_feature_div li:has(#detailBullets_averageCustomerReviews)",
		},
		// Invisible left-to-right and right-to-left marks pad every detail; they would only clutter
		// a reported change.
		ExcludePatterns: []string{`[\x{200E}\x{200F}]`},
	}
}

// ReadingFor returns the ready-made reading for rawURL; false when there is none for that page.
func ReadingFor(rawURL string) (Reading, bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return Reading{}, false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if len(host) > len(amazonHostLabel) && strings.HasPrefix(host, amazonHostLabel) && amazonProductPath.MatchString(u.Path) {
		return amazonProduct(), true
	}
	return Reading{}, false
}
