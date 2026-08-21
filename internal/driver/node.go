package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// splitPackageSpec splits a combined "name@version" spec, as printed
// by tools like yarn and bun in their dependency tree output, into
// its name and version. It splits at the last "@" rather than the
// first so that scoped package names (e.g. "@scope/name@1.2.3")
// resolve correctly.
func splitPackageSpec(spec string) (name, version string) {
	i := strings.LastIndex(spec, "@")
	if i <= 0 {
		return spec, ""
	}
	return spec[:i], spec[i+1:]
}

// resolvedPackageVersion reads the "version" field out of a
// package's own package.json under pkgDir, returning "" if it can't
// be determined (the package isn't installed there, or its
// package.json is missing/malformed).
func resolvedPackageVersion(pkgDir string) string {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Version
}
