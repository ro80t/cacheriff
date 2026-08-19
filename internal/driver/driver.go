// Package driver defines the interface implemented by each supported
// package manager (cargo, npm, ...) and the shared helpers they use to
// report caches, globally installed packages, and local project
// artifacts that cacheriff can help the user remove.
package driver

import (
	"context"

	"cacheriff/internal/platform"
)

// EntryKind distinguishes the different things a driver can report.
type EntryKind int

const (
	// KindCache is a shared, machine-wide download/build cache
	// (e.g. cargo's registry cache, npm's content-addressable cache).
	KindCache EntryKind = iota
	// KindGlobalPackage is a package installed globally via the
	// package manager (e.g. `cargo install`, `npm install -g`).
	KindGlobalPackage
	// KindLocalPackage is a package installed into a single project's
	// local install directory (e.g. a package under a project's
	// node_modules, or a dependency pinned in a project's lockfile).
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
	Version string // empty when not applicable (e.g. a cache directory)
	Path    string
	Kind    EntryKind
	Size    int64 // bytes; -1 if unknown/not computed
}

// Driver is implemented once per package manager. All methods must be
// safe to call even when the underlying package manager is not
// installed; Available reports that up front so callers can skip it.
//
// Concrete drivers embed base (see base.go) to pick up the common
// plumbing (ID, Name, Available, SupportedOS, LocalArtifactDirNames)
// and only implement the parts that genuinely differ between package
// managers: CacheDir, CacheEntries, GlobalInstallDir, GlobalPackages,
// LocalInstallDir, LocalPackages, and Remove.
type Driver interface {
	// ID is a stable, lowercase identifier, e.g. "cargo", "npm".
	ID() string
	// Name is a human-readable display name, e.g. "Cargo", "npm".
	Name() string
	// Available reports whether the package manager's CLI is
	// present on this machine.
	Available() bool
	// SupportedOS lists the operating systems this package manager
	// runs on. Cross-platform tools like cargo/npm list all of them;
	// OS-specific tools (e.g. Homebrew, winget) should restrict this.
	SupportedOS() []platform.OS
	// CacheDir reports the root directory of this package manager's
	// shared, machine-wide cache (e.g. npm's `cache` config value,
	// cargo's CARGO_HOME). Callers that just want the individual
	// cache entries (with sizes) should use CacheEntries instead.
	CacheDir(ctx context.Context) (string, error)
	// CacheEntries reports the shared, machine-wide caches this
	// package manager maintains (downloaded archives, build/index
	// caches, etc).
	CacheEntries(ctx context.Context) ([]Entry, error)
	// GlobalInstallDir reports the directory this package manager
	// installs global/system-wide packages into (e.g. `npm root -g`,
	// cargo's CARGO_HOME/bin). Callers that just want the individual
	// installed packages (with sizes) should use GlobalPackages
	// instead.
	GlobalInstallDir(ctx context.Context) (string, error)
	// GlobalPackages reports packages installed globally/system-wide
	// via this package manager.
	GlobalPackages(ctx context.Context) ([]Entry, error)
	// LocalInstallDir reports the directory under root where this
	// package manager installs a single project's local packages
	// (e.g. root/node_modules for npm), and whether it has one at
	// all: some package managers (e.g. cargo) have no per-project
	// install directory because resolved dependencies live only in
	// the shared cache.
	LocalInstallDir(root string) (dir string, ok bool)
	// LocalPackages reports the packages a project rooted at root
	// depends on locally (e.g. the contents of root's node_modules
	// for npm, or the crates pinned in root's Cargo.lock for cargo).
	// It returns (nil, nil) when root has nothing for this package
	// manager to report.
	LocalPackages(ctx context.Context, root string) ([]Entry, error)
	// LocalArtifactDirNames lists directory names that mark a
	// project-local install/build artifact for this package manager
	// (e.g. "node_modules" for npm, "target" for cargo), for use by
	// a filesystem scanner that looks for them under a project root.
	LocalArtifactDirNames() []string
	// Remove deletes the given entry. For caches this is typically a
	// plain filesystem removal; for installed packages it prefers
	// the package manager's own uninstall command so its metadata
	// stays consistent.
	Remove(ctx context.Context, e Entry) error
}

// All returns every driver cacheriff knows about, regardless of
// whether the corresponding package manager is installed on this
// machine. Callers should filter by Available().
func All() []Driver {
	return []Driver{
		NewCargoDriver(),
		NewNPMDriver(),
	}
}

// SupportsCurrentOS reports whether d declares support for the OS
// cacheriff is currently running on.
func SupportsCurrentOS(d Driver) bool {
	current := platform.Current()
	for _, o := range d.SupportedOS() {
		if o == current {
			return true
		}
	}
	return false
}
