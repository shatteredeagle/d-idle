# Policy preview (card policydryrun, TASK-205)

> status: built and tested offline on fixtures; NOT run against a live bot or in a browser. Source: `docs/research/incremental-arpg.md` proposal 1 (strictness presets plus a dry-run preview, "hide but log"). Tags: V = proven by a test in this tree, S = from a doc, U = unverified.

When the player picks another option on a row of the policy sheet (`docs/policy-sheet.md`), the sheet first shows what that option WOULD have done over the recent recorded events, with Save and Cancel. Nothing is saved, no knob moves and no rule changes until Save (V `TestPolicyPreviewStanceTwoClasses`: no log line, no knob call, option unchanged).

## The honesty rule

A preview is a counterfactual from a log. It reports a delta only where the recorded events carry what the delta needs; every other row says "no preview for this row yet: <reason>" and shows no number. Each line says "based on N events in the last X minutes" and a confidence: `low` under 10 events, `medium` under 30 or for any bound, `high` otherwise (a bound never reaches `high`). Under 3 events the answer is "not enough events" (V `TestPreviewNotEnoughAndWindow`). Bounds are named: "at least" when the log can only show part of the effect, "at most" when it can only show an upper limit. The preview never claims a death, a kill or a question count would change: the log cannot say what happens after a different decision.

## Which rows have a preview

Table: `internal/decide/preview_rows.json` (one entry per policy row; a row without an entry fails `TestPreviewNoPreviewRowsAndErrors`, so a new row must decide).

| row | preview | how each delta is computed | data it needs |
|---|---|---|---|
| `stance` | yes | Each fight's lowest life percent (event `close`, `lifePct`) against the old and new `combat.chickenPct` (leave) and `combat.drinkLifePct` (drink). Raising a line: fights whose low lies in (old, new] "would have crossed it" (retreat: would have ended in a retreat; drink: at least that many more fights with a potion). Lowering: fights whose low lies in (new, old] are an upper bound of retreats or drink-fights that would not have happened. Knob values per class from the option (`classKnobs`), else the profile default (`env.Knobs`). | `close` events with a life low |
| `whites` (pickup strictness) | yes | Recorded pickups (`drop` events with a quality). A stricter level lists the pickups of the qualities it no longer picks up (`hide` table in the JSON; runes and gems never), as an upper bound because the pickit may still keep a wanted item. A looser level shows no number: items left on the ground are not recorded. | `drop` events |
| `sell`, `repair`, `restock`, `town`, `return`, `tp`, `tpreserve`, `retreat`, `potionmix`, `potiontier`, `belt`, `identify`, `stash`, `questhints` | no | The reason is in the JSON and shown on the sheet (for example: no sell or gold event exists, the belt state at each visit is not recorded, hints only speak). | |

The `sell` row has no preview because no sell or gold event is recorded; the pickup list of `whites` is the nearest honest thing and its note says so.

U, the main gap: nothing feeds `close` events to the away log yet (`AwayLog.OnGoalEvent` drops that type; the fight telemetry line `fight ... minlife=` of `internal/combat/dodge.go` is the source a wiring card would use). Until then the live stance preview answers "not enough events"; the numbers above are proven on fixtures only. The preview is built so that wiring the emitter is all that is missing.

## Hide but log

Nothing a setting would hide is dropped silently. The pickups a stricter level would have left on the ground are listed under "Filtered out" with the reason for each (at most 50 shown, the total is always given, V `TestPreviewFilteredIsBounded`).

## Presets

The `stance` row is the only row with the three strictness levels, labelled Cautious, Balanced and Aggressive (already on the sheet, `docs/stance.md`). `questhints` also has three options (Quiet, Hints, Hints and ask me) but they are a chattiness axis, not a risk axis, so they keep their labels. No row was added.

## API

`POST /api/policy/preview` on the answer listener, body `{"row": "stance", "option": "cautious", "minutes": 60}` (`minutes` optional, default 60, at most 1440). Same gate as `POST /api/policy` (origin list, bearer token, JSON only, 4 KB, known fields; V `TestPolicyPreviewHTTPGates`). Read-only; any other verb is 405. Errors: 404 unknown row, 400 unknown option or a row not for the hero's class, 503 class not known. Reply schema `decide.policy.preview/1` (golden: `internal/watch/testdata/policysheet/preview-paladin.json`).

Code: `internal/decide/preview.go` (pure `PreviewSpecs.Preview`), `internal/watch/policypreview.go` (`PolicySheet.Preview`, handler), the page script in `internal/watch/policysheet.go` (click an option: preview card with Save and Cancel; if the preview route is unreachable the card says so and Save still works).

## Built vs verified

Built and tested offline: two classes (Paladin, Sorceress) with different stance numbers, known deltas from synthetic logs, empty history, unknown row, option and class, bounded list, a race test, the gates, golden JSON. NOT verified: the page in a browser (script parsed with `node` only), the preview on a live session (no emitter for `close` events yet), any causal reading.
