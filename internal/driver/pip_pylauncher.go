package driver

import (
	"os/exec"
	"path/filepath"
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
		if pythonExe == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(filepath.Dir(pythonExe), "Scripts"))
	}
	return dirs
}
