package domain

import (
	"errors"
	"testing"
)

func TestSettingsValidate(t *testing.T) {
	t.Parallel()
	if DefaultSettings().Theme != ThemeSystem {
		t.Error("default theme is not system")
	}
	for _, th := range []Theme{ThemeSystem, ThemeLight, ThemeDark} {
		if err := (Settings{Theme: th}).Validate(); err != nil {
			t.Errorf("%s refused: %v", th, err)
		}
	}
	if err := (Settings{Theme: "neon"}).Validate(); !errors.Is(err, ErrInvalidSettings) {
		t.Errorf("unknown theme: %v", err)
	}
}
