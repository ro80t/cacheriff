package ui

import "github.com/ro80t/cacheriff/internal/driver"

// gen guards against a stale load finishing after the user has
// already selected a different driver.
type driverDataMsg struct {
	gen    int
	cache  []driver.Entry
	global []driver.Entry
	local  []driver.Entry
	err    error
}

// gen guards against a stale result finishing after the user has
// since moved on.
type packageRemovedMsg struct {
	gen   int
	entry driver.Entry
	err   error
}
