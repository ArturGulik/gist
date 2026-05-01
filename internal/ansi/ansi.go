package ansi

import (
	"fmt"
	"os"
	"strings"
)

// SGR codes. Exported because config defaults assemble Styles from them.
const (
	SgrReset  = "\x1b[0m"
	SgrBold   = "\x1b[1m"
	SgrDim    = "\x1b[2m"
	SgrItalic = "\x1b[3m"
	SgrStrike = "\x1b[9m"

	FgRed     = "\x1b[31m"
	FgGreen   = "\x1b[32m"
	FgYellow  = "\x1b[33m"
	FgBlue    = "\x1b[34m"
	FgMagenta = "\x1b[35m"
	FgCyan    = "\x1b[36m"
)

// Style is a list of SGR escape codes that compose a visual style. Empty
// means no styling. Stored as a slice (not a struct) so DefaultConfig can
// build them inline as Style{SgrBold, FgGreen}.
type Style []string

// S is a convenience constructor: ansi.S(SgrBold, FgGreen).
func S(codes ...string) Style { return Style(codes) }

// Pen renders styles. Color=false makes every Apply/Format/Hyperlink call
// pass through plain text — the single switch that gates all colorization.
type Pen struct {
	Color bool
}

// Apply wraps text in the style's escape sequence. No-op when color is off.
func (p Pen) Apply(s Style, text string) string {
	if !p.Color || len(s) == 0 {
		return text
	}
	return strings.Join(s, "") + text + SgrReset
}

// Format is Apply on Sprintf — `pen.Format(s, "%s%d", glyph, n)`.
func (p Pen) Format(s Style, format string, args ...any) string {
	return p.Apply(s, fmt.Sprintf(format, args...))
}

// Style applies an ad-hoc list of SGR codes to text. Cheaper for one-off
// styling than constructing a Style{...} value first.
func (p Pen) Style(text string, codes ...string) string {
	return p.Apply(Style(codes), text)
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink. No-op when color is off.
func (p Pen) Hyperlink(url, text string) string {
	if !p.Color {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// DetectColor resolves whether color output should be on. Rules:
//  1. NO_COLOR set (any value) -> off (https://no-color.org)
//  2. GIST_COLOR=always -> on
//  3. GIST_COLOR=never  -> off
//  4. stdout is a TTY  -> on
//  5. otherwise        -> off
func DetectColor() bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	switch os.Getenv("GIST_COLOR") {
	case "always":
		return true
	case "never":
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// VisibleWidth returns the printed width of s in cells, skipping ANSI escape
// sequences. Recognizes CSI (ESC [ ... final) and bare two-byte ESC sequences;
// terminates CSI on any byte in the ECMA-48 final-byte range (0x40–0x7E),
// which covers letters, '~', '@', etc. Assumes single-width runes.
func VisibleWidth(s string) int {
	const (
		stNormal = iota
		stEsc    // saw ESC, awaiting introducer
		stCSI    // inside CSI params/intermediates, awaiting final
	)
	n, state := 0, stNormal
	for _, r := range s {
		switch state {
		case stNormal:
			if r == '\x1b' {
				state = stEsc
				continue
			}
			n++
		case stEsc:
			if r == '[' {
				state = stCSI
			} else {
				// Bare two-byte ESC sequence (e.g. ESC c). Done.
				state = stNormal
			}
		case stCSI:
			if r >= 0x40 && r <= 0x7E {
				state = stNormal
			}
		}
	}
	return n
}
