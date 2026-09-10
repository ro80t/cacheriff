// Package textwrap hard-wraps single lines of already-rendered text
// (e.g. a table row with a long trailing file path) so they fit
// within a fixed column budget, without depending on any UI package.
package textwrap

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ContentLine hard-wraps line into chunks no wider than width,
// indenting every chunk after the first by indent spaces. Each break
// prefers landing right after a path separator ('/' or '\') so a
// wrapped path still reads as whole segments.
//
// Callers rendering into a fixed-size lipgloss box must wrap to its
// real usable width themselves: lipgloss's Height() doesn't clip
// overflow, so an unwrapped line would grow the box past its
// declared height.
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
// in s, or 0 if there's no good break point.
func lastPathSeparator(s []rune) int {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '/' || s[i] == '\\' {
			return i + 1
		}
	}
	return 0
}
