//go:build !linux

package launch

// BecomeSubreaper is Linux-only (prctl); elsewhere it reports ErrUnsupportedOS.
func BecomeSubreaper() error { return ErrUnsupportedOS }

// IsSubreaper reports false with ErrUnsupportedOS outside Linux.
func IsSubreaper() (bool, error) { return false, ErrUnsupportedOS }

// CloseOnExecInherited is a no-op outside Linux (there is no inherited flock fd to protect).
func CloseOnExecInherited() {}
