//go:build !windows

package main

import "os"

// Stub so the protocol code builds and tests on the host; the real helper is Windows-only.
type noSys struct{}

func newSys() sys { return noSys{} }

func (noSys) list() []winInfo                             { return nil }
func (noSys) valid(uintptr) bool                          { return false }
func (noSys) post(uintptr, uint32, uintptr, uintptr) bool { return false }
func (noSys) scan(uint32) uintptr                         { return 0 }
func (noSys) info(uintptr) string                         { return "" }

func (noSys) hung(uintptr) bool { return false }

func procHandles() int { return 0 }
func logDir() string   { return os.TempDir() }
