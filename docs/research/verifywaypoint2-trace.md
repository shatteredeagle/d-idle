# Verify the waypoint refusal claim, second pass (agent verifywaypoint2, 2026-11-02)

> status: research, current. Board item #8 VERIFY-WAYPOINT. Re-checks docs/research/waypoint-trace.md and the first verification docs/research/verify-waypoint.md. Tags: **V run** = game.exe 1.14d's own code under the emulator, or d2d's own pre-fix test binary, run on the worker host; **V read** = read in the disassembly (own capstone sweeps, `tools/emu/vw2_*.py`); **S** = secondary source; **I** = inference; **U** = unverified. No game files and no disassembly are pasted. Offline: no game, no d2d slot, nothing pushed or posted.

What is new against the first verification: (1) d2d is **run**, not read: the same cases go through d2d's real waypoint code (the pre-fix `test_game --waypoint-refuse` binary of the earlier waypoint card) and through game.exe; (2) my own case set with the clock free over the full 32-bit range (the first one stayed in 70,000..2^31) and the field read back after every case; (3) my own cross-reference scan (pointer tables, `lea`/`add` forms, call sites) and the way the hostile path is reached.

## Verdicts

| claim | verdict |
|---|---|
| C1 pcdata+0x160 has exactly one writer, FUN_0055b720 | **HOLDS** (V read, wider scan than before; limits below) |
| C2 game.exe refuses a 0x49 only while field + 10000 > GetTickCount, the field being the tick of the last accepted hostile declaration | **HOLDS**, with one new quirk (D3: unsigned wrapped sum) |
| C3 never in single player | **HOLDS WITH LIMITS**: unreachable by the game's own UI (I); reachable by a crafted C->S message (I, not run) |
| C4 d2d counts the window from every arrival | **HOLDS** (V run, now against d2d's binary: 100,000 of 100,000 cases equal my arrival rule) |
| C5 59,309 of 132,964 differ | **HOLDS** (V run, both programs: 47,157 + 12,152 = 59,309) |
| FAILS | nothing in the claim failed. One earlier side-claim needs a word: "the only reader is the waypoint refusal" is false (3 readers, see Q2), the first verifier already said so |

## Questions

**Q1 Who writes pcdata+0x160?** (V read) `vw2_xref.py` sweeps the whole .text (capstone, linear) for every operand with displacement 0x160 **including `lea`, `add`, `sub`, `push imm`**: 55 hits (the first verifier's sweep was `mov`-style stores and loads only). The only store of a tick into the player-data block (the pointer from FUN_006221a0) is at 0x55b76b inside FUN_0055b720. Other `[reg+0x160]` stores: 0x421e77 (a constructor zeroing a client struct), 0x4b83d5 and 0x4b83fb (a record table); the `add ecx, 0x160` at 0x5a0f47 is an address calculation for a different struct, read through `[ecx+0x1c]`; the `push 0x160` and stack `mov` hits are size constants. (U: a `rep movs` of a whole block, or a write through a base already offset, is not excluded by any sweep, see Limits).

**Q2 Who reads it?** (V read) Three call sites of FUN_0055b6c0: 0x54c5fc (the 0x49 waypoint handler FUN_0054c5d0, window 10000), 0x5848bc (the town-portal operate FUN_00584870, window 5000 and the same voice event 0x13; it is in an object function table at 0x732d54, so it is a different feature), 0x5a5f13 (the hostile declaration itself, passing a second unit: then the reader returns a per-pair timer, not the pcdata field). So the field is a "time of my last hostile declaration" shared by the waypoint and the portal refusals.

**Q3 What does FUN_0055b720 store?** (V run) After each of 100,000 cases the field read back from the player-data block equals the tick of the last accepted declaration (or 0 if none): 0 of 100,000 cases differ. A level change runs no code in game.exe, so the field is untouched by arrivals (V read for the absence of writers; the oracle replays only the declarations).

**Q4 The comparison (V run).** The harness runs game.exe's dispatcher FUN_0054d750 -> FUN_0054c5d0 -> FUN_0055b6c0 natively (same hook set as the earlier oracle; limit 2). Across 100,000 cases of my own generator with the tick free in 0..2^32 (seed 23) the rule

    refused iff target inactive, or ((field + 10000) mod 2^32) > now   (unsigned)

matches the game in **0 of 100,000** cases. The "elapsed < 10000" form matches in 90,780: it differs exactly where the tick counter has wrapped between the declaration and the 0x49 (D3). The signed form of the sum differs in 20,946.

**Q5 d2d, run.** `test_game --waypoint-refuse` (pre-fix, upstream main of 2026-10-31) on my 100,000 cases (the hostile declarations are not run, d2d has none): the arrival rule `refused iff inactive, or (now - last arrival) mod 2^32 < 10000` matches it in **0 of 100,000** cases. d2d vs game.exe: **43,558 differ** (d2d refuses, game travels 23,970; game refuses, d2d travels 19,588).

**Q6 The claimed numbers (V run, both programs).** The trace's own case set (wp_refuse.gen, seed 7: 32,964 grid + 100,000 random = 132,964) through game.exe and through the d2d binary: game refused 57,297, d2d refused 92,302; d2d refuses and game travels **47,157**, game refuses and d2d travels **12,152**, differ **59,309**. Equal to the claim.

**Q7 Is the hostile path reachable in single player?** (V read + I) FUN_005a5e50 (the declaration) sits in a 4-entry table at 0x6e221c..0x6e2228 (slot 4 of the table based at 0x6e2218), indexed only by FUN_005a6000 (the sole indexer, 2 references). FUN_005a6000(type, id, flag) looks the other player up by id (FUN_00552f60, class 0) and calls the table entry only if it exists. It has exactly two callers, both C->S message handlers in the table at 0x6e0f60.. next to the 0x49 handler: 0x54c2a0 (length 7, flag from the message) and 0x54c300 (length 6, flag 0). The first handler's table slot is 20 records after the 0x49 handler's (8-byte records, I), i.e. message 0x5d, the second 0x5e; this matches the public protocol lists (S). Inside FUN_005a5e50: both parties must be player units (FUN_00620bb0, FUN_0061ab00) with the level stat (0xc) of at least 9 (FUN_006253b0), the flag must change the current relation, and a 60 s per-pair timer applies. For the field to change in single player the message must name a player unit that exists with level 9 or more; the game's own UI offers no hostile control with no second player (I). Self-targeting by a crafted message is not excluded by what I read (U).

## Deviations d2d vs game.exe

| id | game.exe | d2d | tag | differential |
|---|---|---|---|---|
| D1 | refuses only while field + 10000 > tick, the field written by a hostile declaration; never an arrival | refuses 10 s after every level change | V run (both programs, 232,964 cases over two sets) | yes: `vw2_trace.py` (before 59,309 of 132,964); after-fix run not repeated here (upstream merged the fix as #153, the earlier card's after-count was 0) |
| D2 | hostile declarations set the field and refuse | none (no declaration) | V run (12,152 + game-only cases in my set) | yes, same harness; d2d has no code to run, so any "after" requires new code |
| D3 | the compare is the unsigned wrapped sum: a declaration made before the 32-bit tick counter wraps (about 49.7 days of uptime) refuses every 0x49 after the wrap until the counter reaches the old tick again | the arrival rule is wrap-safe (`now - arrived` unsigned), so no equivalent | V run (game.exe, 9,220 of 100,000 cases leave the elapsed form; rule B 0 differ) | yes, same harness, but practically unreachable (uptime, hostile) |
| D4 | the town-portal operate FUN_00584870 refuses for 5 s (0x1388) after the same field | not looked at | V read only | U: build a probe as for 0x49 with FUN_00584870 (needs the object and range hooks); not run |

## Bot findings (described, not changed)

- **B1** internal/env/tuning_nav.go:34: the field comment still starts "game.exe refuses a waypoint travel this long after a level change"; the rest of the line (deprecated, default 0) is right. The rule is game.exe's only after a hostile declaration; the arrival window is d2d's old rule.
- **B2** internal/nav/waypoint.go:66-72 (`waitCooldown`): default 0 since the earlier wpcooldown card, correct for game.exe and for d2d after #153. With the knob above 0 the bot waits for a window game.exe does not have; harmless on the lab binary, wasted time on a fixed d2d.
- **B3** the bot has no model of the hostile refusal (voice 0x13, no travel). In single player that is right (Q7). In a multiplayer run it would show as a click that closes the panel without travel; U, no multiplayer in scope.
- **B4** the earlier W1 finding (internal/d2dsrc/items.go:343 files the waypoint with class -1, so nav never finds it) is not re-checked here.

## PR candidates

No PR candidate. The deviation D1 was already proven and merged upstream (#153). D2 and D3 are not fixable inside d2d's current rules without adding a hostile model (D2) or are not reachable (D3).

## Limits and dead ends

1. The scans are linear sweeps. A write through `rep movs`, memcpy of a whole player-data block, or through a base pointer already offset (displacement other than 0x160) is not excluded: U. A dynamic check would be a watchpoint on the field over a long scripted session; not built.
2. The oracle hooks the player-data getter (a zeroed block), the target check (in range, same act, active), the unit lookups of the error path and the timing log; the claim covers the refusal window only, not distance, act or activity. The hooks are the earlier card's; I did not write a hook-free harness (a hook-free run needs a real player unit: not attempted, dead end for this pass).
3. The packet number 0x5d / 0x5e is inferred from the handler-table position (I) and the protocol lists (S); the table is not a plain pointer array, so my position scan only gave the slot offset, not an opcode.
4. D3 depends on the field and the clock both being 32-bit ticks; the 0x49 handler compare was run, not read.
5. d2d's "after the fix" binary was not run here; the earlier card's after-count (0) stands unrepeated. The `waypoint` rule has no row in the coverage denominators, so `docs/coverage/ledger.tsv` gets no line (the generator refuses an unknown subsystem).

## Reproduce (worker host, each under 1 minute)

    tools/emu/run-on-worker.sh vw2_xref.py            > $D2_SCRATCH/xref.txt     # Q1, Q2 (55 hits of 0x160; 4 call sites)
    tools/emu/run-on-worker.sh vw2_tab.py             > $D2_SCRATCH/tab.txt      # the 4-entry table
    tools/emu/run-on-worker.sh vw2_tab2.py            > $D2_SCRATCH/tab2.txt     # its one indexer
    tools/emu/run-on-worker.sh vw2_disp.py            > $D2_SCRATCH/disp.txt     # dispatcher, 2 callers
    tools/emu/run-on-worker.sh vw2_op.py              > $D2_SCRATCH/op.txt       # handler table slots
    tools/emu/run-on-worker.sh --ttl 1500 vw2_game.py 23 100000 > $D2_SCRATCH/vw2_game.txt   # Q3, Q4: seed 23, 100,000 cases
    tools/emu/run-on-worker.sh --ttl 1500 vw2_d2d.sh  > $D2_SCRATCH/vw2_d2d.txt  # Q5: needs the line above first
    tools/emu/run-on-worker.sh --ttl 1500 vw2_trace.py > $D2_SCRATCH/vw2_trace.txt  # Q6: 132,964 cases, seed 7

(`vw2_hdis.py` prints the two functions for reading; nothing from it is pasted.) The scripts import upstream's emulator and `tools/emu/wp_refuse.py` (this repository). Results of the committed run: vw2_game: field mismatches 0, rule B vs game 0 of 100,000, game refused 48,675; vw2_d2d: d2d vs game 43,558 differ, rule D vs d2d 0 of 100,000, rule B vs game 0 of 100,000; vw2_trace: 132,964 cases, 59,309 differ.

## Drill recipe (U to confirm later, lab binary; docs/forkdbg.md and docs/d2d-fork.md describe the verbs)

1. After a fix-carrying lab build: arrive by waypoint, open the panel at once, click a row inside 1 s, at 5 s, at 11 s: expect travel every time (D1). With a pre-fix build expect refusal under 10 s with no voice line.
2. Hostile rule (D2, B3): needs two level-9 players; U whether the lab can host that. If it can: declare hostile, then use a waypoint at 2 s and at 11 s: expect refusal with voice 0x13 at 2 s, travel at 11 s.
3. Town portal operate (D4): declare hostile, then click a town portal at 2 s, 6 s: expect refusal under 5 s.

## Matrix cells

No row of docs/research/*-matrix.md changes. Proposed text for docs/research/waypoint-trace.md section 1 (owner's call): add "pcdata+0x160 is read by three sites (waypoint 10 s, town portal 5 s, the declaration)", and "the compare is an unsigned wrapped sum (D3)".

## Glossary

A missing term for the merger: "tick wrap" (GetTickCount rolling over at 2^32 ms, about 49.7 days).

## Shared-file lines for the merger

See `openspec/changes/verifywaypoint2/merge-lines.txt`.
