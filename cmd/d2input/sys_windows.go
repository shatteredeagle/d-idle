//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	pFindWindowW    = user32.NewProc("FindWindowW")
	pPostMessageW   = user32.NewProc("PostMessageW")
	pGetClientRect  = user32.NewProc("GetClientRect")
	pMapVirtualKeyW = user32.NewProc("MapVirtualKeyW")
	pEnumWindows    = user32.NewProc("EnumWindows")
	pGetWindowTextW = user32.NewProc("GetWindowTextW")

	pGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	pIsWindowVis              = user32.NewProc("IsWindowVisible")
	pIsHung                   = user32.NewProc("IsHungAppWindow")
)

// winSys funnels every Win32 call through one locked OS thread (thread.go).
type winSys struct{ th *osThread }

func newSys() sys { return winSys{newOSThread()} }

// enumCB is created ONCE. syscall.NewCallback allocates a slot in a fixed table of 2000 entries that is never
// freed; creating one per list() call made the helper die with "fatal error: too many callback functions"
// (an uncatchable runtime throw) after ~2000 commands, i.e. a few minutes of bot traffic (docs/input.md).
var (
	enumOut []winInfo // only touched on the Win32 thread
	enumCB  = syscall.NewCallback(enumProc)
)

func enumProc(h uintptr, _ uintptr) uintptr {
	var buf [32]uint16
	n, _, _ := pGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if syscall.UTF16ToString(buf[:n]) != "Diablo II" {
		return 1
	}
	var pid uint32
	pGetWindowThreadProcessID.Call(h, uintptr(unsafe.Pointer(&pid)))
	var r [4]int32
	pGetClientRect.Call(h, uintptr(unsafe.Pointer(&r[0])))
	vis, _, _ := pIsWindowVis.Call(h)
	enumOut = append(enumOut, winInfo{h: h, pid: pid, w: int(r[2] - r[0]), ht: int(r[3] - r[1]), visible: vis != 0})
	return 1
}

// list enumerates the windows on every call, so a restarted game is picked up with no helper restart.
func (s winSys) list() (out []winInfo) {
	s.th.do(func() {
		enumOut = nil
		pEnumWindows.Call(enumCB, 0)
		out = enumOut
	})
	return out
}

func (s winSys) valid(h uintptr) bool {
	for _, w := range s.list() {
		if w.h == h {
			return true
		}
	}
	return false
}

func (s winSys) post(h uintptr, msg uint32, w, l uintptr) (ok bool) {
	s.th.do(func() {
		r, _, _ := pPostMessageW.Call(h, uintptr(msg), w, l)
		ok = r != 0
	})
	return ok
}

func (s winSys) scan(vk uint32) (sc uintptr) {
	s.th.do(func() { sc, _, _ = pMapVirtualKeyW.Call(uintptr(vk), 0) })
	return sc
}

func (s winSys) info(h uintptr) (out string) {
	s.th.do(func() {
		var r [4]int32
		pGetClientRect.Call(h, uintptr(unsafe.Pointer(&r[0])))
		vis, _, _ := pIsWindowVis.Call(h)
		out = fmt.Sprintf("hwnd=0x%x client=%dx%d visible=%d", h, r[2]-r[0], r[3]-r[1], vis)
	})
	return out
}

func (s winSys) hung(h uintptr) (bad bool) {
	s.th.do(func() {
		r, _, _ := pIsHung.Call(h)
		bad = r != 0
	})
	return bad
}

var pGetProcessHandleCount = syscall.NewLazyDLL("kernel32.dll").NewProc("GetProcessHandleCount")

func procHandles() int {
	var n uint32
	h, _ := syscall.GetCurrentProcess()
	pGetProcessHandleCount.Call(uintptr(h), uintptr(unsafe.Pointer(&n)))
	return int(n)
}

// logDir is the Wine C: drive root (<prefix>/pfx/drive_c), visible from the host.
func logDir() string { return `C:\` }
