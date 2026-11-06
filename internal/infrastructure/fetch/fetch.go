// Package fetch implements application.Fetcher over net/http: no cookies, TLS verified, bounded
// redirects and body size, conditional requests and an explicit category for every failure.
package fetch

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/charset"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

const acceptHeader = "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.5"

var errTooManyRedirects = errors.New("too many redirects")

// textualTypes are the media types WebVigil compares. Anything else is unsupported (FR-026).
var textualTypes = map[string]bool{
	"text/html": true, "application/xhtml+xml": true, "text/plain": true, "text/xml": true,
	"application/xml": true, "application/json": true, "application/rss+xml": true, "application/atom+xml": true,
}

// Fetcher retrieves pages.
type Fetcher struct {
	client    *http.Client
	userAgent string
	maxBytes  int64
}

// New returns a fetcher identifying itself as userAgent. Proxy settings come from the
// environment, exactly as for any other Go program; certificates are always verified.
func New(userAgent string, maxBytes int64, maxRedirects int) *Fetcher {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return errTooManyRedirects
			}
			return nil
		},
	}
	return &Fetcher{client: client, userAgent: userAgent, maxBytes: maxBytes}
}

// Fetch performs one GET. It never invents an HTTP status: HTTPStatus stays zero unless a response
// arrived (FR-023).
func (f *Fetcher) Fetch(ctx context.Context, req application.FetchRequest) application.FetchResponse {
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	start := time.Now()
	out := application.FetchResponse{FinalURL: req.URL}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		out.Failure, out.FailureDetail = domain.CategoryNetwork, err.Error()
		return out
	}
	httpReq.Header.Set("User-Agent", f.userAgent)
	httpReq.Header.Set("Accept", acceptHeader)
	if req.ETag != "" {
		httpReq.Header.Set("If-None-Match", req.ETag)
	}
	if req.LastModified != "" {
		httpReq.Header.Set("If-Modified-Since", req.LastModified)
	}
	resp, err := f.client.Do(httpReq)
	if err != nil {
		out.Duration = time.Since(start)
		out.Failure, out.FailureDetail = classify(ctx, err)
		return out
	}
	defer resp.Body.Close()
	out.FinalURL = resp.Request.URL.String()
	out.HTTPStatus = resp.StatusCode
	out.ContentType = resp.Header.Get("Content-Type")
	out.ETag = resp.Header.Get("ETag")
	out.LastModified = resp.Header.Get("Last-Modified")
	out.RetryAfter = retryAfter(resp.Header.Get("Retry-After"), time.Now())
	raw, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes+1))
	out.Duration = time.Since(start)
	out.Bytes = int64(len(raw))
	if err != nil {
		out.Failure, out.FailureDetail = classify(ctx, err)
		return out
	}
	if int64(len(raw)) > f.maxBytes {
		out.Failure = domain.CategoryResponseTooLarge
		out.FailureDetail = fmt.Sprintf("more than %d bytes", f.maxBytes)
		return out
	}
	if resp.StatusCode == http.StatusNotModified {
		return out
	}
	mediaType := mediaTypeOf(out.ContentType)
	if domain.IsSuccessStatus(resp.StatusCode) && !textualTypes[mediaType] {
		out.Failure, out.FailureDetail = domain.CategoryUnsupportedContentType, out.ContentType
		return out
	}
	out.Body = toUTF8(raw, out.ContentType)
	return out
}

func mediaTypeOf(contentType string) string {
	if contentType == "" {
		return "text/html" // a server that names no type is read as HTML, the common case
	}
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}
	return mt
}

// toUTF8 converts a body to UTF-8 from the charset its header or markup declares.
func toUTF8(raw []byte, contentType string) []byte {
	r, err := charset.NewReader(bytes.NewReader(raw), contentType)
	if err != nil {
		return raw
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return raw
	}
	return out
}

// retryAfter reads a Retry-After header as seconds or as an HTTP date.
func retryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil && t.After(now) {
		return t.Sub(now)
	}
	return 0
}

// classify names a transport failure from the error's type, never from its text.
func classify(ctx context.Context, err error) (domain.Category, string) {
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	var unknownAuth x509.UnknownAuthorityError
	var hostErr x509.HostnameError
	var invalidCert x509.CertificateInvalidError
	var recordErr tls.RecordHeaderError
	var netErr net.Error
	switch {
	case errors.Is(err, errTooManyRedirects):
		return domain.CategoryRedirectLoop, ""
	case errors.Is(err, context.Canceled) && ctx.Err() == context.Canceled:
		return domain.CategoryCancelled, ""
	case errors.As(err, &dnsErr):
		return domain.CategoryDNS, dnsErr.Name
	case isRefused(err):
		return domain.CategoryConnectionRefused, ""
	case errors.As(err, &certErr), errors.As(err, &unknownAuth), errors.As(err, &hostErr),
		errors.As(err, &invalidCert), errors.As(err, &recordErr):
		return domain.CategoryTLS, err.Error()
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return domain.CategoryConnectionTimeout, ""
	}
	return domain.CategoryNetwork, err.Error()
}
