//go:build !unix

package savesafe

import "errors"

// FreeBytes is not probed off unix; callers treat an error as "not checked".
func FreeBytes(string) (uint64, error) { return 0, errors.New("savesafe: disk probe is unix only") }
