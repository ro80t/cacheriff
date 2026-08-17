package driver

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"cacheriff/internal/platform"
)

type base struct {
	id          string
	name        string
	binary      string // CLI binary looked up on PATH for Available()
	supportedOS []platform.OS
	dirs        []string // LocalArtifactDirNames
}

func (b base) ID() string   { return b.id }
func (b base) Name() string { return b.name }

func (b base) Available() bool {
	_, err := exec.LookPath(b.binary)
	return err == nil
}

func (b base) SupportedOS() []platform.OS {
	return b.supportedOS
}

func (b base) LocalArtifactDirNames() []string {
	return b.dirs
}

// dirSize walks path and sums the size of every regular file under it.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip entries we can't stat (permissions, races) rather
			// than failing the whole walk.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size, err
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
