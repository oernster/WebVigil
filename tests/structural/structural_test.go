// Package structural holds the tests that enforce WebVigil's architecture by reading its source:
// layering, module size, the wire contract, the supplied icons and the absence of user data.
package structural

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const module = "webvigil/"

// root is the repository root, two levels above this package.
var root = filepath.Join("..", "..")

// MaxLines is the module cap; the danger band below it is refused too (house rule: 5% of 400).
const (
	MaxLines        = 400
	dangerBandWidth = MaxLines / 20
)

// buildScripts are delivery tools exempt from the size cap.
var buildScripts = map[string]bool{
	filepath.Join("tools", "genicons", "main.go"): true,
}

func walk(t *testing.T, dir string, keep func(path string) bool) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "dist" || d.Name() == "wailsjs" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if !d.IsDir() && keep(p) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func imports(t *testing.T, file string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	var out []string
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		out = append(out, p)
	}
	return out
}

func isStdlib(p string) bool {
	return !strings.Contains(strings.Split(p, "/")[0], ".") && !strings.HasPrefix(p, module)
}

func goSources(t *testing.T, dir string) []string {
	return walk(t, dir, func(p string) bool { return strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") })
}

// domainForbidden are standard-library packages that perform I/O, reach the process or the
// machine; others produce randomness. The domain is pure, so it imports none of them.
var domainForbidden = map[string]bool{
	"os": true, "os/exec": true, "log": true, "log/slog": true, "path/filepath": true, "io/ioutil": true,
	"io/fs": true, "syscall": true, "net": true, "database/sql": true, "math/rand": true, "math/rand/v2": true,
	"runtime": true, "unsafe": true,
}

func TestDomainImportsOnlyStdlib(t *testing.T) {
	for _, f := range goSources(t, "internal/domain") {
		for _, p := range imports(t, f) {
			if !isStdlib(p) || domainForbidden[p] {
				t.Errorf("%s imports %s; the domain may import pure standard-library packages only", f, p)
			}
		}
	}
}

func TestApplicationImportsOnlyDomain(t *testing.T) {
	for _, f := range goSources(t, "internal/application") {
		for _, p := range imports(t, f) {
			if !isStdlib(p) && p != module+"internal/domain" {
				t.Errorf("%s imports %s; the application may import the domain and the standard library only", f, p)
			}
		}
	}
}

func TestInnerLayersReadNoClock(t *testing.T) {
	forbidden := regexp.MustCompile(`time\.Now\(|time\.Since\(|time\.Until\(|math/rand`)
	for _, dir := range []string{"internal/domain", "internal/application"} {
		for _, f := range goSources(t, dir) {
			src, _ := os.ReadFile(f)
			if loc := forbidden.Find(src); loc != nil {
				t.Errorf("%s uses %s; time and randomness are injected", f, loc)
			}
		}
	}
}

func TestInfrastructureDoesNotImportTheUI(t *testing.T) {
	for _, f := range goSources(t, "internal/infrastructure") {
		for _, p := range imports(t, f) {
			if strings.Contains(p, "wailsapp/wails") {
				t.Errorf("%s imports %s; the window framework belongs to the composition root", f, p)
			}
		}
	}
}

func countLines(data []byte) int {
	n := strings.Count(string(data), "\n")
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func TestModuleSize(t *testing.T) {
	sources := walk(t, ".", func(p string) bool {
		ext := filepath.Ext(p)
		return ext == ".go" || ext == ".ts" || ext == ".tsx"
	})
	for _, f := range sources {
		rel, _ := filepath.Rel(root, f)
		if buildScripts[rel] {
			continue
		}
		data, _ := os.ReadFile(f)
		switch n := countLines(data); {
		case n > MaxLines:
			t.Errorf("%s has %d lines; the cap is %d", rel, n, MaxLines)
		case n > MaxLines-dangerBandWidth:
			t.Errorf("%s has %d lines, inside the danger band above %d; reduce it to 350 or fewer", rel, n, MaxLines-dangerBandWidth)
		}
	}
}

var (
	goStruct  = regexp.MustCompile(`(?s)type (\w+DTO) struct \{(.*?)\n\}`)
	goTag     = regexp.MustCompile("json:\"(\\w+)")
	tsIface   = regexp.MustCompile(`(?s)export interface (\w+DTO) \{(.*?)\n\}`)
	tsField   = regexp.MustCompile(`(?m)^\s+(\w+):`)
	wireFiles = [2]string{"dto.go", filepath.Join("frontend", "src", "wire.ts")}
)

func wireFields(t *testing.T, file string, block, field *regexp.Regexp) map[string][]string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, m := range block.FindAllStringSubmatch(string(src), -1) {
		var names []string
		for _, f := range field.FindAllStringSubmatch(m[2], -1) {
			names = append(names, f[1])
		}
		sort.Strings(names)
		out[m[1]] = names
	}
	return out
}

// TestWireMatches compares the two statements of the wire: the Go DTOs and the TypeScript
// interfaces. The type checker sees only one and the marshaller only the other.
func TestWireMatches(t *testing.T) {
	goSide := wireFields(t, wireFiles[0], goStruct, goTag)
	tsSide := wireFields(t, wireFiles[1], tsIface, tsField)
	if len(goSide) == 0 {
		t.Fatal("no DTOs found in dto.go")
	}
	for name, fields := range goSide {
		ts, ok := tsSide[name]
		if !ok {
			t.Errorf("%s exists in Go but not in wire.ts", name)
			continue
		}
		if strings.Join(fields, ",") != strings.Join(ts, ",") {
			t.Errorf("%s differs:\n  Go: %v\n  TS: %v", name, fields, ts)
		}
	}
	for name := range tsSide {
		if _, ok := goSide[name]; !ok {
			t.Errorf("%s exists in wire.ts but not in Go", name)
		}
	}
}

var suppliedIcons = []string{
	"application-icon.png", "help-about.png", "light-mode.png", "dark-mode.png", "refresh.png",
	"add-site.png", "edit-site.png", "delete-site.png", "export-sites.png", "import-sites.png", "pause.png", "play.png", "settings.png",
}

func TestSuppliedIconsAreUsed(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(root, "frontend", "src", "icons.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range suppliedIcons {
		if _, err := os.Stat(filepath.Join(root, "assets", name)); err != nil {
			t.Errorf("supplied master %s is missing", name)
		}
		if !strings.Contains(string(src), "./assets/icons/"+name) {
			t.Errorf("icons.ts does not use %s", name)
		}
	}
}

// forbiddenIdentifiers are identifiers of a particular person or their test data. WebVigil is
// written for everyone who installs it, so none may appear in anything shipped (CODE RULES 14).
var forbiddenIdentifiers = []string{"oernster", "codecrafter", "B0HQFRWWGN", "Evergence", "amazon.co.uk/dp"}

// isProjectSite reports a file of the GitHub Pages site under docs/. The site is the publisher's own
// page about WebVigil, so it names the publisher's repository by design; it is built into no binary.
func isProjectSite(p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && strings.HasPrefix(filepath.ToSlash(rel), "docs/")
}

func TestNoUserSpecificData(t *testing.T) {
	shipped := walk(t, ".", func(p string) bool {
		ext := filepath.Ext(p)
		test := strings.HasSuffix(p, "_test.go") || strings.Contains(p, ".test.")
		return !test && !isProjectSite(p) && (ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".css" || ext == ".html" || ext == ".json") &&
			!strings.Contains(p, "package-lock")
	})
	for _, f := range shipped {
		src, _ := os.ReadFile(f)
		low := strings.ToLower(string(src))
		for _, id := range forbiddenIdentifiers {
			if strings.Contains(low, strings.ToLower(id)) {
				t.Errorf("%s contains %q", f, id)
			}
		}
	}
}
