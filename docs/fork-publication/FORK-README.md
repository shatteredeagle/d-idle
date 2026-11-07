# d2d-didle: the d.idle fork of d2d

> Draft for the fork root (phase 2 adds it as README-DIDLE.md on `didle/main`). Written 2026-11-07.

This is a fork of [psi29a/d2decomped](https://github.com/psi29a/d2decomped), the clean-room reimplementation of Diablo II 1.14d ("d2d"). It carries the patches the d.idle project (an idle-play bot for Diablo II) needs to drive and measure d2d headlessly. It is offered to upstream: the maintainer keeps upstream a clean reimplementation until 1.0, so these patches are not expected to merge as they are; fixes that stand alone are sent upstream as normal pull requests.

## What is added over upstream

* Lockstep clock: `--lockstep` and `--time-scale`, with `step`, `clock`, `lockstep` and `timescale` control verbs, so a run is deterministic and faster than real time.
* Dump verbs: read-only JSON of the player, stats, skills, items, quests, UI rectangles, mercenary and time.
* Lab and debug verbs (spawn, give, quest set, mana, states, cast, missile listing and more) for oracle tests against the original game. They are OFF by default and only exist behind `--lab-verbs`.
* `--overlay-dir`: a table row overlay on top of the game data (default off).
* Freeze/resume and Frozen Armor oracle tests.

## What is NOT in it

No Blizzard game data and no Blizzard code: no MPQ, DC6, D2S or table files. You bring your own copy of the Diablo II and Lord of Destruction 1.14d data, exactly as for upstream (`--data /path/to/your/MPQ-folder`). Test cheats stay behind `--lab-verbs` and never go into an upstream pull request without the owner's approval.

## Build

As upstream (C++26 compiler, CMake 3.28+, Ninja, SDL3, OpenAL, zlib, libcurl; see the upstream README section "Build and run"):

```
cmake -S . -B build -G Ninja
cmake --build build
build/apps/d2d/d2d --data /path/to/your/MPQ-folder
```

## Branches and tags

* `didle/main`: the working fork.
* `didle/lab3-labverbs` (tag `didle-gate2-lab3`) and `didle/lab2-freezeresume` (tag `didle-gate2-lab2`): the commits the measured Gate 1 and Gate 2 runs were built from, kept for provenance.

## Licence

GPL-3.0-or-later, as upstream. The full text is in [LICENSE](LICENSE). Every added source file carries the SPDX line. Not affiliated with or endorsed by Blizzard Entertainment; Diablo is a Blizzard trademark.
