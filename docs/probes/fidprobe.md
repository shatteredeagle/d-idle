# d2fidprobe run recipe (same-save probe, slice 1)

> status: UNVERIFIED until the first live run. The tool and its tests exist (offline, replay only); see [implementation status](../research/fidelity-probe-design.md). Card `fidprobe-core` (TASK-55). The first live run waits for the worker host to be free after the Gate 2 g2g wave.

## Per scenario (all read-only: no input, no memory writes, no cheats, d2d lab verbs off)

1. Capacity: `labrun orion -- top -bn2 -d1` (idle + nice at least 25 percent; `tools/capacity.sh --add 2` as the ops rules say).
2. Saves: a directory outside the repo with `<role>.d2s` per scenario (roles: `d2fidprobe -list`). Work on COPIES: the host rewrites the save on exit (d2d-lan-oracle.md V). Never commit them.
3. game.exe side: a hidden instance i1..i5 (private Xvfb), never i0, never another agent's. Only multi/OOG tooling starts it: the coordinator starts it and logs in the character copy (`tools/game.sh start <id>` + OOG login), muted (`-w -ns`, `tools/mute-d2.sh`).
4. d2d side: one slot from `tools/d2dslot.sh`, started through `tools/agentrun.sh <agent> --ttl SECS -- ...`, the copy placed as `save/<Name>.d2s`, character selected, in town. No `--lab-verbs`.
5. Wait until both are in town and at rest, then run through the ops wrapper so the ptrace grant and the instance flock apply, from a build of `cmd/d2fidprobe`:
   `tools/live.sh <agent> d2fidprobe -scenario 1 -instance iN -sock SLOT.sock -tables CACHEDIR -out OUTDIR`
   (`-tables` is the extracted game table cache that `d2dsnap -tables` takes; without it item classes are unknown and the items group is UNREADABLE.)
6. Read the table: any MISMATCH is a matrix row update and, for a d2d deviation, an oracle task; it is never added to `known.json` without a row and a reason. Stop the instance (`tools/game.sh stop <id>`), `tools/agentreap.sh <agent>`, `tools/agentreap.sh --list`.

## Offline replay (no game)

`d2fidprobe -scenario fresh-sorceress-1 -a json:A.json -b json:B.json -reads 1`, with `A.json` from a `d2.Snapshot` JSON and `B.json` from `d2dsnap -progress`.

## Checklist for the first live run (slice 1, scenarios 1 to 4)

- [ ] worker host has capacity; coordinator started and logged in the hidden instance
- [ ] saves for roles `sorc-fresh`, `pal-fresh`, `sorc-6`, `sorc-6-gear` exist as copies
- [ ] confirm U items: game.exe `FindServerPlayer` works in town; d2d `Progress` answers on the lab binary (stock d2d answers ErrNoProgress, so quests and waypoints are UNREADABLE there)
- [ ] scenarios 1 to 4 run, three-read stability passes on both sides
- [ ] reports saved to `docs/research/fidelity-probe-run1.md` (mismatches and UNREADABLE fields against matrix rows)
- [ ] record `writes after start: 0` from `d2cheat ledger --since-run` for the record
