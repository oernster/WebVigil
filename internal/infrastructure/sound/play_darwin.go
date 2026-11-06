//go:build darwin

package sound

// players: afplay is macOS's own command-line player.
var players = []command{{name: "afplay"}}
