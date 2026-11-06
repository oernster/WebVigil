//go:build linux

package sound

// players, in order: paplay reaches PulseAudio and PipeWire's Pulse server alike, pw-play reaches
// PipeWire, aplay reaches ALSA directly and gst-play-1.0 is GStreamer's player. Which of them a given
// desktop or the flatpak runtime carries varies, so the first one present is used.
var players = []command{
	{name: "paplay"},
	{name: "pw-play"},
	{name: "aplay", args: []string{"-q"}},
	{name: "gst-play-1.0", args: []string{"--no-interactive", "-q"}},
}
