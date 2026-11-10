//go:build !unix

package fleet

// DiskFreeMB is unknown here: report plenty, as the unix probe does when statfs fails.
func (OSProbe) DiskFreeMB(string) int64 { return 1 << 20 }
