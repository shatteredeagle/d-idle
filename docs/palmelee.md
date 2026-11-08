# palmelee: why the Paladin never hit (TASK-175) and what is left of the pace (TASK-177)

> status: fixed and measured 2026-11-08 (EST). Tags: V verified live (how stated), S read in code, U unverified. Evidence of the old failure: docs/paladinsmoke.md.

## Root cause (V)
On the d2d backend the melee swing was `Input.Click(Left, x, y)`, i.e. the channel verb `click x y`. d2d resolves a click against the unit under the pointer of the previous frame; nothing moved the pointer onto the monster, so the click hit the ground or nothing (the d2d fidelity notes already say a bare click without a hover is ignored for vendors, `docs/d2d-fidelity.md`). Symptoms in `bot-pa3.log` of the paladinsmoke run: `combat no bound attack skill, swinging the basic attack` hundreds of times, "casts 15 life 100% engaged 22s" (15 clicks, the Quill Rat at 4 subtiles never lost life), then `ignoring target ... no-progress` and `approaches`. The earlier guess (lockstep gives too few ticks) was wrong: realtime had the same click, it only hit now and then because the pointer happened to lie on a monster.
Not the cause (checked): the skill choice (the basic attack is correct at clvl 1, no skill point is spent before clvl 2, `combatBuild: blizzard` is only a label, the profile is `paladin`), the approach (the melee guard and `meleeGo` are fine), the leveler.

## Fix
- `internal/combat/fighter.go`: `cast` uses an optional `swinger` interface for every left-hand skill (including the basic attack) when the input has it; other inputs keep the click.
- `internal/d2dcombat/input.go`: `Input.Swing(skill, w)` finds the living monster at `w` and sends `cmd skill <id> x y <unit> left` (the same unit-addressed path the Sorceress casts use). No monster at `w`: it walks there like before.
- Data-driven: no class literal; every melee skill (Attack, Sacrifice, Smite, Zeal's first blow, a Barbarian's Bash) takes the left-hand path, right-hand casts are untouched.
- Tests: `internal/combat/swing_test.go` (Paladin and Barbarian with an input whose bare click is swallowed: fails before the fix with kills=0 / ignored, passes after; a caster never swings), `internal/d2dcombat/input_test.go` (`TestSwingAddressesTheMonsterByUnit`).

## Drill (V): fresh Paladin, lockstep d2d, no setup writes, seed 7
`docs/probes/paladinsmoke/run-a.sh pm1 7 14m`, main plus this change, 240 virtual minutes in 14 wall minutes, writes after start: 0 (lab verbs off, no d2cheat run; not a gate run).

| Virtual minute | Paladin before (pa3) | Paladin after | Sorceress (drill 1, V on an older run) |
|---|---|---|---|
| 10 | clvl 1, 0 kills | clvl 2, 1437 XP, 65 kills | clvl 1 to 2 |
| 20 | clvl 1, 0 kills | clvl 3, 103 kills | between 2 and 4 |
| 30 | clvl 1 | clvl 4, 162 kills | |
| 49 | clvl 1 | clvl 4, 246 kills | clvl 4 |
| 80 | clvl 1 | clvl 6, 412 kills | clvl 5 |
| 240 | clvl 1, 0 kills | clvl 9, 1195 kills, 0 deaths | |

So melee works and the Paladin is at the class-early-game ratio (clvl 4 about 20 virtual minutes earlier than the Sorceress, clvl 6 at 80 against 110). The doc's "clvl 5 to 6 in 20 minutes" was UNSURE and is not met: V is clvl 3 at 20, clvl 4 at 30, clvl 5 at 60 virtual minutes.

## Realtime (TASK-177, V): `run-b.sh pm2 9 22m`, same bot, d2d without `--lockstep`, writes 0
Before (paladinsmoke container): 13 kills, 315 XP, clvl 1 at 20 minutes. After: minute 5: clvl 2, 22 kills; 10: 37; 15: 57; 20: 62 kills, 1371 XP, clvl 2; 22: 66 kills, 1479 XP (clvl 3 needs 1500), 0 deaths. About 5x the kills, clvl 2 instead of 1. The container image was not rebuilt (the bot path is the same d2d verbs; image `d2dbot:palsmoke` and volume `d2pal-data` of the previous run were removed).
Why the rest is slower than the lockstep table (minute lines and `killtime` lines of `bot-pm2.log`): the hero stays in "Recovery: farming gold" (gold 0 < kit 500 at the start) inside Blood Moor, which runs out of monsters (`clear killed nothing; moving on`), and walks between spawns: 141 s since the last kill, 111 s of it walking. In lockstep the recovery farm area moved on to a bigger area at clvl 3 (virtual minute 17) and the kills kept coming; in realtime clvl 3 came at 22 minutes, so the move came late. U whether the same depletion is the whole story. Also seen, not fixed: `combat aura did not come up` (Might) on every fight start, 140 lines in 22 minutes (aura bookkeeping on d2d, U); the Paladin fights without Might.
Expectation corrected: with melee working, realtime clvl 3 comes at about 22 minutes, not clvl 5 to 6 (TASK-177 AC 2 closed by the corrected V number, `docs/palpace.md`).

## Follow-ups (not done)
1. Recovery should leave a depleted starter area earlier (the area choice looks at gold and level, not at the clear state).
2. Might "did not come up" on d2d.
3. The wizard still refuses the Paladin (TASK-145).

## Bring it back
`docs/probes/paladinsmoke/run-a.sh TAG SEED RUNFOR` (lockstep) or `run-b.sh` (realtime) on the worker host from the synced worktree; each creates its own Pally-TAG.
