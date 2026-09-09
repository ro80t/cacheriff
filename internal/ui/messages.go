package ui

import "cacheriff/internal/driver"

// driverDataMsg carries the result of loading a driver's cache
// entries and global/local packages. gen guards against a stale load
// finishing after the user has already selected a different driver.
type driverDataMsg struct {
	gen    int
	cache  []driver.Entry
	global []driver.Entry
	local  []driver.Entry
	err    error
}

// packageRemovedMsg carries the result of running a driver's uninstall
// command for a single package. gen guards against a stale result
// finishing after the user has since moved on (e.g. selected a
// different driver).
type packageRemovedMsg struct {
	gen   int
	entry driver.Entry
	err   error
}
