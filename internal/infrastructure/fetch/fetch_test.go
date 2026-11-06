package fetch

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

const ua = "WebVigil/test"

func newFetcher() *Fetcher { return New(ua, 1024, domain.MaxRedirects) }

func get(f *Fetcher, url string) application.FetchResponse {
	return f.Fetch(context.Background(), application.FetchRequest{URL: url, Timeout: 5 * time.Second})
}

func TestFetchRecordsResponse(t *testing.T) {
	t.Parallel()
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusMovedPermanently)
			return
		}
		seen = r.Header.Clone()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Last-Modified", "Fri, 06 Nov 2026 12:00:00 GMT")
		_, _ = w.Write([]byte("<p>hello</p>"))
	}))
	defer srv.Close()
	r := get(newFetcher(), srv.URL+"/old")
	if r.Failure != domain.CategoryNone || r.HTTPStatus != 200 || r.FinalURL != srv.URL+"/new" || string(r.Body) != "<p>hello</p>" ||
		r.ETag != `"v1"` || r.LastModified == "" || r.Bytes != 12 || r.ContentType != "text/html; charset=utf-8" || r.Duration <= 0 {
		t.Errorf("response = %+v", r)
	}
	if seen.Get("User-Agent") != ua || seen.Get("Cookie") != "" || !strings.Contains(seen.Get("Accept"), "text/html") {
		t.Errorf("request headers = %v", seen)
	}
}

func TestConditionalHeadersAnd304(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"v1"` && r.Header.Get("If-Modified-Since") == "Fri" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write([]byte("x"))
	}))
	defer srv.Close()
	r := newFetcher().Fetch(context.Background(), application.FetchRequest{URL: srv.URL, Timeout: time.Second, ETag: `"v1"`, LastModified: "Fri"})
	if r.HTTPStatus != http.StatusNotModified || r.Failure != domain.CategoryNone {
		t.Errorf("304 = %+v", r)
	}
}

func TestFetchSizeCap(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 2048)))
	}))
	defer srv.Close()
	if r := get(newFetcher(), srv.URL); r.Failure != domain.CategoryResponseTooLarge || r.HTTPStatus != 200 {
		t.Errorf("oversize = %+v", r)
	}
}

func TestUnsupportedContentType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
		}
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	defer srv.Close()
	if r := get(newFetcher(), srv.URL); r.Failure != domain.CategoryUnsupportedContentType {
		t.Errorf("png = %+v", r)
	}
	if r := get(newFetcher(), srv.URL+"/missing"); r.Failure != domain.CategoryNone || r.HTTPStatus != 404 {
		t.Errorf("an error status is reported as a status, not a content type: %+v", r)
	}
}

func TestRedirectLoop(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path+"x", http.StatusFound)
	}))
	defer srv.Close()
	if r := get(newFetcher(), srv.URL+"/"); r.Failure != domain.CategoryRedirectLoop || r.HTTPStatus != 0 {
		t.Errorf("loop = %+v", r)
	}
}

func TestNetworkFailuresHaveNoStatus(t *testing.T) {
	t.Parallel()
	closed, _ := net.Listen("tcp", "127.0.0.1:0")
	refusedURL := "http://" + closed.Addr().String() + "/"
	_ = closed.Close()
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer slow.Close()
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer tlsSrv.Close()
	cases := map[string]struct {
		url     string
		timeout time.Duration
		want    domain.Category
	}{
		"dns":     {"http://webvigil-test.invalid/", 5 * time.Second, domain.CategoryDNS},
		"refused": {refusedURL, 5 * time.Second, domain.CategoryConnectionRefused},
		"timeout": {slow.URL, 200 * time.Millisecond, domain.CategoryConnectionTimeout},
		"tls":     {tlsSrv.URL, 5 * time.Second, domain.CategoryTLS},
		"bad url": {"http://exa mple.com/", time.Second, domain.CategoryNetwork},
	}
	for name, c := range cases {
		r := newFetcher().Fetch(context.Background(), application.FetchRequest{URL: c.url, Timeout: c.timeout})
		if r.Failure != c.want || r.HTTPStatus != 0 {
			t.Errorf("%s: failure = %q (%s), status %d; want %q", name, r.Failure, r.FailureDetail, r.HTTPStatus, c.want)
		}
	}
}

func TestCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	r := newFetcher().Fetch(ctx, application.FetchRequest{URL: srv.URL, Timeout: 5 * time.Second})
	if r.Failure != domain.CategoryCancelled {
		t.Errorf("cancel = %+v", r)
	}
}

func TestCharsetIsDecoded(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		_, _ = w.Write([]byte{'c', 'a', 'f', 0xE9})
	}))
	defer srv.Close()
	if r := get(newFetcher(), srv.URL); string(r.Body) != "café" {
		t.Errorf("body = %q", r.Body)
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	cases := map[string]time.Duration{
		"": 0, "120": 2 * time.Minute, "-5": 0, "soon": 0,
		"Fri, 06 Nov 2026 13:00:00 GMT": time.Hour, "Fri, 06 Nov 2026 11:00:00 GMT": 0,
	}
	for in, want := range cases {
		if got := retryAfter(in, now); got != want {
			t.Errorf("retryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestMediaTypeOf(t *testing.T) {
	t.Parallel()
	cases := map[string]string{"": "text/html", "Text/HTML; charset=x": "text/html", "text/plain;;bad=": "text/plain"}
	for in, want := range cases {
		if got := mediaTypeOf(in); got != want {
			t.Errorf("mediaTypeOf(%q) = %q", in, got)
		}
	}
}
