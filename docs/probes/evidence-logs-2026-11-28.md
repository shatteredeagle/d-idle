# Evidence lines for the NOT YET knobs, 2026-11-28 (card evid1)

> status: current. Follows docs/probes/graduations-2026-11-28.md section 3. Tags: V = read or run today (how is named), S = another doc, I = inferred, U = unverified. No game was driven; the sandbox bot logs were read read-only (copied out with `docker exec ... cat`). Nothing here is verified live.

## 1. New log lines (V: unit tests, `go test -p 1 ./internal/combat ./internal/bot`)

A window can now tell "the rule never met its case" (no line) from "it met the case and did X" (a line). The scenario matcher prints the message, then the keys in alphabetical order.

| knob | line | written when | limit |
|---|---|---|---|
| `combat.missileDodge` | `combat evidence rule=missileDodge outcome=sidestep\|held\|no-room missiles=N` | knob on and a missile of a dangerous kind is within reach; `held` = evade not allowed, the per-fight cap or the 800 ms gap | one per outcome and 30 s |
| `combat.threatDodge` | `combat evidence rule=threatDodge outcome=dodge\|no-room\|kill-shooter missiles=N shooters=M` | knob on and the threat table ranked a hostile missile inside `combat.dodgeRadius` | same |
| `combat.standOK` | `combat evidence rule=standOK outcome=kept unit=.. engaged=.. approaches=..` | knob on and a stander (monstats AI `fallen`) passed `combat.reachSec` without a shot; knob off writes the old `never in range` drop | same |
| `bot.portalGuard` | `combat evidence rule=portalGuard outcome=allowed adjacent=N incoming=D dmgKnown=.. life=..% mode=1` | mode 1 let a town portal through that the survival rule asked for in a fight; the refusal line `portal cast refused: ... mode=1` is unchanged | same |
| `bot.checkpointEnable` | `checkpoint idle why="..." area=A life=.. scrolls=.. hostiles=..` | knob on, field, and the rule casts nothing; reasons in rule order: area not in the checkpoint table, life under the cast line, hostiles near, scroll reserve, checkpoint already set here, cast gap, far from the entry | one per area and reason and 10 min |

No new knob. None of the lines changes a decision (V: the existing tests of both packages pass unchanged). Nothing is build-specific: the lines sit in the shared fight and loop code; the portal and checkpoint tests run for two classes.

Finding (V, `internal/plandata/checkpoint_areas.go`): the sandbox hero with `bot.checkpointEnable` 1 farmed Tamoe Highland, which is not in the checkpoint table (dungeons and the Monastery only), so zero `checkpoint` lines in 6.5 h was the rule holding by design, not a broken rule. The idle line now says so.

## 2. Scenarios (`tools/provingground/scenarios/evidence.json`; V: `go run ./cmd/d2scenario validate`; U: none has run in the lab)

| id | setup | asserts |
|---|---|---|
| `evid.missile-dodge` | 3 Gloams (`willowisp1`) beside a clvl 30 hero, knob 1 | one `rule=missileDodge` line, no death |
| `evid.threat-dodge` | 3 Fallen Shamans and 3 Fallen, knob 1 | one `rule=threatDodge` line, no death |
| `evid.stand-ok`, `.control` | 6 Fallen, `combat.reachSec` 6, knob 1 against 0 | `rule=standOK` line and no `never in range` drop; control: no evidence line |
| `evid.portal-guard` | clvl 6 hero at 45 percent life, no potions, 12 Fallen, mode 1 | a `mode=1` refusal or a `rule=portalGuard` line, no death |
| `evid.checkpoint-cast` | field start in the Barracks with 5 scrolls, knob 1 | `checkpoint ... to=set`, no death |
| `evid.checkpoint-idle` | Blood Moor, knob 1 | `checkpoint idle ... why=area not in the checkpoint table`, no cast |

Each runs for Paladin and Sorceress, `pre.hero_profile` banked. Open points are in each scenario's notes (U): whether the Gloam bolt's missiles.txt name matches the `lightning` kind, whether a shaman fireball gets inside the dodge radius in the budget, whether a stander stays out of shot for 6 s, which of the two portal lines shows. A red run on the first three is a finding about the rule's reach, not about the scenario. Retreat and return after death of the checkpoint stay with card ckptlive (TASK-990); the cast time of the portal stays with drill R3.

Ready-to-run (lab, night window): `d2scenario run -id evid.missile-dodge` and the other six ids, three seeds each before any graduation.

## 3. `bot.zoneModel`: decision matrix per class

| class | evidence | verdict |
|---|---|---|
| Sorceress | 8 fresh seeds, XP ratio 0.497, interval under the 0.95 floor, chickens 6 against 1 (V docs/probes/survival-holdout5-2026-11-21.md section 2); 4 builder seeds ratio 0.615 to 0.669 (S holdout 4) | FAIL, do not flip |
| Paladin | 8 seeds PASS (S holdout 4 section 3); 2 fresh pairs ratio 1.364 but 2 chickens against 0, under the 3-pair minimum (V holdout 5 section 3); 6.5 h clean on two sandboxes, margin 1.5 to 6.0 (V graduations doc) | PASS on yield, chicken signal open: 3 more pairs before a flip |
| Necromancer, Amazon, Druid, Assassin, Barbarian | no A/B pair found in docs/probes (U) | no data |

So a single default cannot graduate: one class fails, one passes with an open signal, five are unmeasured. The failing cause is NOT isolated in data (I, holdout 5 section 4): the caster's first pick (Burial Grounds, model 150 to 160 xp/min against realized 4 to 11), `k` pinned at the `kMax` clamp on the three worst seeds, and the merc left dead 15 to 26 times a run all point the same way, but no experiment separates them. No zone row was changed here: a data edit without a pair run would be a guess. Next card (unchanged from holdout 5): merc revive under the model and a way down for `k`, then a 3-seed Sorceress screen.

## 4. `bot.arbiterShadow`: the `arbiter-starved` WARN

Cause (V, both sandbox logs, 2026-11-27 22:59Z to 2026-11-28 05:56Z): 32 of 32 WARN lines name the intent `level/sweep` ("nothing else wanted: level the hero"), 30 of them outranked by `quest/step`. That intent is the standing filler: its source proposes it on every decision (`internal/arbiter/sources.go`), so it loses whenever any objective exists and reaches `StarveSec` (600 s) by construction. It is a false positive of the monitor, not starvation.

Fix (V: `TestStandingFillerIsNotStarved`): the bot no longer raises the soft invariant for a filler without a TTL (`standingFiller`, `internal/bot/starvedcause.go`). The arbiter's own `Decision.Starved` and the `Lost` hook are unchanged, a temporary filler (`level/level_up`, TTL 900) and every other class still raise it. Not verified live.

Second observation (I, not fixed): with a board chunk parked (`countess gave_up`, `no_path`, `malus handin_failed`) the shadow still chooses `quest/step` for hours while the chain sweeps. The quest source backs off only on `FactQuestRefused`; a parked board run does not set it. That is a shadow/chain disagreement the outcome rule will count.

## 5. Tick lines: the count rule of docs/design/ticklog-t4.md step 4

V (same logs): 960 and 896 tick lines; gap between ticks p10 3 to 4 s, median 16 s, p90 63 to 66 s, max 181 and 418 s; 22 and 25 gaps over 120 s. V (code, `runGoalArb` in `internal/bot/shadow.go`): the line is written once after each pass of the goal chain returns, and a pass blocks for the whole action (a fight, a walk, a town trip). So the count is the number of chain passes, not seconds; "close to the night's seconds divided by the gap" can never hold.

Proposal for the owner (the criteria text is not edited here): replace the count rule by (a) ticks at least twice the `minute` lines of the same window (measured 2.2 to 2.3), and (b) no gap between two ticks longer than the longest action timeout. The replay scores decisions, and one tick per chain pass is one record per decision, so a denser record adds nothing to the verdict (I).
