# Review sweep 2026-11-28: what is still open from the 13 Fable reviews

> status: current on 2026-11-28 (card sweep1, TASK-1122). A snapshot: statuses were read from the cards and branches named below at the time of the sweep; re-check a row before acting on it.

Source: `docs/design/fable-review-2026-11-28/01` to `13`. Evidence tags: V = read in the card, the branch diff or the code in this run; S = taken from the round-2 reviews (10, 11, 12), which re-read the fixes; U = not checked.

Status words: `fixed-by <card>` (on main), `in-flight <branch>` (committed on an agent branch, not merged), `OPEN` (no card, no branch), `needs-owner` (a decision, not code), `needs-live` (only a live or lab run can settle it), `needs-fork` (needs a change in the d2d fork), `ops` (lives in the private ops repo; the brief names ops3 for it).

Cards on main: fix1 TASK-1095, fix2 TASK-1099, fix3 TASK-1094, fix4 TASK-1100, fix6 TASK-1098, fix7 TASK-1096, fix8 TASK-1097. Branches: `agent/fix1b` (TASK-1114), `agent/vault2` (TASK-1113), `agent/fix7b` (TASK-1115), `agent/fix8b` (TASK-1117), `agent/live1` (TASK-1119), `agent/ops2` (TASK-1116), `agent/led1` (TASK-1110), `agent/evid1` (TASK-1118), `agent/grad1` (TASK-1112), `agent/cls1` (TASK-1111), `agent/cls2` (TASK-1109).

## Fixed in this branch (agent/sweep1)

| finding | sev | what | test |
|---|---|---|---|
| 12 C3 | medium | `Consent.Granted` only answers: a look by another hero or by a snapshot without a hero name no longer deletes the owner's difficulty go; such a go is never used and is logged once, not at every poll. Only an ended go is dropped by a read. | `internal/diffready` `TestConsentHeroAndExpiry`; `internal/bot` `TestDifficultyGoIsPerHeroAndEnds` (Sorceress, Paladin). Both asserted the old delete and were turned round. |
| 03 #5 (part) | medium | A gap landing (Leap Attack, Charge) is refused in a cave mouth's trigger area; gap casts and whirl legs count as casts of the fight, so the cast watchdog does not force casts into them. | `internal/combat` `TestGapAndWhirlAskTheMouthGateAndCountAsCasts` (Barbarian, Paladin) |
| 12 low 6 | low | The rate-limit refusal of the difficulty go prints `difficultyGoPerHour`, not a literal beside it. | none (text only) |

Correction to review 3 finding 5 (V, `internal/combat/fight.go` `reachable`): escape and whirl landings already ask the mouth gate, because `f.valid` refuses a point for which the mover's `NearMouth` is true. Only the gap landing did not.

## OPEN, no card and no branch (the list this sweep was asked for)

Ordered by severity. "Why not here" says why this branch did not fix it.

| finding | sev | what is open | why not here |
|---|---|---|---|
| 10 N4 (rest) | medium | Direct Equip: `saveBookLocked` ignores write errors (the journal can silently not exist); an order with an unparsed `Reply` can be failed in the same call, and one that survives blocks the record for ever with no owner way out | `internal/vaultkeep` is being rewritten by fix1b and vault2 (their cards list it as left) |
| 10 N8 / 01 F6 | medium | Twin Keep ends "record in the vault, item maybe on the hero"; the kept count is not reduced when the displaced twin is filed | same files as fix1b/vault2 |
| 10 B1 | medium | Deaths add up over the whole life of a stored run (more `died_too_often` parks than the baseline) | needs-owner: window or documented behaviour |
| 10 section 4 | medium | Test gaps: no two-keeper race test, rung 4 pairing holds by the wording "not ready" only | vault part in vault2 (V criterion 5); the rung 4 test is OPEN, `internal/bot` invariants are in flight (fix8b) |
| 03 #5 (rest) | medium | `escapePlan` needs contact AND low life; the Sorceress rows' source text says "or" (U, SME medium confidence). `escapeCell` ignores hostiles beyond 30 subtiles; no hop cooldown | needs-owner: a rule choice by the SME (a plain OR hops at low life with nobody near); V the code comment says "and" |
| 02 #8 | medium | Hand-in: no derive before each attempt (a hop that worked can answer "failed 3 times"); a tapper that leaves mid-dialogue leaves the awards unsettled; hand-in is not served in gold farm, recovery or under the arbiter hold | `internal/questboard/exec.go` and `internal/bot/questboard.go` are in flight (fix1b, live1); the last point is a doc line |
| 11 N7 | medium | Stance: a melee row under its life floor leaves from every target (a, zeal); a surround leave is followed by "inside" on the next tick (b). Only when a row carries `enabled` (none does, V grep) | dark today; needs the SME's per-kind floor |
| 11 N8 | low | `earlyRouteCheck` asks kit before every row's life floor; the differ cell "words" has no counter | `internal/bot/earlyroute.go` is in flight (fix7b lists N8 as open) |
| 11 section 4 / 07 A4 | medium | 40 minute hold slice against a 120 s lease: no slice budget | fix7b documents it and leaves it open (no data row); step 3 of `docs/design/arbiter-target-and-migration.md` |
| 07 A5 | medium | Player-origin hooks (points, checkpoint, pause, recall, errand, repair trip, save point) have no arbiter source and wait under control | design step, arbmig TASK-973 is the home; no card names it |
| 07 A6 | medium | Area choice outside the arbiter, five hysteresis mechanisms | design; TASK-973 |
| 07 A7 | medium | Aura has three writers and no dwell; planner `agree` ignores the aura | OPEN; fix3 kept the profile rules ahead of the driven aura, the dwell is not built |
| 07 A8 | medium | Under control the agreement is by construction; a control night cannot feed the verdict | needs-live (fix7 criterion 7: one control-0 recorded night) |
| 07 A9 | low | `escalate` drops an emergency trip; fight stall is checked before safety in the guard; a stuck "portal in flight" mutes the guard (U); `arbChores` has no idle check | `internal/bot/loop.go` in flight (live1, fix7b) |
| 07 admission | medium | Before `bot.admission` 1: an exhaustive old-against-new equality test, `fillEarly` reading the function, `View.Knobs`/`View.Legacy` never filled, purposes farm/boss/quest-walk unused | fix7 closed differences 1 and 2 and split the counter (V card result); the rest is OPEN |
| 12 C2 | medium | Flip run check reads the exec environment and PID 1 only and swallows errors; watcher revert and autodeploy have no check | `tools/flip/flip.py` is in flight (led1); autodeploy is ops |
| 12 C4 | medium | Scrubber: lowercase or spaced key, multi-word or unlabelled passcode still leak; JSON lines broken; commit hashes hidden; two-word home rule eats prose (low 7) | OPEN, a shell rule set with fixtures; not started for time |
| 12 C3 (rest) | low | Refused requests count against the hourly go budget; `GoTTL` is a Go variable, not table data; an old-format `diffgo.json` is dropped as unreadable | OPEN |
| 12 D2 | medium | "Every run recorded" compares two counts of one spool; a failed ingest still exits 0; a run cut by the nightly timeout is not counted | `cmd/d2scenario/run.go` is in flight (nf1b) |
| 12 D3 | medium | Stability keys on the bot binary only: an edited scenario reads as a flake; agent-branch runs can make a key stable; the older report compares commits; the cell follows the latest seed | needs a setup hash in the ledger line written by `cmd/d2scenario` (in flight, nf1b) |
| 12 E3 | medium | Docs-class and scoped gates skip Go tests that read a file by path; the flip and autoflip Python tests run in no gate | ops (merge classifier) plus `tools/ci/scripts.sh` |
| 12 low 5, 8, 9, 10 | low | `eachLine` joins the line after a NUL hole; `golden.go` ignores the `scenarioHero` error; smoke marker regex pins "prep 4"; `MercAura` always answers true | OPEN, each a few lines |
| 02 #9 | low | Reward hold is per pending stretch, not per quest id | live1 rewrote the same function for L4 (V card): fix after it merges |
| 02 smaller | low | No duplicate check on step executor adapter keys (questspec rule 15); seven dead SM events; `refused_repeatedly` nearly unreachable | OPEN |
| 03 #7 | low | Crowd aura held against a single boss; War Cry cast without line of sight; zone lock reopens after 8 h | OPEN |
| 10 N6, B3, B4 | low | "This order filed it" looks at live records only; hand-in park writes counters without the seed; the go is taken before the hop is verified | fix1b lists all three as left; files in flight |
| 01 F7, F8, F10, F11 | medium, medium, medium, low | Old Equip path between Wear and filing; order files not fsynced and credit ids across a d2d restart (U); log not truncated after a short write; price cache has no difficulty in its key | F7/F8/F10 sit in `internal/vaultkeep` and `internal/vault/store.go` (in flight); F8's id question is needs-fork (`docs/design/fork-order-id-spec.md` on vault2) |
| 04 #6, #9, #10, #11 | medium, low, low, low | Dev update server lists a directory on every interface for 8 h; login limiter keyed by peer address; option injection through unvalidated names; `err.Error()` with file paths returned to the page | OPEN (not in fix4's criteria, V card); #6 and #10 are in `tools/dev-update*.sh` |
| 13 L9 | low | Empty sweeps of a level emptied on the way through; `combatBuild=blizzard` printed for a Paladin; `tick` lines carry the previous boundary's area; save point timer restarts on something other than the last save (S) | OPEN; fix8 removed the silent fallback, U whether the label line changed |
| 06 E3, E7, E8, fix 9, fix 10 | high to medium | Log-only scenarios may gate; three flake rules disagree; thin fixed seeds and classes, no held-out seed | OPEN rules; needs-owner for the one flake rule and the matrix floor |
| 06 F1 to F5, fix 6, fix 7 | high | `flip.py` checks no evidence (`--evidence <run id>`), no knob-off control arm, no exercise counter | OPEN; evid1 adds evidence log lines and scenarios, not the tool rule; `tools/flip` in flight (led1) |
| 05 1.2, 1.3, 1.4, 1.6, improvements 2 to 6 | high | Watcher fault against bot fault, exposure and exercise proof before CLOSE, deaths counted, save snapshot at flip, scoped restore | OPEN in `tools/flip` (in flight by led1) and ops |
| 08 fix 2 | medium | Per-level skill cap: one live read on Game.exe decides the leveler term | needs-live |
| 08 fix 7 | low | One archetype source for act2plan and classtraits | cls1 and cls2 add agreement tests per class (V cards); the single source is OPEN |

## Tracked: fixed or in flight

| finding | sev | status |
|---|---|---|
| 01 F1, F2, F3, F4, F5, F9, F12 | blocker to low | fixed-by fix1 (S review 10 table); residues N1, N2, N3, N5 in-flight agent/fix1b; sale reservation, merc journal, two-hander shield in-flight agent/vault2 |
| 01 F6 | high | not closed (S); see N8 above |
| 02 #1 to #7, #10 | high, medium | fixed-by fix2 (S review 10 table) |
| 02 #3 residue (10 B2) | medium | in-flight agent/fix1b |
| 03 #1, #2, #3, #4, #6 | high to medium | fixed-by fix3; residues 11 N4, N5, N6 in-flight agent/fix7b |
| 04 #1, #3, #4, #5, #7 | blocker to medium | fixed-by fix4; residues 12 C1 in-flight agent/ops2, C2 to C4 above |
| 04 #2 | high | fixed-by fix4 as a procedure doc (`docs/ops/docker-volumes.md`); the practice is ops |
| 04 #8, #12 | medium, low | ops (board, mergeq env file) |
| 05 1.1 | blocker | ops, needs-live: restart the watcher on current code (led1 criterion 6 says the same) |
| 05 1.5, improvement 7; 09 redesign 4 | high | in-flight agent/led1 |
| 05 2.3, 2.4, improvements 9, 10 | medium | autoflip TASK-1092 and darkthrottle TASK-1091 on main, in-flight agent/grad1; the CI refusal of a default flip without an evidence record is OPEN |
| 05 section 3; 09 sections 5 and 6 (rest) | high to medium | ops (merge queue, autodeploy, card resolver); 09 redesign 3, 5, 7 are needs-owner |
| 06 links 1 to 6, fix 1, 2, 3, 5 (part) | high | fixed-by fix6; the smoke step in the gate (12 D1) in-flight agent/ops2 |
| 06 fix 4 | medium | ops (nightly timer), needs-live |
| 06 E1, E10 | high, medium | pgearned TASK-756 (`pre.from_save`); needs-live |
| 06 E4, fix 8 | high | needs-fork (write count from the game side) |
| 06 E5, E6 | high | fixed-by fix6 in part (binary hash, BLOCKED visible); "a stalled clock is RED" is OPEN |
| 07 A1, A2, A3, A4 (heartbeat) | blocker, high | fixed-by fix7; residues 11 N1, N2, N3 in-flight agent/fix7b |
| 07 A8 arbiter-starved | medium | in-flight agent/evid1 and agent/live1 (both touch it: merge order matters) |
| 08 fix 1, 3, 4, 5, 6, 8, 9, 10 | high to low | fixed-by fix8; residues 12 B1, B2 in-flight agent/fix8b; class rows in-flight agent/cls1, agent/cls2 |
| 12 A1, A2, A3, A4, E1, E2, low 1 to 4, 11 | high, medium, low | ops (autodeploy, mergeq union); the night log says A1 and A3 were done in ops2 (S) |
| 13 L1, L2, L3 (bot side), L4, L5, L6 | high, medium | in-flight agent/live1 |
| 13 L3 nav | high | needs-live: the recorded Tower Cellar 3 grid (live1 criterion 4) |
| 13 L4 vendor pays an owner-only reward | medium | needs-owner (live1 criterion 6) |
| 13 L7, L8 | medium, low | classified by live1 as owner designs, no change |

## Notes for whoever picks these up

- Two branches change the same rule: evid1 and live1 both quiet `arbiter-starved` for the leveling filler (V both card results).
- The fix4 tests asserted that a foreign look deletes the go; review 12 C3 called that the defect. If the owner wants the delete back, it belongs in `Take`, never in the polled `Granted`.
- Not swept: the ops repo itself (reviews 5, 9 and 12 sections 1 and 4 are classified from the review text and the night log only, U).
