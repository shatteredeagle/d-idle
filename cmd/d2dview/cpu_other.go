//go:build !(linux || darwin)

package main

import "time"

func procCPU() time.Duration { return 0 }
