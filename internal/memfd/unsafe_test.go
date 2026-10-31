package memfd

import "unsafe"

// unsafeBytes views memory we mmap'ed ourselves at a fixed address (not Go-managed).
func unsafeBytes(addr uintptr, n int) []byte {
	var p unsafe.Pointer
	*(*uintptr)(unsafe.Pointer(&p)) = addr
	return unsafe.Slice((*byte)(p), n)
}
