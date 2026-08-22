// Package textwrap hard-wraps single lines of already-rendered text
// (e.g. a table row with a long trailing file path) so they fit
// within a fixed column budget, without depending on any UI package.
package textwrap

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ContentLine hard-wraps line into one or more chunks so that none of
// them exceeds width columns, indenting every chunk after the first
// by indent spaces so a long value (a full file path, a long module
// name) reads as a natural continuation rather than being cut off.
// Each break prefers landing right after a path separator ('/' or
// '\') over an arbitrary column, so a wrapped path still reads as
// whole segments rather than being sliced mid-word.
//
// Callers that hand wrapped text to a fixed-size box (e.g. a
// lipgloss.Style with Border/Padding and an explicit Width/Height)
// should wrap it down to that box's real usable width themselves
// before rendering: lipgloss's own Height() doesn't clip content
// taller than it, so a too-wide line left for lipgloss to wrap on its
// own would grow the box past its declared height and misalign it
// against neighboring boxes.
func ContentLine(line string, width, indent int) []string {
	if width <= 0 || lipgloss.Width(line) <= width {
		return []string{line}
	}
	if indent < 0 || indent >= width {
		indent = 0
	}

	runes := []rune(line)
	var out []string
	for i := 0; len(runes) > 0; i++ {
		w := width
		pad := ""
		if i > 0 {
			w = width - indent
			pad = strings.Repeat(" ", indent)
		}
		if w > len(runes) {
			out = append(out, pad+string(runes))
			break
		}

		cut := w
		if at := lastPathSeparator(runes[:w]); at > 0 {
			cut = at
		}

		out = append(out, pad+string(runes[:cut]))
		runes = runes[cut:]
	}
	return out
}

// lastPathSeparator returns the index just after the last '/' or '\'
// in s, so a wrap can break there instead of mid-segment. It returns
// 0 (meaning "no good break point") when s has no separator, or when
// the only one found sits right at the start (which would produce a
// degenerate near-empty chunk).
func lastPathSeparator(s []rune) int {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '/' || s[i] == '\\' {
			return i + 1
		}
	}
	return 0
}
