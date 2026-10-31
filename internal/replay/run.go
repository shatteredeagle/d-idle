package replay

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

// CheckBuild rejects fixtures recorded from a different build. Legacy (schema 0) fixtures carry no
// build and pass.
func CheckBuild(h mem.Header) error {
	if h.Schema == 0 {
		return nil
	}
	if h.Build != mem.ExpectedBuild() {
		return fmt.Errorf("replay: fixture build %+v is not the supported build %+v; re-record it", h.Build, mem.ExpectedBuild())
	}
	return nil
}

// Run is the golden test helper. It loads the memory fixture (any format that has pages), calls fn
// with a Source (first frame applied) and a simulating Input, and compares the input calls fn made
// with file+".golden" (one call per line). Run with REPLAY_UPDATE=1 to write the golden file.
// A fixture with no golden file passes only if fn made no calls.
func Run(t *testing.T, file string, fn func(src *Source, in *Input)) {
	t.Helper()
	fx, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if fx.Stream == nil {
		t.Fatalf("%s: kind %q has no memory stream", file, fx.Header.Kind)
	}
	if err := CheckBuild(fx.Header); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	in := NewInput(input.Point{})
	fn(NewSource(fx.Stream), in)
	got := FormatCalls(in.Calls())
	golden := file + ".golden"
	if os.Getenv("REPLAY_UPDATE") != "" {
		if got == "" {
			os.Remove(golden)
			return
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if errors.Is(err, os.ErrNotExist) {
		if got != "" {
			t.Fatalf("%s: no golden file; input calls were:\n%srun with REPLAY_UPDATE=1 to create it", file, got)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("%s: input calls differ from golden\nwant:\n%sgot:\n%s", file, want, got)
	}
}
