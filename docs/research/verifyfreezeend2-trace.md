# VERIFYFREEZEEND2: a think that comes due during a get-hit, and a freeze that lands during one (game.exe vs d2d)

> status: card verifyfreezeend2, 2026-11-02, offline (no game, no d2d slot, no Go change). Second pass of VERIFY-AIDEL (board item 11): the part the first pass did not check, the freeze-end claim and a think that comes due during a get-hit. Already answered and NOT redone: the think schedule FUN_005a73e0 (verify-aidel.md, 352,800 cases), the freeze rules (freeze-rules-trace.md), the freeze-end callback 0x57b170 and the event queue under the real emulator (pr-freezeresume-verifier.md, 34,299 claimed cases, no off-by-one).

Tags: V run (game.exe's own code under the emulator against an oracle I wrote, or a real d2d build run with my probe), V read (disassembly, address and instructions), D (MPQ tables), S (d2d source, file:line), I (inferred), U (unverified).

## Summary

1. **Freeze end (the item's first half): nothing new to find.** The claim "a frozen monster thinks again at the freeze end + aidel" was already run for real by pr-freezeresume-verifier.md (V run, real queue). I re-read 0x57b170 only for the rules below and re-ran its interplay with a get-hit (Q3): consistent.
2. **A think due during a get-hit: a deviation, D1 (V run on both sides).** game.exe runs the AI driver at the due frame whatever the unit's mode is, a get-hit included; d2d holds the think until the get-hit's mode ends. Of 1,133 cases with the think inside the get-hit, 1,133 differ (d2d later by 1..16 frames, the rest of the get-hit); 1,371 cases with the think after the get-hit: 0 differ. That the driver then USES the opening (starts an attack, or stands, out of GH) is V read, not run (U for the run, Q2).
3. **A freeze that lands during a get-hit: game.exe loses the get-hit's end event** (it is gated on state 1 and dropped while frozen; V run, 3,000 cases, 0 differ from my rule). The monster stays in mode GH until the think the freeze end files (end + aidel) runs the driver. d2d plays the get-hit out then stands (S read only, D2, no d2d run).
4. **No PR candidate** (Q6): D1 is proven with a real differential but the game-side half "what the driver does at that frame" is V read, and there is no existing d2d fix to prove before / after with; recorded as a deviation, not a PR.

## Questions

| Q | Question | Answer | Tag |
|---|---|---|---|
| Q1 | What does a get-hit start do to a pending think (type-2 event)? | Nothing. `FUN_005a7c20` (mode set) clears event types 0 and 1 of the unit through `FUN_00553990` (0x553990: `FUN_00540e60(unit, 0, 0)` and `(unit, 1, 0)`, pushes at 0x553992..0x5539a7), never type 2; the mode-3 start function 0x5a7580 returns 1, so `FUN_005a73e0` (the only refile, 0x5a7d9c) is not called on that path (0x5a7d67..0x5a7da5: refile only when the start function returns 0; which modes return 0 was not run, I: NU, whose start function IS 0x5a73e0, mode table row 1). A pending think stays filed at its frame | V read; V run (below: 3,000 cases, driver entered at exactly T) |
| Q2 | Is the AI driver (event type 2, handler 0x5b1740) entered while the unit is in GH (mode 3)? | Yes. `FUN_005a7f80` gates types 0, 1, 2, 6, 7, 9, 10 on state 1 (freeze) only (pr-freezeresume-verifier.md section 1); the driver 0x5b1740 reads no mode (0x5b1740..0x5b1876: stun step `FUN_005b10e0` tests state 0x15 only, door `FUN_005b0f50`, flee `FUN_005b0ff0` tests mode 0 (DT) at most, search `FUN_005b1650`, then the AI function through the table). Run: unit mode 3 at the driver entry in all 1,271 of 3,000 cases with T before the get-hit end | V read; V run (entry) |
| Q3 | What can the driver do from GH? | Start an attack: the Zombie think `FUN_005efe20` reads no mode (0x5efe20..0x5eff4d; only the seed, the target and `FUN_005dd2b0` last mode), the hand-off `FUN_005ddf90` is `FUN_005a7e60` + `FUN_005a7c20(game, req, 1)` (0x5ddf96..0x5ddfb4); `FUN_005a7c20` tests state 0x36 and dying only (0x5a7c50..0x5a7cb6) and the attack start 0x5a75c0 tests neither GH nor the old mode. Or stand: `FUN_005de080` sets NU unless the unit is already NU (`cmp [unit+0x10], 1` at 0x5de09d), so a stand from GH forces NU, removes type 2 events and files the next think at now + n. So game.exe's get-hit is cut short by the next think | V read (U for a run: the A1 start and the mode set were not run, they call about ten path / animation functions I did not hook) |
| Q4 | Get-hit length on both sides in this build | 16 ticks of animation, end event at frame hit + 17 on game.exe (5539b0 run through the real `FUN_005a7c20`, Zombie rate 76 = 152 x (50 + 0) / 100); d2d's Zombie GH ends at 17 too (probe) | V run both |
| Q5 | A freeze lands at frame f, length n (after the divisor) while the unit is in GH that ends at E = hit + 17: is the GH end event delivered? | Dropped iff f <= E <= f + n (the type-1 event comes due while state 1 is on, `FUN_005a7f80` drops it, nothing refiles it); delivered if E < f or E > f + n. The think is at f + n + aidel (the freeze removed the pending type-2 event, 0x57b170 files the new one). 3,000 cases, 1,952 dropped, 1,048 delivered, 0 differ from this rule | V run |
| Q6 | PR candidate? | None, see Summary 4 | n/a |

## Deviations (d2d vs game.exe)

| id | game.exe | d2d | tag | differential |
|---|---|---|---|---|
| D1 | a think due in a get-hit runs the driver at its frame (Q1, Q2); the driver may start an attack or stand out of GH (Q3) | `monster_update` returns while `mode == "GH"` and `now < mode_until` (ai.cpp:1882-1885 in the build base, :1718-1722 in the older checkout), the think waits for the GH end | V run both sides for the entry/swing frame: 1,133 of 1,133 differ with T inside the get-hit (d2d swing - game think = 17 - T, deltas 1..16, top counts consistent), 0 of 1,371 with T after it; V read for "the driver acts" | yes, built: `vfe2_all.sh` (below). Oracle limit: the game column is the driver ENTRY frame, d2d's the first SWING; they coincide for a Zombie in melee (VERIFYFREEZERESUME used the same pair), the entry-to-attack step is V read |
| D2 | a freeze during a get-hit drops the GH end event; the monster stays in GH until the think at freeze end + aidel (Q5) | `ai.cpp:1878`: stunned and not GH/BL stands; a GH plays out first and then the stand | V run game side (3,000), S read d2d side (no d2d run: the build base's freeze path is the stun site, a refile is not in this base) | could be built: a probe like test_vfe2_gh.cpp with a `Fight::strike` freeze at f after `hurt`, oracle `vfe2_run.py frz`; the observable (what the player sees) is a standing monster either way, so low value (I) |

Not deviations (checked): GH start does not refile (d2d's GH start leaves next_act alone too, V run: 1,371 cases after the GH match), and the GH length matches (17 vs 17).

## Bot findings

- **B1: none.** No Go code models a monster's get-hit or think timing (`grep` of internal/ and cmd/ for hit recovery, get-hit, flinch, stagger finds only party/launch staggers and drill docs). Consequence for the bot's assumptions, in text only: "a staggered monster cannot act for its whole recovery" is wrong on game.exe by D1 (it can act at its next think, at most `aidel` = 13 to 15 frames after the hit for a monster that was idle, V read + V run for the entry); drill and kiting docs that rely on the full 17 frames of GH should say "up to 17".

## PR candidates

None. D1 is proven at the entry frame, but there is no before / after fix to run, the "acts out of GH" half is V read, and changing d2d's GH hold is a behaviour change the user must decide (I do not know in-game evidence of monsters attacking out of hit recovery; the live drill below would give it).

## Method

- Game side: `tools/emu/vfe2_run.py` on top of the VERIFYFREEZERESUME oracle (`verify_freeze_resume.Oracle`: real event queue FUN_005417d0 / 005416b0 / 005414d0 / 00541060 / 005a7f80, FUN_0057b230, state lists, 0x57b170). New and run natively: `FUN_005a7e60` (request), `FUN_005a7c20` with requested mode 3, its start 0x5a7580, `FUN_00553990`, `FUN_005539b0` (files the GH end), the type dispatch to the handler 0x5a7be0 (hooked to record its frame). Hooked (listed in the script docstring): FUN_005533d0 (sets +0x30 / +0x44 / +0x48 as read at 0x55343c..0x553469), FUN_00553570 (writes the mode, read: calls FUN_00624690(unit, mode)), FUN_005a68e0 (saves the last mode), FUN_00649400, FUN_005a4360, FUN_0046c140 (mGH bit: 1), the driver 0x5b1740 (records entry frame and unit mode). Data: MonStats `aidel` triples as the first pass (D), game type 3, Zombie GH animation speed 152, 5 frames.
- d2d side: `tools/emu/test_vfe2_gh.cpp`, the real `hurt` and `monster_update` on a zombie1 body with the case row's loaded aidel, hand-compiled by `vfe2_build.sh` against the scratch upstream build (same recipe as ms_build.sh; base: the sorcrest scratch checkout, ai.cpp GH hold at :1882).
- Oracle for `gh`: driver entry frame = T; unit mode 3 when T < GH end. Oracle for `frz`: rule in Q5 written before the run.

```sh
tools/emu/run-on-worker.sh --ttl 900 vfe2_run.py gh 0x5EED0001 3000     # gh: cases 3000, 1271 due inside the GH, differ 0, GH end frame 17 in all
tools/emu/run-on-worker.sh --ttl 900 vfe2_run.py frz 0x5EED0002 3000    # E = 17; cases 3000 differ 0 {end dropped 1952, end delivered 1048}
tools/emu/run-on-worker.sh --ttl 2400 vfe2_all.sh 0x5EED0003 2000       # 2,504 cases (504 grid: 14 aidel triples x 3 difficulties x 12 T, + 2,000 random)
#   think after the game GH end 0 of 1371 differ; think inside both GHs 1133 of 1133 differ; GH end 17 on both sides
```

Counts: gh 3,000 (1,271 inside) + frz 3,000 + differential 2,504 = 8,504 cases, all on real MonStats rows' aidel columns.

## Limits

- The attack start / stand out of GH are read, not run (Q3). The 91 AI functions were not all read; the Zombie think, the driver and its first steps were. A different AI function might refuse in GH (U).
- Only a Zombie GH (speed 152, 5 frames, FHR 0) was run; game.exe's GH rate and length with FHR stats are the pr-get-hit-length-proof.md claim, not repeated. The hit lands in the missile / player pass before the monster group (S, FUN_005414d0).
- State 0x36 (uninterruptable) and dying are not modelled (nothing sets them here).
- D2 has no d2d run.
- The freeze-end half was not re-run (already done by the verifier).

## Drill recipe (lab, d2d side; the original game cannot be driven)

Needs the lab binary of docs/forkdbg.md (`debug mon set`, `debug cast ... unit`, `monsters`, `step`), a throwaway offline Sorceress, Blood Moor, one Zombie or Fallen made 1000 life.
1. D1 on d2d (confirms the harness read of `monster_update`): a monster at melee reach of a standing hero, `debug cast <Charged Bolt or Fire Bolt> unit <id>` so it flinches; poll `monsters` every `step 1` and note the frame its mode leaves GH and its first A1. Expect the first A1 at the GH end (17 frames), never earlier. This only re-checks d2d (V run already).
2. D1 on game.exe (U, not drivable here): the same read on the original game needs a way to see monster mode per frame; not available in the lab. The proxy is a recorded live fight against a Zombie with the bot's own log: look for a monster A1 starting less than 17 frames after a hit on it (the bot's combat trace); the hit frame is the cast's missile frame (docs/research/sk-missile-trace.md).
3. D2 (freeze in GH): needs Glacial Spike on a monster in GH: cast Fire Bolt then Glacial Spike the next tick; read `monsters` mode over 120 frames. d2d expected: GH out at 17, then standing; game.exe expected (V run rule): GH held until freeze end + aidel.

## Matrix-cell updates (proposed text)

Matrix `docs/research/sorc-skill-matrix.md` rows (Frost / Cold skills, Glacial Spike, Frozen Armor) and the monster-think rows: "think period: aidel per difficulty, 0 -> 15 (V run, 352,800 cases) ; freeze end files the think at end + aidel (V run, 34,299 cases) ; a think due in a get-hit runs at its frame on game.exe, d2d holds it to the GH end (D1, V run 1,133 of 1,133, VERIFYFREEZEEND2) ; a freeze during GH drops the GH end event on game.exe (V run, 3,000)". Coverage ledger: no line added (rows 1, 11 of `states` and the monster AIs are already claimed by other evidence; the coverage tool refuses a second line for the same row; this is an aspect, not a row).

## Dead ends

- Running the full Zombie think plus the attack start in GH: stopped, the mode-set path for an attack calls 648b90 / 648ad0 / 6490e0 / 5a63f0 / 5a4f50 / 5a4350 / 648780 / 463860 / 649970 / 620250 / 56faf0 and I did not pin each one's stack arguments; the claim stays V read.
- My first dump of the monster mode table started at 0x6e2298 and printed misaligned rows (`FUN_005a78a0`'s default table is 0x6e2260 + 16 x mode: [start, event 0, event 1, flag]; mode 3 start 0x5a7580, mode 4 start 0x5a75c0, mode 1 (NU) start 0x5a73e0): fixed (`vfe2_tab.py`).

## Shared-file lines for the merger

See openspec/changes/verifyfreezeend2/merge-lines.txt.
