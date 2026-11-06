package domain

import "strings"

// challengeMarker is a lower-case fragment that appears in a bot-check or CAPTCHA interstitial and
// not in an ordinary page. Each is specific to the interstitial itself: a generic marker such as a
// reCAPTCHA script tag is deliberately absent, because ordinary pages embed one on a contact form.
type challengeMarker struct {
	name   string
	needle string
}

var challengeMarkers = []challengeMarker{
	{"Amazon CAPTCHA", "/errors/validatecaptcha"},
	{"Amazon robot check", "to discuss automated access to amazon data"},
	{"Cloudflare challenge", "/cdn-cgi/challenge-platform/"},
	{"Cloudflare challenge", "<title>just a moment...</title>"},
	{"Cloudflare block", "attention required! | cloudflare"},
	{"DataDome CAPTCHA", "geo.captcha-delivery.com"},
	{"PerimeterX CAPTCHA", "px-captcha"},
	{"Imperva block", "_incapsula_resource"},
	{"AWS WAF CAPTCHA", "awswaf captcha"},
	{"Generic bot check", "are you a robot?"},
	{"Generic bot check", "verify you are human"},
}

// DetectChallenge reports whether body is a challenge page and names the marker that matched.
// Detection is a fixed table, so an unknown interstitial is not recognised; a required selector
// or field is the second line of defence against that.
func DetectChallenge(body string) (string, bool) {
	lower := strings.ToLower(body)
	for _, m := range challengeMarkers {
		if strings.Contains(lower, m.needle) {
			return m.name, true
		}
	}
	return "", false
}
