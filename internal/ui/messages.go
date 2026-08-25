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
