package domain

import "testing"

func TestDetectChallenge(t *testing.T) {
	t.Parallel()
	pages := map[string]string{
		"Amazon CAPTCHA":       `<form action="/errors/validateCaptcha">`,
		"Amazon robot check":   `To discuss automated access to Amazon data please contact`,
		"Cloudflare challenge": `<TITLE>Just a moment...</TITLE>`,
		"DataDome CAPTCHA":     `<script src="https://geo.captcha-delivery.com/c.js">`,
	}
	for want, body := range pages {
		if got, ok := DetectChallenge(body); !ok || got != want {
			t.Errorf("DetectChallenge(%q) = %q, %v; want %q", body, got, ok, want)
		}
	}
	ordinary := `<html><title>Shop</title><form><div class="g-recaptcha"></div></form></html>`
	if name, ok := DetectChallenge(ordinary); ok {
		t.Errorf("ordinary page flagged as %q", name)
	}
}
