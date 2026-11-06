package domain

import "fmt"

// Theme is the appearance preference.
type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

// Settings are the application-wide preferences.
type Settings struct {
	Theme            Theme
	Paused           bool
	StartAtSignIn    bool
	SoundPath        string // empty for the built-in chime
	RepeatAlarm      bool
	ForegroundDialog bool
}

// DefaultSettings returns the settings of a first run.
func DefaultSettings() Settings {
	return Settings{Theme: ThemeSystem}
}

// Validate refuses a theme the application does not know.
func (s Settings) Validate() error {
	switch s.Theme {
	case ThemeSystem, ThemeLight, ThemeDark:
		return nil
	}
	return fmt.Errorf("%w: unknown theme %q", ErrInvalidSettings, s.Theme)
}

// Retention and fetch limits (REQUIREMENTS.md section 1.4).
const (
	KeepChecksPerSite    = 1000
	KeepSnapshotsPerSite = 50
	MaxResponseBytes     = 8 << 20
	MaxRedirects         = 10
)
