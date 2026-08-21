package driver

import (
	"context"
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

// dirSize walks path and sums the size of every regular file under
// it. Some caches (e.g. cargo's registry/src, npm/bun/pnpm/yarn's
// content-addressable stores) hold enough files that a full walk can
// take a long time, so this checks ctx on every entry and aborts as
// soon as it's canceled or its deadline passes, rather than running
// to completion regardless of the caller's timeout.
func dirSize(ctx context.Context, path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
