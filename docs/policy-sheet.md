# The chores policy sheet (TASK-128)

> status: built and tested offline on fixtures; NOT run against a live bot or in a browser. Design: `docs/research/fun-ideas.md` 3.2, ADR 0018 item 4. Tags: V = proven by a test in this tree, S = from a doc, U = unverified.

One page, `GET /watch/policy` (JSON: `GET /api/policy`), lists the chores the hero does in town and on the way: repair, restock, town trips, a lost portal scroll, the scroll reserve, low-life retreat, potion mix and tier, belt fill, white items, selling junk, identify, a full stash. Each row shows today's setting, the default and every allowed value. The player may pick poor values on purpose (user, 2026-11-07: "should allow suboptimal choices ... since that is a player decision that they can refine and learn from by experimenting"): a poor option is flagged "your call" with one risk line and is never refused (V `TestPolicySetAppliesRuleAndSurvivesReload`).

## The rows are data

`internal/decide/policy_chores.json` (embedded, validated against the catalogue by `decide.ParsePolicy`, V `TestPolicyTableErrors`):

| field | meaning |
|---|---|
| `id`, `label`, `help` | the row |
| `classes` | catalogue classes the row is for; empty = all. A class the catalogue does not know is an error naming it (`PolicyTable.RowsFor`), never a fallback (V) |
| `default`, `options[]` | the default option must change nothing: it is today's behaviour |
| option `rules[]` | ladder rules `{point, when, answer}`; first match wins; an unknown point, field, value or answer is a load error |
| option `knobs` | tuning knob values set when the option is chosen (reset to the profile value when another option replaces it) |
| option `suboptimal` + `risk` | the flag and the one-line risk (required together) |
| option `applies` + `note` | `part` (some of it is wired) or `record` (stored, no bot path reads it yet), with the gap in `note` |
| row `effect[]` | `{counter, label, heuristic}` for the effect line |

Two rows may not decide the same point. Shipped rows: repair, restock, town, return, tp, tpreserve, retreat, potionmix, potiontier, belt, whites, sell, identify, stash. Their knob keys exist and their values are in range (V `TestPolicyKnobKeysAreRealKnobs`).

## How a change applies

`POST /api/policy {row, option}` on the answer listener (same gate as the return screen: origin list, token, JSON only, 4 KB, known fields; V `TestPolicyHTTPGates`). `PolicySheet.Set`:

1. Knob options first, through the tuner (`bot.Tuner.Edit`, all or nothing, saved in its `tuning.json`). No tuner connected: the option is refused with 501 and nothing is logged (V).
2. One log line of the decide store, `Store.SetPolicy` (op `policy`, row, option, rules). This is the single persistence: the store reloads and replays it with the answers (V `TestPolicyAnswersAndPersists`). A failed log line puts the knobs back.
3. `Store.ask` consults the sheet's rules before a learned rule and before any card: the answer is returned with source `policy`, no thought opens (V). Chore points that the bot asks today (`t1.repair`, `t1.restock`, `t1.town.trigger`, `t1.tp.lost`, `t0.retreat.drink`) therefore stop opening cards when a row covers them; two builds (V sorceress and paladin). On start `PolicySheet.Reapply` sets the knobs of the stored options again.

The default option is a log line with no rules, so "back to default" is one change like any other and the legacy answer acts again.

Old builds: a profile file with a `policy` line is not readable by a build without the op (it reports a corrupt profile); the file version is unchanged.

## Effect lines (sheet) and before/after cards (return screen)

Per row the sheet prints "Since you changed this to "X" (N ago): label: n, ..." computed from the away events since the change (`policyCounters` in `internal/watch/policysheet.go`; V on a fixture journal, `TestPolicyCounters`). Counters: potions (all, mana, healing, rejuvenation: told by the item name, S), deaths, `deaths_dry` (deaths with no potion drunk in the 120 s before, U: my window), town trips (an area change into a town, U as in the return screen), white items picked up, kills, close calls, gold gained, repairs. A counter no event carries prints "not reported" (V: repairs, and gold until a gold event exists). A counter whose link to the setting is a guess carries `(U)`: `deaths_dry`, `trips`, `whites`. The text only says what happened since, never that the setting caused it.

The return screen gets a section "Policy changes" (`PolicySection`, registered by `NewFun`): per change of the last 24 h (newest first, 5 at most) one neutral card, "Potion mix: now "Mana potions first" (was ...). After, 15 min: ... Before, 15 min: ...", with the two windows the same length (the shorter of the time since the change and the history before it; under a minute of history: "no comparison yet"). A "Change it back" button posts the old option to the same route while the row still has the new one (V golden `card-*.json`, `TestPolicySectionInReturnScreen`).

Changes are remembered in memory only (hub time base): after a restart the sheet shows "Set before this run: there are no events to compare yet" until the next change.

## Set at creation

The setup wizard's hero step now says chores start on defaults and can be changed on the Policy sheet (`cmd/d2dsetup/page/app.js`, one hint line; the wizard is otherwise untouched). The sheet itself needs the hero to run; there is no pre-run editor (U).

## Built vs verified

Built and tested offline: table and loader, store op and precedence, page and JSON, effect counters, return section, answer route and gates, wiring in `cmd/d2bot/funwire.go` (V `TestFeedHubFunLoop`: GET, POST, knob path through a fake tuner). NOT verified: a live bot reading the rules and knobs (the call sites are the T1/T0 ones of `docs/t0-slice.md`, themselves not drilled live), the page in a browser (`node --check`-level only for the script), the effect lines against a real soak, any causal reading of an effect.

## Gaps (what a bot or event card would add)

- Potion mix: solved by the belt plan knob (TASK-164, `docs/belt-plan.md`): the `potionmix` options now set `resource.beltPlan`, so the shop, the fill, the loot pickup and the kit gate follow the chosen columns. Potion tier (`potiontier`) is still a record.
- `t1.identify`, `t1.potions.buy`, `t1.stash.full`, `t1.keep.kind` are not asked by the bot yet (`docs/t0-slice.md`): their rows store the choice; identify acts through its scroll knobs only.
- `t0.retreat.drink` rules act only at the abort step and not under the stage-2 urgency key (U).
- No repair, purchase, or gold event exists, so repair counts and gold are "not reported".
- Belt layout columns and a TP scroll reserve exist only as the knobs `town.beltFillPct` and `town.tpReserve`.
- Rules panel: policy rules are shown on the sheet (`Store.PlainRule`), not yet as `source: policy` rows of `/api/rules`.
