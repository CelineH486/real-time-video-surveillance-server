//go:build windows

package main

// Production uses Linux. Windows development has no portable directory fsync.
func syncDirectory(string) error { return nil }
