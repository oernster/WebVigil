package domain

import (
	"fmt"
	"net/http"
)

// Outcome is the result class of the latest check.
type Outcome string

const (
	OutcomeNone    Outcome = ""
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)

// Category names why a check failed. CategoryNone accompanies a success.
type Category string

const (
	CategoryNone                   Category = ""
	CategoryDNS                    Category = "dns"
	CategoryConnectionRefused      Category = "connection_refused"
	CategoryConnectionTimeout      Category = "connection_timeout"
	CategoryTLS                    Category = "tls"
	CategoryHTTP401                Category = "http_401"
	CategoryHTTP403                Category = "http_403"
	CategoryHTTP404                Category = "http_404"
	CategoryHTTP429                Category = "http_429"
	CategoryHTTP500                Category = "http_500"
	CategoryHTTP502                Category = "http_502"
	CategoryHTTP503                Category = "http_503"
	CategoryHTTP504                Category = "http_504"
	CategoryHTTPStatus             Category = "http_status"
	CategoryRedirectLoop           Category = "redirect_loop"
	CategoryUnsupportedContentType Category = "unsupported_content_type"
	CategoryResponseTooLarge       Category = "response_too_large"
	CategoryParseFailure           Category = "parse_failure"
	CategorySelectorMissing        Category = "selector_missing"
	CategoryChallenge              Category = "challenge"
	CategoryUnexpectedStructure    Category = "unexpected_structure"
	CategoryInvalidContent         Category = "invalid_content"
	CategoryNetwork                Category = "network"
	CategoryCancelled              Category = "cancelled"
)

var statusCategories = map[int]Category{
	http.StatusUnauthorized:        CategoryHTTP401,
	http.StatusForbidden:           CategoryHTTP403,
	http.StatusNotFound:            CategoryHTTP404,
	http.StatusTooManyRequests:     CategoryHTTP429,
	http.StatusInternalServerError: CategoryHTTP500,
	http.StatusBadGateway:          CategoryHTTP502,
	http.StatusServiceUnavailable:  CategoryHTTP503,
	http.StatusGatewayTimeout:      CategoryHTTP504,
}

// IsSuccessStatus reports whether code is a 2xx status.
func IsSuccessStatus(code int) bool {
	return code >= http.StatusOK && code < http.StatusMultipleChoices
}

// CategoryForStatus classifies a final HTTP status. A 2xx is not a failure.
func CategoryForStatus(code int) Category {
	if IsSuccessStatus(code) {
		return CategoryNone
	}
	if c, ok := statusCategories[code]; ok {
		return c
	}
	return CategoryHTTPStatus
}

var categoryText = map[Category]string{
	CategoryDNS:                    "The host name could not be found (DNS)",
	CategoryConnectionRefused:      "The server refused the connection",
	CategoryConnectionTimeout:      "The connection or response timed out",
	CategoryTLS:                    "The secure connection failed certificate checks",
	CategoryHTTP401:                "The server asks for authentication",
	CategoryHTTP403:                "The server refused access",
	CategoryHTTP404:                "The page was not found",
	CategoryHTTP429:                "The server says too many requests",
	CategoryHTTP500:                "The server reported an internal error",
	CategoryHTTP502:                "The server reported a bad gateway",
	CategoryHTTP503:                "The server is unavailable",
	CategoryHTTP504:                "The server's gateway timed out",
	CategoryHTTPStatus:             "The server answered with an error status",
	CategoryRedirectLoop:           "Too many redirects",
	CategoryUnsupportedContentType: "The response is not text or HTML",
	CategoryResponseTooLarge:       "The response is larger than the size limit",
	CategoryParseFailure:           "The page could not be parsed",
	CategorySelectorMissing:        "A required selector matched nothing",
	CategoryChallenge:              "The site served a CAPTCHA or bot check instead of the page",
	CategoryUnexpectedStructure:    "The response did not have the expected structure",
	CategoryInvalidContent:         "Extraction produced no content",
	CategoryNetwork:                "A network error occurred",
	CategoryCancelled:              "The check was cancelled",
}

// Describe returns a one-line explanation of a failure, with the HTTP code where there is one.
func Describe(c Category, status int) string {
	text, ok := categoryText[c]
	if !ok {
		return ""
	}
	if status != 0 && !IsSuccessStatus(status) {
		return fmt.Sprintf("HTTP %d: %s", status, text)
	}
	return text
}

// Categories lists every failure category, for exhaustive tests and the help text.
func Categories() []Category {
	out := make([]Category, 0, len(categoryText))
	for c := range categoryText {
		out = append(out, c)
	}
	return out
}
