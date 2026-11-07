//go:build !unix

package currency

import (
	"errors"
	"os"
	"time"
)

// lockFile is a create-exclusive lock file with a bounded wait (no flock off unix).
func lockFile(path string) (func(), error) {
	for i := 0; i < 200; i++ {
		f, err := os.OpenFile(path+".excl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(path + ".excl") }, nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return nil, errors.New("currency: ledger lock busy")
}
