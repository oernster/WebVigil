// Package sound gives WebVigil its alarm. The built-in chime is synthesised rather than shipped as
// an asset, so it is a pure function of the named constants here: no binary blob in the repository,
// nothing to resolve at runtime and a result that can be unit tested. The engine is ported from
// PigeonPost; the score in alarm.go is WebVigil's own.
package sound

import (
	"encoding/binary"
	"math"
)

// PCM format of the rendered chime: CD-quality mono, which every platform plays without a codec.
const (
	sampleRate    = 44100
	bitsPerSample = 16
	channels      = 1
	bytesPerSlot  = bitsPerSample / 8
	maxSample     = 32767.0
)

// peakLevel keeps the chime below full scale; tailFadeSecs removes the click of a hard end.
const (
	peakLevel    = 0.4
	tailFadeSecs = 0.02
)

// partial is one overtone of a note: a multiple of the fundamental at a fraction of its amplitude.
type partial struct {
	ratio float64
	level float64
}

// note is one struck tone.
type note struct {
	startSecs  float64
	freq       float64
	level      float64
	attackSecs float64
	decaySecs  float64 // exponential time constant, not a hard cutoff
	partials   []partial
}

// render sums the notes into samples in the range -1 to 1.
func render(lengthSecs float64, notes []note) []float64 {
	total := int(lengthSecs * sampleRate)
	buf := make([]float64, total)
	for _, n := range notes {
		start := int(n.startSecs * sampleRate)
		for i := start; i < total; i++ {
			t := float64(i-start) / sampleRate
			buf[i] += n.level * envelope(t, n.attackSecs, n.decaySecs) * timbre(t, n.freq, n.partials)
		}
	}
	fadeTail(buf)
	return normalise(buf)
}

func envelope(t, attackSecs, decaySecs float64) float64 {
	if t < attackSecs {
		return t / attackSecs
	}
	return math.Exp(-(t - attackSecs) / decaySecs)
}

func timbre(t, freq float64, partials []partial) float64 {
	v := math.Sin(2 * math.Pi * freq * t)
	for _, p := range partials {
		v += p.level * math.Sin(2*math.Pi*freq*p.ratio*t)
	}
	return v
}

// fadeTail ramps the last few milliseconds to exact silence.
func fadeTail(buf []float64) {
	fade := int(tailFadeSecs * sampleRate)
	if fade > len(buf) {
		return
	}
	last := len(buf) - 1
	for i := len(buf) - fade; i < len(buf); i++ {
		buf[i] *= float64(last-i) / float64(fade)
	}
}

// normalise scales the buffer so its loudest point sits at peakLevel, so it cannot clip.
func normalise(buf []float64) []float64 {
	peak := 0.0
	for _, v := range buf {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak == 0 {
		return buf
	}
	for i := range buf {
		buf[i] *= peakLevel / peak
	}
	return buf
}

// encodeWAV wraps the samples as 16-bit PCM in the canonical 44-byte RIFF/WAVE header.
func encodeWAV(samples []float64) []byte {
	const (
		headerSize    = 44
		fmtChunkSize  = 16
		pcmFormat     = 1
		riffPrefixLen = 8
	)
	dataSize := len(samples) * bytesPerSlot * channels
	out := make([]byte, 0, headerSize+dataSize)
	out = append(out, []byte("RIFF")...)
	out = binary.LittleEndian.AppendUint32(out, uint32(headerSize-riffPrefixLen+dataSize))
	out = append(out, []byte("WAVEfmt ")...)
	out = binary.LittleEndian.AppendUint32(out, fmtChunkSize)
	out = binary.LittleEndian.AppendUint16(out, pcmFormat)
	out = binary.LittleEndian.AppendUint16(out, channels)
	out = binary.LittleEndian.AppendUint32(out, sampleRate)
	out = binary.LittleEndian.AppendUint32(out, sampleRate*channels*bytesPerSlot)
	out = binary.LittleEndian.AppendUint16(out, channels*bytesPerSlot)
	out = binary.LittleEndian.AppendUint16(out, bitsPerSample)
	out = append(out, []byte("data")...)
	out = binary.LittleEndian.AppendUint32(out, uint32(dataSize))
	for _, s := range samples {
		out = binary.LittleEndian.AppendUint16(out, uint16(int16(s*maxSample)))
	}
	return out
}
