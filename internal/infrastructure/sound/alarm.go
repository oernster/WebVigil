package sound

import "sync"

// The alarm is two strikes of a small bell, a major third apart, rising: the bell in the icon.
// The inharmonic partials (2.76 and 5.40 of the fundamental) are the classic struck-bell spectrum,
// which is what makes it read as a bell rather than as a beep.
const (
	alarmLengthSecs  = 1.6
	firstStrikeSecs  = 0.0
	secondStrikeSecs = 0.22
	bellLowHz        = 1046.5 // C6
	bellHighHz       = 1318.5 // E6
	strikeAttackSecs = 0.004
	strikeDecaySecs  = 0.42
	firstLevel       = 0.85
	secondLevel      = 1.0
)

var bellPartials = []partial{{ratio: 2.76, level: 0.35}, {ratio: 5.40, level: 0.12}}

var (
	alarmOnce sync.Once
	alarmWAV  []byte
)

// AlarmWAV returns the built-in alarm as a WAV image, rendered once and cached for the process.
func AlarmWAV() []byte {
	alarmOnce.Do(func() {
		alarmWAV = encodeWAV(render(alarmLengthSecs, []note{
			{firstStrikeSecs, bellLowHz, firstLevel, strikeAttackSecs, strikeDecaySecs, bellPartials},
			{secondStrikeSecs, bellHighHz, secondLevel, strikeAttackSecs, strikeDecaySecs, bellPartials},
		}))
	})
	return alarmWAV
}
