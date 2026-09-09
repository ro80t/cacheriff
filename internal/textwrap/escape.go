package textwrap

import (
	"fmt"
	"strings"
)

// EscapeArg validates that s is safe to pass as a single argv element
// to an external command (a package manager's own uninstall command,
// e.g. `npm uninstall -g <name>`), and returns it unchanged if so.
//
// Every command a driver's Remove method runs goes through
// exec.CommandContext(ctx, binary, args...), which hands each element
// of args to the OS as a discrete argv entry rather than interpolating
// it into a shell string. That means classic shell-metacharacter
// injection (";", "|", "`", "$()", ...) cannot happen here regardless
// of s's contents - there is no shell in the loop to interpret them.
//
// The risk EscapeArg actually guards against is argument injection: a
// value that begins with "-" can be misread by the target command as
// a flag instead of a plain name (e.g. a package literally named
// "--force" turning `npm uninstall -g --force` into something other
// than what it looks like), and control characters/NUL bytes have no
// legitimate place in a package name either. EscapeArg rejects such
// values outright rather than rewriting them: silently stripping or
// quoting characters out of a package name would just make it stop
// matching anything real, which is worse than failing loudly.
func EscapeArg(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("argument is empty")
	}
	if strings.HasPrefix(s, "-") {
		return "", fmt.Errorf("argument %q looks like a command flag, not a name", s)
	}
	for _, r := range s {
		if r == 0 || r == 0x7f || (r < 0x20 && r != '\t') {
			return "", fmt.Errorf("argument %q contains a control character", s)
		}
	}
	return s, nil
}
