# Graduations, 2026-11-28 (card grad1, TASK-1112)

> status: current. Policy (user, 2026-11-27): a feature that ran clean on a sandbox graduates: its behaviour becomes the code default, the knob is deleted (the key stays loadable through `env.RetiredKnobs`) and the ledger row closes. Tags: V = read or run today (how is named), S = secondary (another doc), I = inferred, U = unverified. No game was driven and nothing was written to a sandbox; the sandbox bot logs were read read-only.

Rule used per row: GRADUATE only when the row's `enable_criteria` text (`data/darklaunch/<knob>.json`) is met by cited evidence, or the owner is the coordinator, a sandbox window exists, the feature can be shown to have RUN in it, and no open review finding says otherwise (docs/design/fable-review-2026-11-28/). Review finding 06 F2 (high) applies to every row below: "no hard invariant counter rose in the window" is absence of a breach, not presence of the feature. So each row was checked for presence in tonight's logs.

## 1. Evidence read today (V)

Both sandbox bot logs (`bot.log.1` + `bot.log`, 2026-11-27 22:59Z to 2026-11-28 05:31Z, about 6.5 h, one Paladin each; the first sandbox carries the 2026-11-27 bundle and arbiter control, the second the quest board knobs), counted with `grep` inside the container and piped into `go run ./cmd/d2arbiter [-follow] /dev/stdin`:

| what | first sandbox | second sandbox |
|---|---|---|
| `level=ERROR` lines | 0 | 0 |
| new deaths (`death report` total and `lastHour`) | 0 (total stays 1) | 0 (total stays 24) |
| `tick` lines; `d2arbiter` and `-follow` | 838; exit 0, panics=0 | 902; exit 0, panics=0 |
| `zone choice` lines | 189 (Tamoe Highland, margin 1.5 to 6.0) | 220 (Dark Wood, Tamoe, Black Marsh, margin 1.5 to 2.6) |
| `arbiter shadow` lines; `invariant arbiter-starved` WARN | 29; 16 | 53; 15 |
| `delivery expect demoted` (mode 1) | 1727, all `skill=Zeal reason=mana` | 0 (knob not set there) |
| `sidestepping missile` | 0 | 0 |
| `checkpoint`, `checkpoint retreat` | 0 | 0 |
| `portal cast refused: ... mode=1` | 0 | 0 |

## 2. Verdicts

| row | verdict | criteria text | evidence |
|---|---|---|---|
| `bot.tickLogSec` | **GRADUATE** | one night on both sandboxes; `d2arbiter -follow` replays each night log without a panic or a parse error | V section 1: both logs replay, exit 0, panics=0, no `bad time`; on both since 2026-11-27 06:01Z (S ledger), no revert. Log only. |
| `bot.zoneModel` | NOT YET | A/B harness passes | The criteria have a cited FAIL: Sorceress 8 fresh seeds, XP ratio 0.497 (V in docs/probes/survival-holdout5-2026-11-21.md section 5: "do not flip"; cause I there: merc left dead and `k` stuck at the 3.00 clamp), and docs/probes/dark-queue-final-2026-11-27.md lists that fix as NEEDS CODE. Both sandboxes are Paladins, so tonight's clean 6.5 h (V section 1) and the audit's 5.8k to 18.9k xp/h say nothing about the caster. The default serves every class (AGENTS.md, build flexibility: a rule that only passes for one build is not done). The review 3 `bandDrop` finding is fixed (fix3, TASK-1094) and the unreachable-hole problem too (TASK-1088), both offline only. |
| `quests.stepExec` | NOT YET | four scenarios green 2 of 3 each, then a sandbox watch | S docs/probes/lab-night-results-2026-11-28.md: `drill.act2.duriel.stepexec` 0 of 3 (BLOCKED), `drill.act2.cube.stepexec` and `.shaft.stepexec` RED, `.summoner.stepexec` BLOCKED; "NOT met". Review 06 F2 names the row: enabled on an Act 1 hero that cannot reach two of the four ported steps. Graduating also means deleting the old handlers. |
| `bot.arbiterShadow` | NOT YET (closest call) | outcome rule on replays and incident scenarios (fewer monitor hits, no new ones); first family on a sandbox; owner go per family | Runs and is harmless: V 29 and 53 shadow lines, panics=0; `arbiter-replay -strict` green (S dark-queue-status). Against: S lab night single-seed arm "12 against 16 hits, two new kinds: NOT met"; and with the shadow always on every hero logs the soft WARN `invariant arbiter-starved` (V 16 and 15 in 6.5 h; "a standing fact, not a fault", audit 2026-11-27 problem 7, card `starvedquiet` not built, U). The text is mostly the control rows' criteria, which this card must not graduate. |
| `combat.deliveryExpect` | NOT YET | mode 1 failure log reviewed; then scenario or drill green | Mode 1 runs (V 1727 demotions in 6.5 h, every one Zeal for mana, back-off 20 s doubling) but nobody has reviewed that log (U), and it reads like a finding: at mode 2 the Paladin's main attack would be skipped most of the time. The knob cannot be deleted while mode 2 exists; graduation here would mean Min 1. |
| `combat.missileDodge`, `combat.threatDodge` | NOT YET | live drill: dodge succeeds against fire wall, lightning, hydra; no flail | Not exercised: V 0 `sidestepping missile` lines in 6.5 h (an Act 1 Paladin meets none of those missiles). The 45 min window that enabled them is absence of a breach only. |
| `combat.standOK` | NOT YET | live drill with a standing Fallen | The rule writes no log line, so presence cannot be shown (U). No drill cited. |
| `bot.portalGuard` | NOT YET | drill R3 green | V 0 `portal cast refused ... mode=1` lines: the guard never blocked a cast in the window, so the new rule was not seen to decide anything. R3 is not cited as run (U). |
| `bot.checkpointEnable` | NOT YET | live drill green (cast healthy, retreat at low life, return after death) | Not exercised: V 0 `checkpoint` lines in 6.5 h on the sandbox that has it. Card ckptlive (TASK-990) holds the drill. |

Not considered, by the brief: `bot.admission`, `quests.boardSM`, `agency.planner` mode 2, the arbiter control rows, owner rows.

## 3. What each NOT YET row waits for

| row | waits for |
|---|---|
| `bot.zoneModel` | the merc revive and `k` clamp card of holdout 5, then a Sorceress screen that passes (3 seeds, then the gate); or an owner decision to graduate per class, which needs the default to be class data, not one number |
| `quests.stepExec` | card act2night (TASK-1107) and summonerstall (TASK-1104), then the four scenarios 2 of 3 green |
| `bot.arbiterShadow` | card `starvedquiet` (no WARN while the starver makes progress) and one lab rerun of the outcome rule with no new kind; then it is a 30 line change (`ArbiterShadow` tests in `shadow.go`, `arbiter_control.go`) |
| `combat.deliveryExpect` | a review of the mode 1 log (why Zeal is demoted for mana on every fight); then Min 1, the knob stays for mode 2 |
| `combat.missileDodge`, `combat.threatDodge` | a scenario or sandbox hero that meets a dodgeable missile (Act 2 or later, or a lab drill) with sidestep lines and no death |
| `combat.standOK` | a log line or counter for the rule, then one drill with a standing Fallen |
| `bot.portalGuard` | drill R3 (docs/fixportalmelee.md), or a window in which a `mode=1` refusal is logged and the hero lives |
| `bot.checkpointEnable` | ckptlive (TASK-990): a window with a `checkpoint` cast and a retreat |

## 4. Graduated: `bot.tickLogSec`

- The tick line is always written, at most one a second (`tickLogGap`, `internal/bot/ticklog.go`); the field, the knob row and the off path are gone.
- `bot.tickLogSec` is in `env.RetiredKnobs`: a `tuning.json` that still names it loads and changes nothing (`TestRetiredTickLogSecKeyLoads`), so no sandbox file needs an edit.
- Ledger row `data/darklaunch/bot.tickLogSec.json` set `default-on` (graduated 2026-11-28, in its note); docs/knobs.md regenerated; docs/design/ticklog-t4.md carries the night's numbers.
- Found on the way: the procedure's own count rule is not met. About 23,400 s of log gave 838 and 902 ticks (one per 26 to 28 s, about 130 lines an hour, not 3,600). I: the line is written only when the goal chain runs, not while an action is in flight. U whether the session scoring needs the denser record; worth one look before the arbiter verdict leans on tick counts.
- Not verified live after the change (the sandboxes already run the same behaviour through the knob).

## 5. Two findings for the coordinator

1. The six rows of the 2026-11-27 bundle are `enabled` in the ledger, yet four of them (both dodges, checkpoint, portal guard) did nothing observable in 6.5 h. Their rows should say "not exercised" until a drill or a hero in a later act shows them.
2. `delivery expect demoted` 1727 times for Zeal on mana is either a real mana problem of the Paladin or a wrong expectation; either way it blocks mode 2.
