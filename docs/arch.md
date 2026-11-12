# Architecture diagrams (Archify)

> status: current (TASK-373). One command draws the project as clickable HTML; nothing generated is committed.

## Use

```
tools/arch/arch.sh                        # d2lod-bot-overview.html + d2lod-bot-packages.html
tools/arch/arch.sh --diff v0.9.0 main     # drift-<old>-<new>.html: what moved between two refs
tools/arch/arch.sh --repo ~/d2d-lab/fork --name fork   # another Go checkout (see "Other repos")
```

Output goes to `docs/arch/out/` (gitignored; override with `ARCH_OUT`). Each file is one self-contained HTML (about 0.9 MB), open it in a browser; it has pan, zoom, light/dark theme and SVG/PNG export. Runtime: about 13 s on the 4 GB box (V: measured; `go list` takes 0.3 s, the first call also fetches Archify once).

## What it shows

- **Overview**: 8 groups (Commands, Player screen, Ops and lab, Brain, Behaviours, d2d backend, Game state, Foundation) with the package-import flow between them; line thickness is the number of package imports. Imports from Commands, imports that skip a tier and imports of Foundation are left out (the subtitle counts them) because they would cross other boxes.
- **Every package**: all 94 nodes (`cmd/*` is collapsed into three boxes: player binaries, d2d lab tools, probes and drills) inside their group boundary. No edges: 280 cross-package edges are a hairball.
- **Drift**: Archify `compare` of the overview of two refs, built from `git worktree` checkouts of both (removed again afterwards).

The grouping is data: `SEG` and `GROUPS` at the top of `tools/arch/gen.py`. A package that matches no entry lands in "Ops and lab"; add it to `SEG` to move it.

## How it works

`tools/arch/gen.py` (python3 stdlib) runs `go list -json ./...` and writes an Archify architecture JSON-IR with grid placement; `archify render` validates and writes the HTML. No model is involved: Archify is normally an agent skill where the LLM writes the JSON from reading code, here the Go toolchain does, so results are deterministic and the same on every run.

## Version, licence, trust

- Pinned commit `7e45f36c66de1254d5ad6ba1d878f6499ea269fd` (Archify 3.2.0, upstream main of 2026-11-10), fetched sparse (`archify/` only) into `~/.cache/archify/<sha>` by `arch.sh`; a mismatch with the pin aborts. To upgrade, change `ARCHIFY_SHA` after reading the diff.
- Licence: Apache-2.0 (`LICENSE`), with MIT parts (`LICENSE-MIT`, "Cocoon AI" 2025) and third-party brand-mark notices. Use as an external tool is fine; we do not copy, vendor or link its code, and `go.mod` is untouched (V: stdlib-only rule kept).
- Network (V: grepped `archify/bin`, `scripts`, `renderers`): the only outbound call is the update-check fetch of a version manifest on its own GitHub Pages; it sends no source. `arch.sh` sets `ARCHIFY_UPDATE_CHECK_DISABLED=1`. `render` needs no API key and no LLM. Its `browser-check`/`visual-check` gates need Chrome and are not used (not installed here); the `skills` npm helper that installs it into the global agent config (`npx skills add ... -g`) is NOT used.
- Generated HTML embeds Archify's brand-mark attribution text (public URLs only). It holds package names and import counts, no hosts or paths (V: grep for `/home/` and the private domain: 0 hits). Still kept out of git; `go run ./cmd/leakcheck -mode fail` was run on the branch.

## What it cannot see

- Runtime wiring: interfaces assigned in `env`/`d2denv`, the control socket protocol, goroutines and channels. An import edge says "can call", not "does call".
- Data-driven behaviour: builds, plans and knobs live in data (`plandata`, `combat/classdata`, docs/plan/data); the diagram shows the package, not which build runs.
- Test-only imports are included by `go list` Imports only for non-test files (V: `Imports` excludes `_test`), so test fakes show as Ops nodes with no incoming edges.
- Python and shell tooling under `tools/`, the web UI (`web/`), the Electron app (`apps/`) and the fork C code are not in the Go graph.

## Other repos

`--repo DIR` works for any Go module (groups that do not match fall into "Ops and lab"; tune `SEG`). The d2d fork is C and the ops repo is shell and Python: there is no Go import graph to draw, so no diagram is made for them (U: a C include graph would need a different generator).
