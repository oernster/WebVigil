package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormaliseURL(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"HTTPS://Example.COM:443#top":        "https://example.com/",
		"http://example.com:80/a?b=1#x":      "http://example.com/a?b=1",
		"  https://example.com/Path  ":       "https://example.com/Path",
		"https://example.com:8443/x":         "https://example.com:8443/x",
		"http://[::1]:8080/":                 "http://[::1]:8080/",
		"https://[2001:db8::1]/":             "https://[2001:db8::1]/",
		"https://example.com?q=1":            "https://example.com/?q=1",
		"https://shop.example.co.uk/dp/ABC1": "https://shop.example.co.uk/dp/ABC1",
	}
	for in, want := range cases {
		got, err := NormaliseURL(in)
		if err != nil || got != want {
			t.Errorf("NormaliseURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestNormaliseURLRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"", "   ", "ftp://x", "javascript:alert(1)", "example.com", "https://",
		"https://user:pw@example.com/", "http://%zz", "https://" + strings.Repeat("a", MaxURLLength),
	} {
		if _, err := NormaliseURL(in); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("NormaliseURL(%q) error = %v, want ErrInvalidURL", in, err)
		}
	}
}

func TestHostOf(t *testing.T) {
	t.Parallel()
	if got := HostOf("https://example.com:8443/x"); got != "example.com" {
		t.Errorf("HostOf = %q", got)
	}
	if got := HostOf("http://%zz"); got != "" {
		t.Errorf("HostOf(bad) = %q, want empty", got)
	}
}
