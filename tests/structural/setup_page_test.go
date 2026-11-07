package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The setup page takes the product's name from the state Go hands it (house rule 8), so a rename
// is one constant and the page can never disagree with the program it installs.
var (
	htmlComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
	jsLineComment = regexp.MustCompile(`(?m)^\s*//.*$`)
	// scrollGlobal is the bundled scroller's global, a code identifier rather than shown text.
	scrollGlobal = regexp.MustCompile(`\bWebVigilScroll\b`)
)

const productName = "WebVigil"

// pageWithoutComments is the page as the user can meet it: markup and script, no comments.
func pageWithoutComments(src string) string {
	return scrollGlobal.ReplaceAllString(jsLineComment.ReplaceAllString(htmlComment.ReplaceAllString(src, ""), ""), "")
}

func TestTheSetupPageNamesNoProduct(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(root, setupPageDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := pageWithoutComments(string(src))
	for n, line := range strings.Split(page, "\n") {
		if strings.Contains(line, productName) {
			t.Errorf("setup page line %d names the product itself: %s", n+1, strings.TrimSpace(line))
		}
	}
	if !strings.Contains(page, "state.name") {
		t.Error("the setup page never reads the name from state")
	}
}

func TestSetupPageCommentStripping(t *testing.T) {
	kept := pageWithoutComments("<!-- WebVigil -->\n  // WebVigil here\nwindow.WebVigilScroll\n<h1>WebVigil</h1>")
	if strings.Count(kept, productName) != 1 || !strings.Contains(kept, "<h1>WebVigil</h1>") {
		t.Errorf("stripping kept %q", kept)
	}
}
