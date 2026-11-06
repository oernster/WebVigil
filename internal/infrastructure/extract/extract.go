// Package extract implements application.Extractor: it parses HTML without running any script
// and produces the content each detection mode compares.
package extract

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"webvigil/internal/domain"
)

// Extractor turns response bodies into comparable content.
type Extractor struct{}

// New returns an extractor.
func New() Extractor { return Extractor{} }

// hiddenElements never contribute visible text (FR-051).
var hiddenElements = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Svg: true,
	atom.Head: true, atom.Iframe: true, atom.Object: true, atom.Canvas: true,
}

// blockElements start a new line in extracted text.
var blockElements = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Br: true, atom.Li: true, atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.H5: true, atom.H6: true, atom.Tr: true, atom.Td: true, atom.Th: true, atom.Section: true,
	atom.Article: true, atom.Header: true, atom.Footer: true, atom.Ul: true, atom.Ol: true, atom.Table: true,
	atom.Main: true, atom.Nav: true, atom.Aside: true, atom.Dd: true, atom.Dt: true, atom.Blockquote: true,
	atom.Pre: true, atom.Form: true, atom.Hr: true, atom.Title: true, atom.Figcaption: true, atom.Address: true,
}

// Validate compiles every selector of the definition, so a typo is refused when it is saved.
func (Extractor) Validate(site domain.Site) error {
	all := append(append([]string{}, site.Extraction.Selectors...), site.Ignore.ExcludeSelectors...)
	for _, f := range site.Extraction.Fields {
		all = append(all, f.Selector)
	}
	for _, sel := range all {
		if _, err := cascadia.Parse(sel); err != nil {
			return fmt.Errorf("%w: the selector %q is not valid: %v", domain.ErrInvalidSite, sel, err)
		}
	}
	return nil
}

// Extract produces the content for the site's detection mode.
func (x Extractor) Extract(body []byte, contentType string, site domain.Site) (domain.Content, error) {
	if site.Mode == domain.ModeRaw {
		return domain.Content{Text: string(body)}, nil
	}
	if site.Mode == domain.ModeText && !isMarkup(contentType) {
		return domain.Content{Text: string(body)}, nil
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return domain.Content{}, fmt.Errorf("%w: %v", domain.ErrParseFailure, err)
	}
	for _, sel := range site.Ignore.ExcludeSelectors {
		m, err := compile(sel)
		if err != nil {
			return domain.Content{}, err
		}
		for _, n := range cascadia.QueryAll(doc, m) {
			if n.Parent != nil {
				n.Parent.RemoveChild(n)
			}
		}
	}
	switch site.Mode {
	case domain.ModeSelector:
		return selectorContent(doc, site.Extraction.Selectors)
	case domain.ModeFields:
		return fieldContent(doc, site.Extraction.Fields)
	}
	return domain.Content{Text: VisibleText(doc)}, nil
}

func isMarkup(contentType string) bool {
	ct := strings.ToLower(contentType)
	return ct == "" || strings.Contains(ct, "html") || strings.Contains(ct, "xml")
}

func compile(sel string) (cascadia.Sel, error) {
	m, err := cascadia.Parse(sel)
	if err != nil {
		return nil, fmt.Errorf("%w: selector %q: %v", domain.ErrParseFailure, sel, err)
	}
	return m, nil
}

func selectorContent(doc *html.Node, selectors []string) (domain.Content, error) {
	var parts []string
	for _, sel := range selectors {
		m, err := compile(sel)
		if err != nil {
			return domain.Content{}, err
		}
		nodes := cascadia.QueryAll(doc, m)
		if len(nodes) == 0 {
			return domain.Content{}, fmt.Errorf("%w: %s", domain.ErrSelectorMissing, sel)
		}
		for _, n := range nodes {
			parts = append(parts, VisibleText(n))
		}
	}
	return domain.Content{Text: strings.Join(parts, "\n")}, nil
}

func fieldContent(doc *html.Node, rules []domain.FieldRule) (domain.Content, error) {
	fields := make([]domain.FieldValue, 0, len(rules))
	for _, r := range rules {
		m, err := compile(r.Selector)
		if err != nil {
			return domain.Content{}, err
		}
		n := cascadia.Query(doc, m)
		value := ""
		switch {
		case n == nil && !r.Optional:
			return domain.Content{}, fmt.Errorf("%w: %s (%s)", domain.ErrSelectorMissing, r.Name, r.Selector)
		case n == nil:
		case r.Attribute != "":
			value = attr(n, r.Attribute)
		default:
			value = VisibleText(n)
		}
		fields = append(fields, domain.FieldValue{Name: r.Name, Value: value})
	}
	return domain.Content{Fields: fields}, nil
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

func isHidden(n *html.Node) bool {
	if hiddenElements[n.DataAtom] {
		return true
	}
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "hidden":
			return true
		case "aria-hidden":
			if strings.EqualFold(a.Val, "true") {
				return true
			}
		case "style":
			s := strings.ReplaceAll(strings.ToLower(a.Val), " ", "")
			if strings.Contains(s, "display:none") || strings.Contains(s, "visibility:hidden") {
				return true
			}
		}
	}
	return false
}

// VisibleText returns the text a reader would see under n, one block per line.
func VisibleText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
			return
		case html.ElementNode:
			if isHidden(n) {
				return
			}
			if blockElements[n.DataAtom] {
				b.WriteByte('\n')
				defer b.WriteByte('\n')
			}
		case html.CommentNode, html.DoctypeNode:
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
