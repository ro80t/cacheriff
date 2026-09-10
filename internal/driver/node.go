package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// splitPackageSpec splits "name@version" at the last "@" rather than
// the first, so scoped names (e.g. "@scope/name@1.2.3") resolve correctly.
func splitPackageSpec(spec string) (name, version string) {
	i := strings.LastIndex(spec, "@")
	if i <= 0 {
		return spec, ""
	}
	return spec[:i], spec[i+1:]
}

// resolvedPackageVersion reads pkgDir/package.json's "version" field,
// returning "" if it can't be determined.
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
