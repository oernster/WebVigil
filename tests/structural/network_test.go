package structural

// WebVigil's documents promise two kinds of connection only: the pages the user configured and the
// GitHub releases check. These tests pin that promise to the source, ported from PigeonPost: only
// the packages named below may import anything that opens a connection, the page makes no request
// of its own and only the composition root wires the application and infrastructure together.
//
// What this cannot see: a request Wails or its web view makes on its own account; a connection a
// library opens through a package not listed here. It holds over this repository's source, not
// over the binary.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// networkReachers are the directories allowed to open a connection, each with the documented
// purpose it serves. A new one is a new way out of the machine: it belongs in the documents first.
var networkReachers = map[string]string{
	"internal/infrastructure/fetch":  "the pages the user configured",
	"internal/infrastructure/update": "the GitHub releases check",
}

// networkRoots are the packages a connection is opened through; a path beneath one is one of them.
var networkRoots = []string{"net", "crypto/tls", "golang.org/x/net"}

// parsingOnly are packages under a network root that only read or write a format and cannot open a
// connection.
var parsingOnly = []string{"net/url", "net/mail", "net/textproto", "net/netip", "golang.org/x/net/html"}

// httpPackage may be imported anywhere for its status names alone (the constants and StatusText),
// which say what a server answered and open nothing; statusName is the only shape of use allowed
// outside the reachers.
const httpPackage = "net/http"

var statusName = regexp.MustCompile(`^Status[A-Z]`)

const (
	applicationPkg    = module + "internal/application"
	infrastructurePkg = module + "internal/infrastructure"
)

// goFile is one shipped Go source file, parsed.
type goFile struct {
	rel     string // slash-separated, from the repository root
	relDir  string
	pkg     string
	imports map[string]string // import path to the name it is used by in this file
	ast     *ast.File
}

func scanRepo(t *testing.T) []goFile {
	t.Helper()
	var out []goFile
	for _, p := range goSources(t, ".") {
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		g := goFile{rel: rel, relDir: filepath.ToSlash(filepath.Dir(rel)), pkg: f.Name.Name, imports: map[string]string{}, ast: f}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			name := path[strings.LastIndex(path, "/")+1:]
			if imp.Name != nil {
				name = imp.Name.Name
			}
			g.imports[path] = name
		}
		out = append(out, g)
	}
	if len(out) == 0 {
		t.Fatal("no Go source scanned; check the root")
	}
	return out
}

func underPath(imported, base string) bool {
	return imported == base || strings.HasPrefix(imported, base+"/")
}

func isNetworkPackage(imported string) bool {
	for _, parsing := range parsingOnly {
		if underPath(imported, parsing) {
			return false
		}
	}
	for _, r := range networkRoots {
		if underPath(imported, r) {
			return true
		}
	}
	return false
}

func mayReachNetwork(f goFile) bool {
	for dir := range networkReachers {
		if f.relDir == dir || strings.HasPrefix(f.relDir, dir+"/") {
			return true
		}
	}
	return false
}

// nonStatusUses lists every use of the named package in f that is not a status constant.
func nonStatusUses(f goFile, name string) []string {
	var bad []string
	ast.Inspect(f.ast, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == name && !statusName.MatchString(sel.Sel.Name) {
			bad = append(bad, name+"."+sel.Sel.Name)
		}
		return true
	})
	return bad
}

func TestOnlyTheListedPackagesReachTheNetwork(t *testing.T) {
	for _, f := range scanRepo(t) {
		if mayReachNetwork(f) {
			continue
		}
		for imported, name := range f.imports {
			if !isNetworkPackage(imported) {
				continue
			}
			if imported == httpPackage {
				if bad := nonStatusUses(f, name); len(bad) > 0 {
					t.Errorf("%s uses %s beyond its status constants: %v; only the packages in networkReachers may open a connection",
						f.rel, httpPackage, bad)
				}
				continue
			}
			t.Errorf("%s imports %s: only the packages in networkReachers may open a connection", f.rel, imported)
		}
	}
}

// An exemption must name something that exists, so a moved package cannot leave the rule pointing at
// nothing while its new home goes unchecked.
func TestEveryNetworkExemptionExists(t *testing.T) {
	for dir := range networkReachers {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil || !info.IsDir() {
			t.Errorf("%s is allowed to reach the network but is not a directory: %v", dir, err)
		}
	}
}

func TestNetworkPackageRecognitionIsExact(t *testing.T) {
	for _, network := range []string{"net", "net/http", "net/http/httputil", "net/smtp", "crypto/tls", "golang.org/x/net/proxy"} {
		if !isNetworkPackage(network) {
			t.Errorf("%s was not recognised as a network package", network)
		}
	}
	for _, ordinary := range []string{"net/url", "net/mail", "net/textproto", "net/netip", "golang.org/x/net/html",
		"golang.org/x/net/html/atom", "network", "crypto/sha256"} {
		if isNetworkPackage(ordinary) {
			t.Errorf("%s was taken for a network package", ordinary)
		}
	}
}

// TestStatusOnlyUseIsRecognised: a file naming only status constants passes; one reaching for a
// client, a request or a transport does not.
func TestStatusOnlyUseIsRecognised(t *testing.T) {
	parse := func(src string) goFile {
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		return goFile{ast: f}
	}
	ok := parse("package x\nimport \"net/http\"\nvar a, b = http.StatusOK, http.StatusNotFound\n")
	if bad := nonStatusUses(ok, "http"); len(bad) != 0 {
		t.Errorf("status constants flagged: %v", bad)
	}
	for _, use := range []string{"http.Get(u)", "http.DefaultClient", "http.NewRequest(m, u, nil)", "http.Client{}"} {
		f := parse("package x\nimport \"net/http\"\nvar _ = " + use + "\n")
		if bad := nonStatusUses(f, "http"); len(bad) == 0 {
			t.Errorf("%s was taken for a status constant", use)
		}
	}
}

// TestCompositionRootIsWhitelisted: only package main at the repository root wires the application
// and infrastructure layers together.
func TestCompositionRootIsWhitelisted(t *testing.T) {
	for _, f := range scanRepo(t) {
		app, infra := false, false
		for imported := range f.imports {
			app = app || underPath(imported, applicationPkg)
			infra = infra || underPath(imported, infrastructurePkg)
		}
		if app && infra && (f.relDir != "." || f.pkg != "main") {
			t.Errorf("%s (package %s) imports both application and infrastructure: only the composition root may", f.rel, f.pkg)
		}
	}
}

// frontendRequest matches every way a page script can ask the network for something itself.
var frontendRequest = regexp.MustCompile(`\bfetch\s*\(|XMLHttpRequest|\bWebSocket\b|\bEventSource\b|\bsendBeacon\b`)

// commentLine is a line holding only a comment, where "fetch (" is an English word rather than a call.
var commentLine = regexp.MustCompile(`^\s*(//|/\*|\*)`)

func isRequest(line string) bool {
	return !commentLine.MatchString(line) && frontendRequest.MatchString(line)
}

// TestTheFrontEndMakesNoRequestOfItsOwn: the page reaches Go through the Wails bindings and nothing
// else, so every request leaves through the packages above, where these tests can see it.
func TestTheFrontEndMakesNoRequestOfItsOwn(t *testing.T) {
	sources := walk(t, filepath.Join("frontend", "src"), func(p string) bool {
		ext := filepath.Ext(p)
		return (ext == ".ts" || ext == ".tsx") && !strings.Contains(filepath.Base(p), ".test.")
	})
	if len(sources) == 0 {
		t.Fatal("no front-end source scanned; check the path")
	}
	for _, p := range sources {
		content, _ := os.ReadFile(p)
		for n, line := range strings.Split(string(content), "\n") {
			if isRequest(line) {
				rel, _ := filepath.Rel(root, p)
				t.Errorf("%s:%d makes a request of its own: %s", filepath.ToSlash(rel), n+1, strings.TrimSpace(line))
			}
		}
	}
}

func TestFrontEndRequestRecognition(t *testing.T) {
	for _, request := range []string{"fetch(url)", "await fetch (u)", "new XMLHttpRequest()", "new WebSocket(u)",
		"new EventSource(u)", "navigator.sendBeacon(u, b)"} {
		if !isRequest(request) {
			t.Errorf("%q was not recognised as a request", request)
		}
	}
	for _, ordinary := range []string{"refetch(x)", "prefetchRows()", "fetchedAt", "WebSockets",
		"  // a failed fetch (offline, say) survives", " * during the fetch (the user switched)"} {
		if isRequest(ordinary) {
			t.Errorf("%q was taken for a request", ordinary)
		}
	}
}
