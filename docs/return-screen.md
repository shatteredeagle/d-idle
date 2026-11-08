# The return screen (TASK-127)

> status: first slice, built and tested on fixtures; not wired into `Serve`, not run live. Design: `docs/research/fun-ideas.md` 3.6, ADR 0018.

One server-rendered page, `GET /watch/return`, that says what happened while the player was away. Code: `internal/watch/returnscreen.go`; tests and goldens: `returnscreen_test.go`, `internal/watch/testdata/returnscreen/`.

## What it shows (top to bottom)

1. Headline: away time, class, level, area, expeditions, deaths.
2. The five moments, each only when it happened: closest call, named boss, best drop, level-up (with quests done), the stats line that moved.
3. Deaths with cause, drops worth noting (unique, set, rune, rare, crafted; the rank table is `dropRank`), areas entered.
4. Expedition cards, newest first: zones, length, kills, XP, potions, bosses, kept drops, deaths.
5. Registered sections placed before the missed questions (placeholder: "Waiting for your yes", card 131 proposezone).
6. Questions missed, grouped by decision point with the question text and up to three situations. Buttons: "answer once for all N still open" (one per answer of the point) and "make this a rule" (yes/no on the open confirms of the point).
7. Registered sections placed after (placeholder: "Tally", card 129 tallystrip; card 130 policy effects goes here too).
8. A box for three sentences about the session, posted to a `NotesSink` (the session record, TASK-106). Disabled with a message while no sink is connected.
9. "What this page cannot see yet": the honest list of missing events.

`GET /api/return` returns the same data as JSON (`watch.return/1`); `?hours=N` forces a window of N hours.

## Data sources

| Part | Source | Tag |
|---|---|---|
| Missed questions, closest call (fight questions print the life), journal-only deaths | `Hub.Journal` lines (`missed`, `question`, `incident`) | V (existing lines, tested on fixtures) |
| Kills, drops, deaths, level-ups, area changes, potions, quests | `AwayLog.OnGoalEvent(goals.Event)`; the kinds are `goals.Ev*` | V that the kinds exist (internal/goals/runner.go); U that the live bot fills `Monster`, `Boss`, `Item` and `Area` on every one |
| Expedition (town trip) boundary | an `area_change` whose `AreaID` is a town (`data.IsTown`) ends a trip; the next change out starts one | U until traced on a real soak: the rule is only exercised on a synthesized stream |
| Closest call, exact | an optional `close` event with `LifePct`; none is emitted today | U (no emitter) |
| Gold, supplies cost, waypoints lit | no event exists; the page prints "not reported" | gap, listed on the page |
| Absence | `Visit` (last look at the live screen; optional file) | V (tested with a fake clock) |

The fixture format is one `AwayEvent` JSON per line (`ReadAwayEvents`); a recorded soak log can be converted to it. No game was used.

## Landing and absence

`ReturnConfig.ReturnHandler(next)` wraps the live handler. `/watch` goes to `/watch/return` when the last look is older than `AwayAfter` (20 min); `/watch?live=1` (the link on the page) reaches the live screen and counts as a look. A first start with no earlier look is not a return.

## Questions: what the buttons really do

A faded question cannot be answered afterwards (the store refuses it, `ErrThoughtFaded`). So "answer once for all" acts on the questions of that point that are STILL OPEN (`Hub.AnswerKind`, over `Hub.Answer`); each answer counts like a tap, so the store's own streak and rule learning apply. "Make this a rule" is the yes/no on the open confirm thoughts of the point (`Hub.ConfirmKind`, over `Hub.Confirm`). With nothing open the page says the faded ones were counted only and the hero asks again. The decision-point store and the ladder are untouched.

## Extending: sections

```go
reg := watch.DefaultSections()
reg.Register("tally", watch.SectionFunc(func(d *watch.ReturnData) *watch.SectionView {
    return &watch.SectionView{Title: "Tally", After: true, Lines: []string{"+12 tally, next buy: charter"}}
}))
cfg.Sections = reg
```

`Register` with an existing id replaces it in place; returning nil hides the section; `After` puts it behind the missed questions. A section reads the finished `ReturnData` (trips, deaths, moments) and returns text lines; no HTML.

## Wiring (outside TASK-127's owns; needed to go live)

1. `cmd/d2bot` `feedHub`: build `away := hub.AwayLog()` once, call `away.OnGoalEvent(ev)` next to `o.OnGoalEvent(ev)`.
2. `Serve`: make `rc := watch.ReturnConfig{Hub: h, Events: away, Visit: watch.NewVisit(<state>/visit.json, nil), Speed: ...}`; mount `rc.ReturnHandler(h.ViewHandler())` on `watch.ReturnRoutes` and `/watch` (inside the existing view gate, so the token or passcode covers it); wrap the answer handler so `watch.ReturnPostRoutes` go to `rc.ReturnAnswerHandler(ansOpts)`.
3. Optional: `rc.Notes = <the session record>` once TASK-106 exists.

## Not verified

Live: not run (no game). Event fields on a real run (see U above), trip boundaries, the look of the page in a browser (the HTML is snapshot-tested and CSP-clean, not screenshotted), the buttons against a real answer listener (tested with the handler directly).
