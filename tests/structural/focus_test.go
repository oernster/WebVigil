package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// noborderfocus, held by a scan: a ring belongs to a control, never to the pane that holds it. Both
// stylesheets are read, the application's and the setup page's (which has no build step and so no
// other guard). A state rule (hover, focus, focus-visible, focus-within) whose subject is a pane or a
// text view fails; so does losing either rule that keeps the browser's own ring off a scroller, which
// Chromium makes a Tab stop while it overflows (measured on the setup licence: a real Tab painted
// "outline: auto" round the whole pane until the rule was added).

var (
	appStylesheet    = filepath.Join("frontend", "src", "app.css")
	setupPage        = filepath.Join("installer", "frontend", "dist", "index.html")
	cssComment       = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssRule          = regexp.MustCompile(`([^{}]+)\{[^}]*\}`)
	stateSelector    = regexp.MustCompile(`:(hover|focus|focus-visible|focus-within)\b`)
	pseudo           = regexp.MustCompile(`::?[a-z-]+(\([^)]*\))?`)
	combinator       = regexp.MustCompile(`\s+|>|\+|~`)
	appSuppression   = regexp.MustCompile(`(^|\n):focus\s*\{\s*outline:\s*none;\s*\}`)
	setupSuppression = regexp.MustCompile(`pre\.licence\s*\{\s*outline:\s*none;\s*\}`)
)

// paneElements and paneClasses are the subjects that hold things rather than being acted on.
var paneElements = []string{"*", "div", "section", "main", "pre", "header", "footer", "table", "dl"}

var paneClasses = []string{
	".content", ".body", ".table-wrap", ".preview", ".details", ".modal", ".backdrop", ".shell",
	".licence", ".licence-text", ".guide", ".guide-section", ".about", ".diff", ".form", ".checks",
}

// stateSubjects answers the last compound of every selector carrying a state, its pseudo-classes
// removed.
func stateSubjects(css string) []string {
	var subjects []string
	for _, m := range cssRule.FindAllStringSubmatch(cssComment.ReplaceAllString(css, ""), -1) {
		for _, sel := range strings.Split(m[1], ",") {
			sel = strings.TrimSpace(sel)
			if !stateSelector.MatchString(sel) {
				continue
			}
			parts := combinator.Split(sel, -1)
			subjects = append(subjects, pseudo.ReplaceAllString(parts[len(parts)-1], ""))
		}
	}
	return subjects
}

// isPane reports a subject naming a pane: a bare pane element, that element with classes, any
// compound carrying a pane class.
func isPane(subject string) bool {
	for _, el := range paneElements {
		if subject == el || strings.HasPrefix(subject, el+".") {
			return true
		}
	}
	pieces := strings.Split(strings.ReplaceAll(subject, ".", " ."), " ")
	for _, class := range paneClasses {
		for _, piece := range pieces {
			if piece == class {
				return true
			}
		}
	}
	return false
}

func paneOffences(css string) []string {
	var offences []string
	for _, s := range stateSubjects(css) {
		if isPane(s) {
			offences = append(offences, s)
		}
	}
	return offences
}

func readSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func setupStyles(t *testing.T) string {
	page := readSource(t, setupPage)
	start, end := strings.Index(page, "<style>"), strings.Index(page, "</style>")
	if start < 0 || end < start {
		t.Fatalf("%s carries no <style> block", setupPage)
	}
	return page[start:end]
}

func TestNoPaneWearsARing(t *testing.T) {
	for name, css := range map[string]string{appStylesheet: readSource(t, appStylesheet), setupPage: setupStyles(t)} {
		if len(stateSubjects(css)) == 0 {
			t.Errorf("%s: no state rules found; the scan is reading the wrong thing", name)
		}
		for _, o := range paneOffences(css) {
			t.Errorf("%s: a state rule rings the pane %q", name, o)
		}
	}
}

func TestTheBrowserRingStaysOffScrollers(t *testing.T) {
	if !appSuppression.MatchString(readSource(t, appStylesheet)) {
		t.Errorf("%s lost its `:focus { outline: none; }`", appStylesheet)
	}
	if !setupSuppression.MatchString(setupStyles(t)) {
		t.Errorf("%s lost its `pre.licence { outline: none; }`", setupPage)
	}
}

func TestThePaneScanBites(t *testing.T) {
	for _, planted := range []string{
		".modal .content:focus { outline: 2px solid red; }",
		"pre.licence:focus-visible { outline: auto; }",
		".table-wrap:hover { border-color: red; }",
		"*:focus { outline: 1px solid; }",
	} {
		if len(paneOffences(planted)) != 1 {
			t.Errorf("planted %q was not caught", planted)
		}
	}
	if o := paneOffences("button:enabled:focus-visible, .menu-item:hover { outline: 2px solid; }"); len(o) != 0 {
		t.Errorf("a control's ring was flagged: %v", o)
	}
}
