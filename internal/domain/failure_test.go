package domain

import (
	"strings"
	"testing"
)

func TestCategoryForStatus(t *testing.T) {
	t.Parallel()
	cases := map[int]Category{
		200: CategoryNone, 204: CategoryNone, 401: CategoryHTTP401, 403: CategoryHTTP403,
		404: CategoryHTTP404, 429: CategoryHTTP429, 500: CategoryHTTP500, 502: CategoryHTTP502,
		503: CategoryHTTP503, 504: CategoryHTTP504, 418: CategoryHTTPStatus, 300: CategoryHTTPStatus,
	}
	for code, want := range cases {
		if got := CategoryForStatus(code); got != want {
			t.Errorf("CategoryForStatus(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()
	for _, c := range Categories() {
		if Describe(c, 0) == "" {
			t.Errorf("category %q has no description", c)
		}
	}
	if got := Describe(CategoryHTTP503, 503); !strings.HasPrefix(got, "HTTP 503: ") {
		t.Errorf("Describe with status = %q", got)
	}
	if got := Describe(CategoryChallenge, 200); strings.HasPrefix(got, "HTTP") {
		t.Errorf("a 200 must not be presented as an error code: %q", got)
	}
	if Describe(CategoryNone, 200) != "" {
		t.Error("success has no failure description")
	}
	if len(Categories()) != 23 {
		t.Errorf("FR-040 lists 23 categories, got %d", len(Categories()))
	}
}
