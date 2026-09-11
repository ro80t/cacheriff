package driver

import (
	"os/exec"
	"regexp"
	"strings"
)

// Columns are separated by runs of 2+ spaces since a path itself may
// contain single spaces (e.g. under "Program Files").
var pyLauncherFieldsRe = regexp.MustCompile(`\s{2,}`)

// discoverPyLauncherPythonDirs finds Pythons that may never have been
// added to PATH. Returns nil if py isn't installed.
func discoverPyLauncherPythonDirs() []string {
	out, err := exec.Command("py", "-0p").Output()
	if err != nil {
		return nil
	}
	return parsePyLauncherOutput(string(out))
}

// `py -0p` output looks like:
//
//	-V:3.13 *        C:\Users\me\AppData\Local\Programs\Python\Python313\python.exe
//	-V:2.7           C:\Python27\python.exe
func parsePyLauncherOutput(out string) []string {
	var dirs []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-V:") {
			continue
		}
		fields := pyLauncherFieldsRe.Split(line, -1)
		pythonExe := fields[len(fields)-1]
		// Always a Windows path regardless of host OS, so split on a
		// literal backslash rather than path/filepath.
		idx := strings.LastIndexByte(pythonExe, '\\')
		if idx < 0 {
			continue
		}
		dirs = append(dirs, pythonExe[:idx]+`\Scripts`)
	}
	return dirs
}
