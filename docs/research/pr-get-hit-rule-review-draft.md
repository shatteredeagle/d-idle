# PR #134 review: CodeRabbit findings, analysis and draft replies (agent HITREC, 2026-10-30)

> status: PR #134 follow-up pushed and replied 2026-10-31 (commit e3c1e535). Draft text below was posted. PR https://github.com/psi29a/d2decomped/pull/134 (head d55973e1, base 32768ddb). Findings read in full with GET only (`gh api repos/psi29a/d2decomped/pulls/134/comments` and `/reviews`; CodeRabbit review 5720082484, two inline comments 4429578410 and 4429578414). Checks at read time: CodeRabbit pass, guard pass, Windows (MSVC) pass, Flatpak (aarch64) pass; Linux (GCC), macOS (Clang), Flatpak (x86_64) pending. Tags: V = read in game.exe's dump or run on the worker host, S = secondary, U = unverified. The fix commit is on the same branch in the clone `~/d2d-lab/pr/hitrec`: `e3c1e535` on top of `d55973e1` (author and committer shatteredeagle, no trailer).

## Finding 1: fight.cpp:4800 (and 4907), Major, "Evaluate get-hit for each landed blow": VALID

- V (game.exe): the get-hit is decided once per damage record, on that record's total +0x4c. `FUN_0057c6c0` (apply one record) and `FUN_0057cee0` (the takes-damage event, the `FUN_0057cb00` call on result flag 4) are called as a pair per hit: `0x56ba73` / `0x56ba7f` (the melee hit), `0x57d5c7` / `0x57d658`, `0x5a6e82` / `0x5a6e90`, `0x5ad8c6` / `0x5ad8d3` (the missile hit). Two hits in one tick are two records, two tests, two seed draws; a unit already in GH restarts it (`FUN_005809d0` mode 4 / `FUN_005a7e60` mode 3 are called again).
- V (d2d): `Foe::take` adds every blow of the frame into `foe.damage` and keeps only the last blow's `hit_class`; `hurt_side` tested that sum once (the player at :4903, the merc at :4800). That was upstream main's structure too (the twelfth on the sum), and my first commit kept it. The monster side was already per blow (`Fight::land` -> `hurt` per blow).
- V (differential, `tools/emu/hit_rule.py` with multi-record frames: 73,694 frames, 21,532 of them with 2 or 3 records on one unit in one tick, seed 7; game.exe's `FUN_0057cee0` run once per record on the same unit): the summed model of d55973e1 (with the stun / frozen / poison-only / GH-mode gates modelled as the after side models them) differs from game.exe in **702 frames, all player-side, all multi-record** (0 of the 52,162 single-record frames); the per-blow fix (commit e3c1e535, upstream's `test_combat --get-hit`) differs in **0 of 73,694**. CodeRabbit's example (two class-2 blows of 6 on 64 life: frame 2264) is one of the 702: game.exe gives no get-hit (each record under an eighth), the sum gave a 75 % roll. Upstream main's own rule differs in 13,935 of the same frames.
- Fix (e3c1e535, +56 -22, amended after the verifier): `Foe::Blow { damage, hit_class, marked }` and `Foe::blows`, pushed in `Foe::take` when a blow lands, the class set by the new `Foe::struck_by` at the six sites that set `foe.hit_class` (ai.cpp x5, ai.hpp:769), an aura pulse pushing its own blow; `hurt_side` tests `rules::get_hit` once per blow (`struck_big`: every blow rolls, any get-hit wins) for the player (d2d's `rng`) and the merc (`merc->seed`). `test_game`'s arrow scenario lands its one blow through `blows`; `test_combat --get-hit` takes a 17th column `next` (records of one frame) and prints one line a frame. Tests: test_combat ok, test_game 23 OK, pr_guard 0 FAIL 0 WARN, presubmit 0 FAIL (the same 3 WARN as before).
- Verifier round 2 (independent, Fable, read-only; re-ran the oracle, fresh `gen 30000 4242` 23,778 frames 0 differ, 7 hand frames; read the flag-4 path: no current-mode test, so a unit in GH is re-tested and re-rolls per record, V): HOLDS WITH LIMITS, two of which were fixed in the amended e3c1e535: (1) the per-blow test used the pre-absorb damage while `absorb()` (Bone Armor, Cyclone Armor, Energy Shield) runs on the frame sum; game.exe sums a record's total after the absorb states (FUN_0057c1e0 at 0x57c3a5 after state event 0xb at 0x57c2fc): the absorbed amount now comes off the blows in landing order (a pool record by record; Energy Shield's share in that order too, ponytail); (2) burning-ground missiles (ai.hpp:763) added damage without a blow: they push one now (whether game.exe's record carries flag 4 there is U). Left and to say: an aura pulse's blow does not count as a `hit`, so a stunned player hit only by an aura gets no GH; `merc_hurt` (thorns, poison ticks) and `self_hurt` have no blow (no GH, as game.exe's poison-only rule; thorns U); a first record's GH changes the mode mid-tick in game.exe while d2d's frame keeps one motion for all its records (hit / block rolls, not the get-hit rule); the oracle hooks the room to 0 (the prologue's attacker gate at 0x57cefd..0x57cf37 is not modelled).

## Finding 2: tests/test_combat.cpp:68, Minor, one-letter loop index: VALID (trivial)

- `i` -> `sample` in e3c1e535 (the repository rule: no one- or two-letter names). presubmit's own short-names check had not flagged a loop index.

## Draft replies (not posted)

Reply to finding 1 (comment 4429578410):

> Right, thanks. game.exe decides the get-hit once per damage record: `FUN_0057c6c0` (apply the record) and `FUN_0057cee0` (the event that calls `FUN_0057cb00`) run as a pair per hit (0x56ba73 / 0x56ba7f for a melee hit, 0x5ad8c6 / 0x5ad8d3 for a missile), so two hits in a tick are two tests on two records, each rolling on the unit's seed. `Foe::take` summed the frame and the test ran once on the sum with the last blow's class (upstream main did the same with the twelfth; the monster side was already per blow through `hurt`). e3c1e535 keeps every landed blow in `Foe::blows` (damage, hit class, element mark) and `hurt_side` tests each one; the merc the same on its own seed. On 73,694 frames of game.exe's own `FUN_0057cee0` (21,532 with 2 or 3 records on one unit), the summed version differed in 702 frames, all multi-record, your two-sixes-on-64 case among them; the per-blow version differs in 0. test_combat and test_game pass.

Reply to finding 2 (comment 4429578414):

> Done in e3c1e535: `sample`.

## Commands once the user says go (NOT RUN)

```sh
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && git log --format="%h %an %s" -3 && python3 tools/pr_guard.py upstream/main'   # e3c1e535 on d55973e1, shatteredeagle
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && git push origin fix/get-hit-rule'            # ONE push: fast-forward d55973e1..e3c1e535, no force
# then the two replies (after the push, so the hash resolves):
ssh worker.example "cd ~/d2d-lab/pr/hitrec && gh api repos/psi29a/d2decomped/pulls/134/comments/4429578410/replies -f body=\"$(cat reply1.txt)\""
ssh worker.example "cd ~/d2d-lab/pr/hitrec && gh api repos/psi29a/d2decomped/pulls/134/comments/4429578414/replies -f body=\"$(cat reply2.txt)\""
# then: gh pr checks 134 --repo psi29a/d2decomped --watch
```

Save each reply text (the blockquotes above, without the `> `) to `reply1.txt` / `reply2.txt` in the clone first. Cleanup: none left (no game started; the test binaries in `/tmp/hitrec-build` are files, no process).
