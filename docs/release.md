# Release packaging

`packaging/release.sh [--dry-run] [VERSION]` builds a release tarball on this machine. It never uploads, tags or pushes.
VERSION defaults to `git describe --tags --always --dirty` ("dev" in a copy without `.git`). Output dir: `dist/` (`DIST=` overrides).

## Layout (linux-amd64) [V: checked by packaging/release_test.sh]
```
d2lod-bot-<ver>-linux-amd64/
  bin/        d2bot d2fleet d2doctor d2soak probes d2sentinel d2cheat d2multi d2snap d2view d2watch d2dash oog
  helper/     d2input.exe          (windows/386, the path internal/setup and internal/launch expect under the data dir)
  VERSION  RELEASE_NOTES.md  CHANGELOG.md  THIRD_PARTY.md  SHA256SUMS
d2lod-bot-<ver>-linux-amd64.tar.gz  and  .tar.gz.sha256
```
Live-test rigs (`act2live`, `navlive`, `townlive`, ...) are not shipped. Web assets (d2watch, d2dash, d2view, setup wizard) are
`go:embed`ded, so there is no loose `web/` directory. The tarball is reproducible apart from the Go build id (sorted names, owner 0,
mtime = last commit time).

## Version stamp
`-ldflags "-X main.version=<ver> -X d2lod-bot/internal/buildinfo.Version=<ver>"` for every binary. `d2bot --version` keeps its own
`main.version`; all others call `buildinfo.Exit(name)` first in `main` and print `<name> <ver>` for a lone `--version`/`-version`.
d2input.exe is stamped too (`grep -a <ver> d2input.exe`); it is a GUI-subsystem PE, so run `--version` on Linux builds only.

## Release notes
`RELEASE_NOTES.md` = header plus the first two `## ` sections of `CHANGELOG.md` (Unreleased / known issues, newest dated
section). Keep that order in the changelog. The full CHANGELOG ships too.

## Windows rule
The "Windows apps ship installer + portable" rule applies to apps a user runs on Windows. Here only `d2input.exe` is Windows
code; it is started by the Linux launcher inside the Wine prefix from `helper/`, never by a person, so no installer (and no
CI-built portable pair) is needed. If a Windows-run app is ever added, it must ship both, CI-built.

## Dry run and test
`packaging/release.sh --dry-run` builds into a temp dir, runs `packaging/release_test.sh TARBALL VERSION` (layout, all 12
`--version` outputs, PE header, SHA256SUMS in both directions, notes sections, no stray .exe or save/key files) and deletes it.
Run it on the worker host: `labrun worker -- bash packaging/release.sh --dry-run` (about 30 s). [V: passed there and on the head box, 2026-10-29.]
Not verified: running the extracted binaries on a clean machine; publishing (CI/the forge release) is out of scope here.
