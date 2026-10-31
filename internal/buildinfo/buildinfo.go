// Package buildinfo carries the release version stamp shared by every binary.
// packaging/release.sh sets Version with -ldflags "-X d2lod-bot/internal/buildinfo.Version=<git describe>".
package buildinfo

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// Version is "dev" for a plain go build.
var Version = "dev"

// Current is the version to report: the ldflags stamp when set, else the VCS revision the go tool embedded
// ("<7 chars>", "-dirty" when the tree was modified), else "dev". Builds without a .git (labrun, tools/live.sh on
// a copied tree) have no VCS info, so those callers must pass the -X stamp (tools/stamp.sh).
func Current() string {
	if Version != "dev" && Version != "" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		return fromSettings(bi.Settings)
	}
	return "dev"
}

func fromSettings(ss []debug.BuildSetting) string {
	rev, dirty := "", false
	for _, s := range ss {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}

// Handle prints "<name> <version>" to w and returns true when args (os.Args[1:]) is exactly one version flag.
// Binaries call it first thing in main, before any flag parsing or setup.
func Handle(name string, args []string, w io.Writer) bool {
	if len(args) != 1 || (args[0] != "--version" && args[0] != "-version") {
		return false
	}
	fmt.Fprintln(w, name, Current())
	return true
}

// Exit is Handle on os.Args and os.Stdout that also exits 0 when it printed.
func Exit(name string) {
	if Handle(name, os.Args[1:], os.Stdout) {
		os.Exit(0)
	}
}
