// Package platform identifies the operating system cacheriff is
// running on, so drivers for OS-specific package managers (e.g.
// Homebrew on macOS/Linux, winget/scoop on Windows) can declare where
// they apply.
package platform

import "runtime"

// OS identifies a broad operating system family.
type OS int

const (
	Unknown OS = iota
	Windows
	MacOS
	Linux
)

func (o OS) String() string {
	switch o {
	case Windows:
		return "Windows"
	case MacOS:
		return "macOS"
	case Linux:
		return "Linux"
	default:
		return "Unknown"
	}
}

// Current returns the OS cacheriff is currently running on.
func Current() OS {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "darwin":
		return MacOS
	case "linux":
		return Linux
	default:
		return Unknown
	}
}

func IsWindows() bool { return Current() == Windows }
func IsMacOS() bool   { return Current() == MacOS }
func IsLinux() bool   { return Current() == Linux }
