package sound

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestAlarmWAV(t *testing.T) {
	t.Parallel()
	wav := AlarmWAV()
	const header = 44
	samples := int(alarmLengthSecs * sampleRate)
	if string(wav[:4]) != "RIFF" || string(wav[8:16]) != "WAVEfmt " || len(wav) != header+samples*bytesPerSlot {
		t.Fatalf("bad WAV: %d bytes", len(wav))
	}
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != sampleRate {
		t.Errorf("sample rate = %d", got)
	}
	peak := 0
	for i := header; i < len(wav); i += bytesPerSlot {
		v := int(int16(binary.LittleEndian.Uint16(wav[i:])))
		peak = max(peak, abs(v))
	}
	if want := peakLevel * maxSample; math.Abs(float64(peak)-want) > 1 {
		t.Errorf("peak = %d, want about %.0f", peak, want)
	}
	last := int16(binary.LittleEndian.Uint16(wav[len(wav)-bytesPerSlot:]))
	if last != 0 {
		t.Errorf("the alarm does not end on silence: %d", last)
	}
	if &AlarmWAV()[0] != &wav[0] {
		t.Error("the alarm is rendered more than once")
	}
}

func TestNormaliseSilence(t *testing.T) {
	t.Parallel()
	if got := normalise([]float64{0, 0}); got[0] != 0 {
		t.Error("silence changed")
	}
	fadeTail([]float64{1}) // shorter than the fade: left alone
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
