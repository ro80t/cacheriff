package ui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"

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

// startRemove should batch every marked package rather than just the one
// under the cursor, once the user has toggled marks with Select.
func TestStartRemoveUsesMarkedSelection(t *testing.T) {
	pkgs := []driver.Entry{
		{Name: "pkg0", Version: "1.0.0", Kind: driver.KindGlobalPackage},
		{Name: "pkg1", Version: "1.0.0", Kind: driver.KindGlobalPackage},
		{Name: "pkg2", Version: "1.0.0", Kind: driver.KindGlobalPackage},
	}

	m := Model{
		ready:          true,
		state:          loadDone,
		drivers:        []driverItem{{driver: driver.NewNPMDriver(), available: true}},
		activeIdx:      0,
		scope:          scopeGlobal,
		globalPackages: pkgs,
		viewport:       viewport.New(0, 0),
	}

	m.packageCursor = 0
	m = m.toggleSelected()
	m.packageCursor = 2
	m = m.toggleSelected()

	mdl, _ := m.startRemove()
	m = mdl.(Model)

	if !m.confirmRemove {
		t.Fatalf("expected confirmRemove to be true")
	}
	if len(m.pendingRemove) != 2 {
		t.Fatalf("got %d pending removals, want 2", len(m.pendingRemove))
	}
	if m.pendingRemove[0] != pkgs[0] || m.pendingRemove[1] != pkgs[2] {
		t.Fatalf("got pendingRemove=%v, want [pkg0, pkg2]", m.pendingRemove)
	}
}

// With nothing marked, startRemove falls back to the single entry under
// the cursor, matching the pre-multi-select behavior.
func TestStartRemoveFallsBackToCursorWhenNothingMarked(t *testing.T) {
	pkgs := []driver.Entry{
		{Name: "pkg0", Version: "1.0.0", Kind: driver.KindGlobalPackage},
		{Name: "pkg1", Version: "1.0.0", Kind: driver.KindGlobalPackage},
	}

	m := Model{
		ready:          true,
		state:          loadDone,
		drivers:        []driverItem{{driver: driver.NewNPMDriver(), available: true}},
		activeIdx:      0,
		scope:          scopeGlobal,
		globalPackages: pkgs,
		packageCursor:  1,
		viewport:       viewport.New(0, 0),
	}

	mdl, _ := m.startRemove()
	m = mdl.(Model)

	if len(m.pendingRemove) != 1 || m.pendingRemove[0] != pkgs[1] {
		t.Fatalf("got pendingRemove=%v, want [pkg1]", m.pendingRemove)
	}
}
