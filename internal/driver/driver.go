// Package driver defines the interface implemented by each supported
// package manager (cargo, npm, ...) and the shared helpers they use to
// report caches, globally installed packages, and local project
// artifacts that cacheriff can help the user remove.
package driver

import (
	"context"

	"github.com/ro80t/cacheriff/internal/platform"
)

// EntryKind distinguishes the different things a driver can report.
type EntryKind int

const (
	KindCache EntryKind = iota
	KindGlobalPackage
	KindLocalPackage
)

func (k EntryKind) String() string {
	switch k {
	case KindCache:
		return "cache"
	case KindGlobalPackage:
		return "global package"
	case KindLocalPackage:
		return "local package"
	default:
		return "unknown"
	}
}

// Entry is a single removable thing reported by a driver: a cache
// directory or an installed package.
type Entry struct {
	Name    string
	Version string // empty when not applicable
	Path    string
	Kind    EntryKind
	Size    int64 // bytes; -1 if unknown
}

// Driver is implemented once per package manager. All methods must be
// safe to call even when the underlying package manager is not
// installed; Available reports that up front so callers can skip it.
//
// Concrete drivers embed base (see base.go) for the common plumbing
// and only implement the parts that differ per package manager:
// CacheDir, CacheEntries, GlobalInstallDir, GlobalPackages,
// LocalPackages, and Remove.
type Driver interface {
	ID() string
	Name() string
	Available() bool
	SupportedOS() []platform.OS
	CacheDir(ctx context.Context) (string, error)
	CacheEntries(ctx context.Context) ([]Entry, error)
	GlobalInstallDir(ctx context.Context) (string, error)
	GlobalPackages(ctx context.Context) ([]Entry, error)
	// LocalInstallDir reports the directory under root where this
	// package manager installs a project's local packages, and
	// whether it has one at all (e.g. cargo has none).
	LocalInstallDir(root string) (dir string, ok bool)
	// LocalPackages returns (nil, nil) when root has nothing for
	// this package manager to report.
	LocalPackages(ctx context.Context, root string) ([]Entry, error)
	LocalArtifactDirNames() []string
	// Remove prefers the package manager's own uninstall command for
	// installed packages, so its metadata stays consistent.
	Remove(ctx context.Context, e Entry) error
}

// All returns every driver cacheriff knows about, regardless of
// whether the corresponding package manager is installed on this
// machine. Callers should filter by Available().
func All() []Driver {
	return []Driver{
		NewCargoDriver(),
		NewGoDriver(),
		NewNPMDriver(),
		NewPnpmDriver(),
		NewYarnDriver(),
		NewBunDriver(),
		NewDenoDriver(),
		NewNixDriver(),
	}
}

func SupportsCurrentOS(d Driver) bool {
	current := platform.Current()
	for _, o := range d.SupportedOS() {
		if o == current {
			return true
		}
	}
	return false
}
