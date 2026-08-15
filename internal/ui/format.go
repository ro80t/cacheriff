package ui

import "fmt"

// formatBytes renders n as a human-readable size (e.g. "12.3 MB").
// Negative values mean "unknown" (a driver couldn't determine a size).
func formatBytes(n int64) string {
	if n < 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	units := "KMGTPE"
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), units[exp])
}
