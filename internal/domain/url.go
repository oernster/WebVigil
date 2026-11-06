package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// MaxURLLength bounds a stored URL; longer values are refused rather than truncated.
const MaxURLLength = 2048

const (
	schemeHTTP       = "http"
	schemeHTTPS      = "https"
	defaultHTTPPort  = "80"
	defaultHTTPSPort = "443"
)

// NormaliseURL validates raw and returns its canonical form: scheme and host lower-cased, a
// default port removed, the fragment removed and an empty path given as "/". Two URLs that
// normalise equal are treated as the same website for duplicate detection.
func NormaliseURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("%w: the address is empty", ErrInvalidURL)
	}
	if len(s) > MaxURLLength {
		return "", fmt.Errorf("%w: the address is longer than %d characters", ErrInvalidURL, MaxURLLength)
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("%w: %q cannot be read as an address", ErrInvalidURL, s)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != schemeHTTP && scheme != schemeHTTPS {
		return "", fmt.Errorf("%w: %q must start with http:// or https://", ErrInvalidURL, s)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", fmt.Errorf("%w: %q has no host name", ErrInvalidURL, s)
	}
	if u.User != nil {
		return "", fmt.Errorf("%w: %q must not carry a user name or password", ErrInvalidURL, s)
	}
	port := u.Port()
	if (scheme == schemeHTTP && port == defaultHTTPPort) || (scheme == schemeHTTPS && port == defaultHTTPSPort) {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	u.Scheme = scheme
	u.Host = host
	u.Fragment = ""
	u.RawFragment = ""
	if u.Path == "" {
		u.Path = "/"
		u.RawPath = ""
	}
	return u.String(), nil
}

// HostOf returns the host part of an already normalised URL, used to space requests per host.
func HostOf(normalised string) string {
	u, err := url.Parse(normalised)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
