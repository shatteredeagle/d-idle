# Lab verbs: the `--lab-verbs` gate of the private d2d fork

> status: current (TASK-42, 2026-11-06). Tags: V = read in the fork commit or run on the worker host, U = unverified.

## What it is
The fork branch `d2d.lab3-labverbs` (commit `f75816c616c8`, private remote; V `git ls-remote`) adds the option `--lab-verbs`, default OFF. Without it every `debug` sub-verb that is not on the read-only allow-list answers `err lab verbs off: debug <sub> needs d2d --lab-verbs`. The list is an allow-list in `DebugVerbs::read_only` (`apps/d2d/devctl/debug.cpp`), so a sub-verb added later is gated until someone lists it. There is no env twin (V: the code has only the CLI11 flag, the earlier proposal `D2D_LAB_VERBS=1` was not built). The CLI11 parser rejects an unknown flag: a d2d older than this branch (for example the Gate 2 binary `ee5e06d8ef99`) refuses `--lab-verbs`, so the flag and the pin move together.

## Not gated (answer without the flag)
`debug` read-only: players, level, warps, portals, presets, objects, corpses, layout, nearby, grid, blocked, attack, passives, itemat, missiles, help, tomb, unbuilt, pets, critters, net, roster, laid, light, walls, tiles, bare `states`, `quest <q> show`, `quest list`. Other verbs: ping, state, info, clock, dump, screenshot, step, lockstep, timescale, cmd, the input verbs (click, key, move), save, quit.

## Gated (need `--lab-verbs`)
Everything else under `debug`: clearinv, give, stat, statpts, skillpts, points, equip, unequip, belt, durability, wear, unid, skill, spawn, goto, warp, wp, act, enter, hit, kill, die, merc, pin, cast, mana, mon, `states set|clear`, quest writers, difficulty, boss, charges, release, day, rain, alt, collision, automap, relevel, operate, chat, typing, netsend, tradestate, party and any sub-verb not listed above (fork `docs/control_channel.md`).

## Who passes it
- Passed: `cmd/d2drill` (ExtraArgs of the drill runner and the D10 lab, `--d2d-args "--lockstep --lab-verbs"`), `cmd/d2dsoak/lab.go`, the private gate supervisor, `tools/d2dview-demo.sh` and `tools/d2dslot-test.sh` (set `D2D_LAB_VERBS=` empty for a d2d older than the branch).
- Not passed: the container tester entrypoint (`deploy/d2d-container/entrypoint.sh`): the shipped tester d2d runs without `--lab-verbs`; only the read-only debug verbs answer; setup verbs reply `err lab verbs off`.
- Not changed on purpose: `internal/d2dreplay` `SafeArgs` (a replay header is untrusted input and must not enable setup verbs). Finding: that package treats `debug points` as read-only, but it is a writer (gated by the fork).
- `tools/d2dfarm.sh record` keeps its d2d args without the flag (a recording is input only; U: it sends no setup verb).

## Proof (V, worker host, 2026-11-06)
See the TASK-42 result in `backlog/tasks/`: ctest `lab_verbs_d2d`, `coop_d2d`, `coop_lobby_d2d`, `pad_use_d2d` pass; `smoke_d2d` fails at "cmd skill didn't hurt" on this host with the new AND the old Gate 2 binary alike (pre-existing, not caused by the gate).
