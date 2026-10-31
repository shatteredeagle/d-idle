# Replay, fixtures and canaries: internal/replay, internal/canary, cmd/d2rec

## Envelope (internal/mem/envelope.go) - design-gaps D10
`"D2FX" | u16 schema | u32 headerLen | header JSON | payload`, schema 1. Header: `schema, kind (pages|mapimage|stream|trace), build{timestamp,size}, runner, recorder, instance, char, class, difficulty, seed, area, createdAt, notes, verified[]`.
Payload is the old file verbatim for pages (D2SN) and mapimage (d2mi); gzip JSONL for stream and trace.
Legacy files (no magic) load as schema 0 with the kind inferred from content (D2SN magic, gzip = d2mi, `{` = trace), so every existing fixture loads unchanged. `mem.ReadSnapshot` unwraps pages envelopes. A newer schema is rejected.

## Stream (internal/replay)
JSONL records `{"t":ms,"pages":{"hexaddr":"base64 4 KiB"}}` (changed pages; the first record holds every touched page) and `{"t":ms,"in":"click L 10 20"}`.
- `replay.Recorder` wraps the readers' Source; `Tick()` re-reads touched pages and stores deltas; `WrapInput(ctrl)` records input calls; `Run(ctx, every, work)`.
- `replay.Source` = mem.Source at a cursor (`Step`, `Seek(t)`, `SeekFrame(n)`). `replay.Input` = input.Controller that records canonical call text (simulates like input.Fake when `Inner` is nil).
- `replay.Load(path)` decodes any fixture. `replay.Run(t, file, fn)`: build check, run fn, compare Input calls with `file.golden`; `REPLAY_UPDATE=1` writes it.

## Canary (internal/canary)
`canary.Check(src) []Result` (pass/fail/skip per rule), `canary.Gate(src) error` (start gate for WP-LIFECYCLE). Rules: build (VerifyBuild; skip if the PE page is absent), client-server table delta, menu-or-game, server-table-chain, player-type, player-area 1..136, Inventory.pOwner, StatList owner.
CORRECTION (V on town.snap and town_inv.snap, the worker host 2026-10-28): the player StatList has `pUnit` at +0x04 = 0 on 1.14d, not the owner. The player pointer is at +0x44; +0x08 = 0 (type), +0x0C = unit id, +0x10 = 0x80000000. Section 6 of d2-1.14d-memory.md ("StatList.pUnit") should read "StatList+0x44".
Committed fixtures have no PE header page, so `build` skips on them (not a pass). d2mi images skip the unit rules (not recorded); area passes.

## cmd/d2rec
`d2rec check FILE...`, `d2rec wrap IN.snap OUT` (legacy to envelope), `d2rec record --for 30s --every 100ms --out FILE --char NAME --note TEXT --runner VER`: opens the instance's game, runs the canary gate, records `d2.Snapshot` reads (so only pages the readers touch). Live command (main agent): `tools/live.sh <agent> d2rec record --for 30s --out runs/town.d2fx`, then copy the file back; never commit real saves. UNVERIFIED live. Input recording is a library feature (`Recorder.WrapInput`), not wired into the CLI yet.
