package textwrap

import (
	"fmt"
	"strings"
)

// EscapeArg validates that s is safe to pass as a single argv element
// to an external command, returning it unchanged if so. Every args
// element passed through exec.CommandContext reaches the OS directly
// rather than through a shell, so classic shell-metacharacter
// injection can't happen here regardless of s's contents. What
// EscapeArg guards against is argument injection: a value starting
// with "-" could be misread as a flag (e.g. a package named
// "--force"), and control characters have no legitimate place in a
// name either. It rejects such values rather than rewriting them,
// since silently stripping characters would just make s stop
// matching anything real.
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
