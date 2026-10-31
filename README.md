# d2lod-bot (D.idle)

A bot that plays one character of classic **Diablo II: Lord of Destruction 1.14d**, offline single player, on a game you own
and run yourself. Written in Go (standard library only). It reads the game's memory, decides what to do (level, quest, farm,
shop, survive), and sends ordinary mouse and keyboard input. The idle-game layer on top of it (upgrades, prestige, a public
status page and stream) is called **D.idle**; it is parked until the bot plays well.

## Status

Early, honest, and measured rather than promised. Today a Sorceress levels through Normal Act 1 unattended for a couple of
hours (deaths and experience per hour are the gates, see `openspec/specs/soak-gates/spec.md`); other classes ran only short
smoke runs; Acts 2 to 5 and Nightmare/Hell exist partly or only as offline code. The current state is always in
`docs/plan/STATUS.md`, the day-by-day notes in `CHANGELOG.md`, and the entry point for everything else is `docs/INDEX.md`.

## How it runs: two backends

- **d2d (primary).** [d2decomped](https://github.com/psi29a/d2decomped) is an open-source, native reimplementation of the
  1.14d engine. The bot talks to it over its local control socket as a separate program; no Wine. This is the supported path
  and the one the container uses (`deploy/container/`, `docs/d2d-container/`).
- **Original Game.exe under Wine (reference).** The proven path the bot grew up on; frozen, kept as the acceptance reference
  until the d2d container passes its long soak (ADR 0009, `docs/adr/`).

In both cases **you supply your own Diablo II install**. Nothing Blizzard owns is in this repository, downloaded, or mirrored,
and the bot never touches Battle.net or chat. Some of our fixes to d2d live in a downstream fork and are offered upstream when
the maintainer wants them; this repo contains no d2d code.

## Quick start (safe, offline)

```sh
go version            # Go 1.27 or newer (go.mod's `go 1.24` is only the language floor)
go build ./...        # standard library only, no network
go test ./internal/... # unit tests run against recorded snapshots, no game needed
```

Running the bot against a real game: `docs/quickstart.md` (user guide) and `docs/container.md` (the container). Expect rough
edges; the guides say what has been verified on a stranger's machine and what has not.

## Layout

| path | what |
|---|---|
| `cmd/` | binaries: `d2bot` (the bot and its dashboard), `d2fleet` (many bots), `d2soak` (long-run verdicts), tools |
| `internal/` | packages: memory reading, input, navigation, combat, town, leveling, gear, merc, API |
| `web/` | dashboard front end (embedded) |
| `deploy/`, `packaging/` | container files and a NAS app template |
| `openspec/` | contracts (`specs/`) and the change cards every piece of work is briefed and tracked with |
| `docs/` | design, evidence and research notes; `docs/INDEX.md` is the map |
| `tools/` | developer and maintainer scripts (some assume the maintainer's lab and are documented as such) |

## Contributing: agents first

This repo is developed mostly by AI coding agents working from written briefs. Every job is a change card in
`openspec/changes/<id>/` with the paths it owns and the gates it must pass; `AGENTS.md` holds the rules those agents (and
humans) follow, and `GLOSSARY.md` the vocabulary. If you want to contribute:

- open an issue or a pull request against `main`; describe what you verified and how (live run, unit test, or neither);
- tag every game fact you rely on as verified or unverified, like the rest of the docs do;
- do not add game files, extracted tables beyond what `THIRD_PARTY.md` allows, decompiled code, or anything that talks to Battle.net;
- keep to the Go standard library.

This repository is the working repository: pull requests are merged here directly. Its history starts at an initial snapshot; the working history before it is kept privately.
Read the rules in `AGENTS.md` (agents first, upstream style: small changes, tagged facts, no game data) before you open a pull request.
How CI works, in one line: every push to `main` and every pull request runs `.github/workflows/ci.yml` (changecheck, leakcheck, hygiene, gofmt/vet, then the short tests); a `v*` tag also builds and publishes the release tarball.
Maintainer-only operations material (lab hosts, status board, publishing) lives elsewhere and is not needed to build, test or contribute.

## Licence

GPL-3.0-or-later, same as d2d (ADR 0010); see `LICENSE`. Third-party code and provenance: `THIRD_PARTY.md`.
Diablo II is a trademark of Blizzard Entertainment; this project is not affiliated with or endorsed by Blizzard.
