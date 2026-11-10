//go:build unix

package fleet

import "syscall"

func (OSProbe) DiskFreeMB(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 1 << 20
	}
	return int64(st.Bavail) * int64(st.Bsize) >> 20
}
