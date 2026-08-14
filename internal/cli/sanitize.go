package cli

import "strings"

// sanitizeTerminal strips ASCII control characters (except '\n' and '\t')
// from s before it's written to the terminal in a plain-text output path.
//
// Repo metadata (name, description, topics, README) comes straight from
// GitHub — anyone who can create a public repo controls this text, and a
// naive fmt.Print of it would execute any ANSI/OSC escape sequences it
// contains in the user's terminal (title spoofing, cursor tricks that
// hide/overwrite output, disguised OSC 8 hyperlinks for phishing, etc.).
// Every such sequence starts with an ESC (0x1B) or another C0 control
// byte, so stripping those neutralizes them regardless of the specific
// sequence; whatever's left prints as harmless literal text.
//
// This only matters for the plain-text display path (show/list/search's
// table output) — --json is unaffected and already safe, since
// encoding/json escapes control characters into \u00XX sequences rather
// than emitting them raw.
func sanitizeTerminal(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || r == 0x7f:
			return -1
		default:
			return r
		}
	}, s)
}
