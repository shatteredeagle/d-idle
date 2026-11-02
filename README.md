<p align="center"><img src="assets/branding/final/header-1280x640.png" alt="d.idle" width="640"></p>

# d2lod-bot (d.idle)

Chat and help: the [d.idle Discord](https://discord.gg/U2v9q4h5UH).

A bot that plays one character of classic **Diablo II: Lord of Destruction 1.14d**, offline single player, on a game you own
and run yourself. Written in Go (standard library only). It reads the game's memory, decides what to do (level, quest, farm,
shop, survive), and sends ordinary mouse and keyboard input. The idle-game layer on top of it (upgrades, prestige, a public
status page and stream) is called **d.idle**; it is parked until the bot plays well.

## Status

Early, honest, and measured rather than promised. The bot runs on d2d (the primary backend) in a lockstep lab. A Sorceress
levels through Normal Act 1 unattended: the last 2 h unattended soak reached level 13 with no deaths (one injected d2d kill
recovered; per-hour numbers and caveats are in `docs/probes/postfix-soak1.md`, a single run, not a statistical claim). The
basic game primitives (belt, pickup, town portals, waypoints, shopping, repair, stash, merc revive, equip, save and rejoin)
are checked by a live drill suite (`cmd/d2drill`, `docs/primitives.md`, `docs/probes/drills/`) on hidden game instances; some
drills still fail or are blocked, and the drill reports say which.

Still unproven: monster fights beyond early Act 1, quests, Acts 2 to 5, Nightmare and Hell (code and offline simulations only),
and classes other than the Sorceress (short smoke runs). The fidelity gap between d2d and the original game is measured, not
claimed closed (`docs/d2d-baselines-early.md`). The current state is always in `docs/plan/STATUS.md` (generated), the
day-by-day notes in `CHANGELOG.md`, and the entry point for everything else is `docs/INDEX.md`.

## What it can do today (honestly)

- Plays a Sorceress through early Normal Act 1 (level 1 to about 13) unattended on d2d; it has not played Andariel, Act 2 or later.
- The second two-hour d2d soak did NOT pass Gate 1 (one seed); fixes merged since have not been re-soaked.
- Basic actions (potions, portals, waypoints, shopping, repair, stash, save and rejoin) pass most live drills; some fail or are blocked.
- Teleport hops in navigation are proven live on the lab build (DRILLS24, 25, 27: for example Stony Tomb 211 ticks against 254 walking); the Frozen Armor recast retry after a cut cast is proven live in one scenario (DRILLS29).
- Frost Nova and Chain Lightning numbers measured on d2d are not trustworthy yet (the hit-once fix is merged upstream, not re-soaked). Melee, resistance and Paladin handling are offline-verified only; other classes only had short smoke runs.
- Six of our d2d fixes are merged upstream; two more (#159, #160) are open. The freeze fixes live in the private fork until then.
- Offline single player only: no Battle.net, no multiplayer, no game files in this repo.

Details, with a link to the evidence for every row: [docs/capabilities.md](docs/capabilities.md).

## Running it

```sh
go build -o bin/ ./cmd/d2bot ./cmd/d2dsoak      # standard library only
bin/d2bot -h                                    # every flag, with its environment twin
gunzip -c docs/probes/soak-logs/postfix2-minutes.jsonl.gz > bin/postfix2.log && bin/d2dsoak compare -hours 21.25 -baselines tools/d2dsoak/baselines-early.json bin/postfix2.log
```

The last line replays a recorded soak into a verdict (no game needed). Playing against a game needs your own Diablo II
install and a d2d binary; steps, flags and troubleshooting: [docs/running.md](docs/running.md).

## How it runs: two backends

- **d2d (primary).** [d2decomped](https://github.com/psi29a/d2decomped) is an open-source, native reimplementation of the
  1.14d engine. The bot talks to it over its local control socket as a separate program; no Wine. This is the supported path
  and the one the container uses (`deploy/container/`, `docs/d2d-container/`; ADR 0009 in `docs/adr/`).
- **Original Game.exe under Wine (reference).** The path the bot grew up on. It is frozen as a reference: no new features, and
  the old 2 h and 8 h Game.exe gate soaks are retired in favour of the same gates on d2d (ADR 0013, spec
  `openspec/specs/soak-gates/spec.md`, "d2d gate ladder" in `docs/review-strategy.md`).

This repo contains no game files: no Blizzard binaries, MPQ archives, saves or data tables; you need your own legitimate install of Diablo II: Lord of Destruction 1.14d.

In both cases **you supply your own Diablo II install**. Nothing Blizzard owns is in this repository, downloaded, or mirrored,
and the bot never touches Battle.net or chat. Our fixes to d2d are offered to its maintainer upstream as small, proven pull
requests; this repo contains no d2d code.

## How we check it

- **Drills (Gate 0):** scripted live checks of each basic primitive on a hidden d2d slot with a throwaway character.
- **Soak ladder:** 2 h, then 8 h, then 24 h unattended runs on one build, judged by the gates in the soak-gates spec.
- **Differential proofs:** any change to d2d is checked against the original game's own code running under an emulator
  (`tools/emu`, `docs/research/`), not against our assumptions.

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
| `cmd/` | binaries: `d2bot` (the bot and its dashboard), `d2fleet` (many bots), `d2soak` and `d2dsoak` (long-run verdicts), `d2drill` (live drill suite), `d2dview` (game view), tools |
| `internal/` | packages: memory reading, input, navigation, combat, town, leveling, gear, merc, API; `internal/drill` holds the drills |
| `web/` | dashboard front end (embedded) |
| `deploy/`, `packaging/` | container files and a NAS app template |
| `openspec/` | contracts (`specs/`) and the change cards every piece of work is briefed and tracked with |
| `docs/` | design, evidence and research notes; `docs/INDEX.md` is the map; `docs/research/` and `docs/probes/` hold traces, drill and soak reports |
| `tools/` | developer and maintainer scripts (some assume the maintainer's lab and are documented as such); `tools/emu` is the emulator harness |

## Contributing: agents first

This repo is developed mostly by AI coding agents working from written briefs. Every job is a change card in
`openspec/changes/<id>/` with the paths it owns and the gates it must pass; `AGENTS.md` holds the rules those agents (and
humans) follow, and `GLOSSARY.md` the vocabulary. If you want to contribute:

- open an issue or a pull request against `main`; describe what you verified and how (live run, unit test, or neither);
- tag every game fact you rely on as verified or unverified, like the rest of the docs do;
- do not add game files, extracted tables beyond what `THIRD_PARTY.md` allows, decompiled code, or anything that talks to Battle.net;
- keep to the Go standard library;
- the project name is styled **d.idle**, lowercase d.

Chat and help: the [d.idle Discord](https://discord.gg/U2v9q4h5UH). Rules and FAQ are pinned there.

Want to help with decomps and verifications (with your own copy of the game and your own agent)? See [docs/contributing/agent-work.md](docs/contributing/agent-work.md).

Who has contributed what: [CONTRIBUTORS.md](CONTRIBUTORS.md).

This repository is the working repository: pull requests are merged here directly. Its history starts at an initial snapshot; the working history before it is kept privately.
Read the rules in `AGENTS.md` (agents first, upstream style: small changes, tagged facts, no game data) before you open a pull request.
How CI works, in one line: every push to `main` and every pull request runs `.github/workflows/ci.yml` (changecheck, leakcheck, hygiene, gofmt/vet, then the short tests); a `v*` tag also builds and publishes the release tarball.
Maintainer-only operations material (lab hosts, status board, publishing) lives elsewhere and is not needed to build, test or contribute.

## Licence

GPL-3.0-or-later, same as d2d (ADR 0010); see `LICENSE`. Third-party code and provenance: `THIRD_PARTY.md`.
Diablo II is a trademark of Blizzard Entertainment; this project is not affiliated with or endorsed by Blizzard.
