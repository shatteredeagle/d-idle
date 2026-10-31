package saves

import (
	"fmt"
	"os"
	"path/filepath"

	"d2lod-bot/internal/cheat/d2s"
)

// Verify checks a character file image: signature, version 96, size field, checksum, section markers
// and the stat block (all inside d2s.Parse; layout V on Botty and Testy, see docs/saves.md). The item
// section is not parsed, so a file truncated inside it but with consistent size and checksum passes.
func Verify(b []byte) (*d2s.File, error) { return d2s.Parse(b) }

// VerifyFile reads and verifies path.
func VerifyFile(path string) (*d2s.File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := Verify(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return f, nil
}
