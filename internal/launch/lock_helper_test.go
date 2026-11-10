package launch

import "os"

func flockEx(f *os.File) error { _, err := tryFlock(f); return err }
