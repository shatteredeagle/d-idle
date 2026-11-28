//go:build unix

package savesafe

import "syscall"

// FreeBytes is the space a normal user can still write under path (statfs).
func FreeBytes(path string) (uint64, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, err
	}
	return uint64(s.Bavail) * uint64(s.Bsize), nil
}
