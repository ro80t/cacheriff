package ui

import (
	"fmt"
	"testing"

	"github.com/ro80t/cacheriff/internal/driver"
)

// Regression test: navigating back to the first package used to leave
// the viewport scrolled to that entry's own line, permanently hiding
// the Paths section above it once the list had been scrolled down far
// enough (see ensureCursorVisible).
func TestEnsureCursorVisibleScrollsToTopAtFirstEntry(t *testing.T) {
	var pkgs []driver.Entry
	for i := range 50 {
		pkgs = append(pkgs, driver.Entry{Name: fmt.Sprintf("pkg%02d", i), Version: "1.0.0", Size: 1024})
	}

	m := Model{
		width:     100,
		height:    20, // small enough that the list overflows the viewport
		ready:     true,
		state:     loadDone,
		drivers:   []driverItem{{driver: driver.NewNPMDriver(), available: true}},
		activeIdx: 0,
		cache: []driver.Entry{
			{Name: "Global install", Path: `/usr/lib/node_modules`, Size: -1},
			{Name: "npm cache", Path: `/home/me/.npm`, Size: 123456789},
		},
		globalPackages: pkgs,
	}
	m.applyLayout()

	for range len(pkgs) {
		m = m.movePackageCursor(1)
	}
	if m.viewport.YOffset == 0 {
		t.Fatalf("expected the viewport to have scrolled down")
	}

	for range len(pkgs) {
		m = m.movePackageCursor(-1)
	}
	if m.packageCursor != 0 {
		t.Fatalf("got packageCursor=%d, want 0", m.packageCursor)
	}
	if m.viewport.YOffset != 0 {
		t.Errorf("got YOffset=%d, want 0 (Paths section should be visible again)", m.viewport.YOffset)
	}
}
