# Upstream PR candidates from the movement and hit-recovery trace (agent DECOMPREC, 2026-10-30)

> status: findings only, NO PR drafted. The user's rule of 2026-10-30 ("PRs as close to absolutely sure with verifiable proof": a differential test against game.exe's own code under the emulator over boundary cases and 100k+ random cases, mismatches before the patch and zero after, no U link in the causal chain; otherwise report it as a finding) is not met by either candidate yet. The standard doc docs/process/upstream-pr-standard.md (agent PRPROOF) had not landed when this was written. Nothing was pushed, posted or filed; the fresh clone `~/d2d-lab/pr/gh-rate` (user's fork, remote `upstream` = psi29a/d2decomped at 32768ddb) has an empty branch `fix/gethit-rate` at upstream/main and no changes. Trace: docs/research/movement-and-recovery-trace.md; oracle: tools/emu/move_recovery.py.

## Candidate 1: a get-hit plays at AnimData speed x (50 + effective FHR) / 100

PROOF: incomplete.

- Claim (one): d2d rates a player's GH by `speed_frames` (256 + 256 E / 100, i.e. 100 + E; fight.cpp:456) and a monster's GH by the plain AnimData length (ai.cpp:172); game.exe's GH branch of `00623f50` (0x62409f..0x6240b1, tail 0x62435c..0x624381) stores speed x (50 + E) / 100 at unit +0x4c. A Sorceress's 8-frame GH: 15 ticks at 0 FHR, not 8; a Fallen's 7 frames: 13, not 7; a Zombie's 5 at speed 152: 16, not 8.
- What is proven (V): the rate formula, by running that branch under unicorn (`move_recovery.py rate`: 0..175 effective at speeds 256 and 152, 352 cases, all equal to `speed * (50 + E) / 100`); `00621740` row 1 (E = 120 v / (120 + v)) by the same oracle; the plain-mode tick count `ceil(256 frames / rate) - 1` for the client's player tick (`006217c0` + `00463390`, docs/research/cast-trace-gameexe.md, merged as PR #128). The Sorceress FHR breakpoint table 15/14/13/12/11/10/9/8/7/6/5 at 0/5/9/14/20/30/42/60/86/142/280 falls out of it (S, community table; corroboration only).
- Missing before a PR:
  1. The monster side's causal chain has a U link: the server's mode end for a monster in GH (`005a8030`'s caller, whether it uses the same look-ahead end test as the client tick) was not read. Until it is, "13 ticks" for a Fallen is 13 or 14.
  2. The 100k random differential: `move_recovery.py rate` sweeps the branch after the stat call; the standard asks for random (stat, speed) pairs through `00621740` + the branch against the patched `rules::gethit_rate`, with the mismatch count before (d2d's `speed_frames`) and after. The emulator run is cheap (the branch is 20 instructions); the harness lines exist in move_recovery.py and need the random driver and the d2d-side `test_combat --gethit-rates` printer.
  3. "Nothing else changes": upstream tests plus state hashes on a scripted scenario (a hit on the player, a hit on a Fallen) with only the mode's end field differing. Not run: the fresh clone has no build.
  4. The block branch (0x62403d: 50 + FBR, 100 under state 0x65) is the same shape and should stay out of this PR (one claim), but the reviewer will ask; say so in the body.
- Draft body (to fill once 1..3 are done): "What changes: a get-hit (GH) lasts `ceil(256 x frames / (speed x (50 + E) / 100)) - 1` ticks for the player and for monsters (FHR stat 99); d2d played it at (100 + E) for the player and at the AnimData length for monsters, about half as long. Evidence: `00623f50` 0x62409f (the GH branch: `00621740`(unit, 1) + 0x32), 0x62435c (x speed / 100, 0..0x7fff, +0x4c); `006217c0` / `00463390` (the end test). Re-check: `uv run python gethit_rates.py > g.txt; ../../build/tests/test_combat --gethit-rates | diff g.txt -` (no output) and the random driver with its before/after counts. Agent: Claude Code (Claude, working for the maintainer)."

## Candidate 2: the get-hit decision rolls by hit class

PROOF: incomplete.

- Claim (one): d2d flinches a monster at damage x 8 >= max life (ai.cpp:2853) and a player at x 12 (fight.cpp:4797), always above and never below. game.exe's `0057cb00` uses a divisor by the hit's class (damage struct +0x60: 8 for classes 2, 6, 10, 11; 32 for 4, 8; 64 for 5; 16 for the rest, Charged Bolt's HitClass 64 among them): no get-hit under max/d, 50 % under 2 max/d (the unit's seed, rand(2)), 75 % under 4 max/d (rand(4)), sure above; never on a frozen unit (state 1).
- What is proven (V): the function's whole decision for hit classes 0..13 and 64 at 13 damage fractions x 4 roll values under unicorn (`move_recovery.py hit`, with the state test, max-life getter, seed roll and MonStats2 bit hooked); the jump table at `0057cbf8` read from the image; Charged Bolt's HitClass (D).
- Missing before a PR:
  1. The damage struct field +0x28 (too small when not 0 and equal to the damage): meaning U; the oracle holds it at 0. The hit class source for a melee blow (weapon hit class into +0x60) was not traced, only the missile case is D.
  2. The seed: game.exe rolls the DEFENDER's seed (unit +0x20) with `00472210`; d2d's monsters carry a seed (`monster.seed(n)`) but the player-side roll's seed and order would need to match for a state-hash proof; a 100k random differential on the decision alone (inputs: max life, damage, hit class, roll) is straightforward and not yet written.
  3. "Nothing else changes" as above.
- Effect size: small in Act 1 Normal (every surviving monster takes over 1/4 of its life from a level-1 bolt: sure on both), larger from Act 2 on and for melee hits in the 1/16..1/4 band.

## Not a d2d bug: the d2d bot walks

The engine speed is identical (velocity / 16 subtiles a tick on both, `move_recovery.py step` vs `cells_per_sec`); the d2d bot is up to 1.5 x slower on foot because `internal/d2dinput` never sends d2d's `cmd::Run` (world.cpp:2245) and `running` defaults false. Fix on our side (a card for d2dinput: send the run toggle once in game; d2d's stamina port then walks at 0 stamina as the original does). No upstream PR.
