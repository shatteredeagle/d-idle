# Whyline: why the hero does what it does (TASK-206)

> status: built and tested offline (fixtures, no game). V = proven by a test named here, U = not run live. Card text: `docs/research/incremental-arpg.md` (takeaway 2 "visible reasoning", card whyline, overlapping declinereason).

## What the watcher sees

One plain line under the video card and above the tabs of `/watch`: "Fighting in Blood Moor, my life is 80%." It is read from `GET /api/whyline` (viewer side, GET only, V 405 on POST):

```
{"schema":"whyline/1","code":"fight","text":"...","since":7,"history":[{"t":7,"code":"fight","text":"..."}, ...]}
```

`history` is the last 5 reasons, newest first, times in seconds on the hub clock. The Chronicle detail (`/api/chronicle`, field `why`, only while the feed is unlocked) carries the same history and the Chronicle tab lists it as "Recent reasons". Golden: `internal/watch/testdata/whyline-{sorceress,paladin}.json` (V).

## Where the reason comes from

Not from the page and not invented in the watch layer. The bot already decides at a handful of places; there it calls `Runner.setWhy(code, key, value...)` (`internal/bot/whyline.go`) right before the `r.do` or `setState` that starts the activity. The runner keeps it with the activity and returns it as `goals.Status.Why` (code plus string parameters, `internal/goals/runner.go`); a new activity without a reason clears it (no stale reasons). `GoalsOrders.OnStatus` hands it to `Hub.Whyline().Observe`.

| Code | Set at (internal/bot) | Parameters |
|---|---|---|
| `quest_step` | `taskStep` (campaign step title) | step, level |
| `walk` | `travel` | area, level |
| `fight` | `sweep` | area, life |
| `town_trip`, `town_chores` | `advanceTrip` (the chore planner's own reason text, numbers inside) | why |
| `heal_trip` | the low-life gate | life, line (the resume line) |
| `wait_heal` | `lowLifeWait` | why |
| `claim_trip`, `back_to_field`, `to_town`, `area_done` | trip and rest paths | none |
| `guard_exit` | route guard | why |
| `farm_runs`, `farm_gold` | farm and broke recovery | area, level |
| `saving_for` | `sweep` while poor (`resource.SavingFor`: portal scroll, then healing potions, then a requested repair) | area, item, need |
| `after_death_restock`, `after_death_corpse`, `after_death_gold` | death recovery | none |
| `player_items` | `playerActionStep`, `errandTripStep` (the item actions queued on the inventory page, docs/inventory-actions.md; what = identify, equip, trash, buyid, cain, stash or errand) | what |
| `hold` (watch only) | an open question and a bot that is idle or paused, with no reason of its own | question |
| `activity` (watch only) | the bot gave no code: the activity text as is ("Now: ...") | activity |

`TestWhyCodesMatchTemplates` keeps the bot's list equal to the template table (V).

## Templates (data)

`internal/watch/whyline.json`: per code the parameter names and exactly two phrasings (V `TestWhyTemplates`); the phrasing alternates with each new line, deterministic. An unknown code or a missing parameter shows `unknown reason code X` / `reason code X lacks parameter P` when `Whyline.Debug` is set and the neutral line "Doing what the plan says." in production (V `TestWhyUnknownCode`); the session counts it as `unknown`. Parameters are plain text, 80 characters at most (V `TestWhyParamsArePlain`).

## Rate limit and hold

The bot's reason is read at most once a second; a shown line stays at least 3 seconds unless the goal id changes (V `TestWhyRateLimitAndHold`). The same code with a new number (life 90 to 50) replaces the text after the hold but does not add a history entry. History and counts grow only when the code changes.

## Rules kept

- Spoilers (TASK-195 principle): the line names only the area the hero is already walking to or fighting in (the activity text already did) and the active quest step title. No list of upcoming zones. U: a step title can name an unvisited zone the instant the step starts; the area reveal rule of TASK-195 is not built yet, hook the step title into it when it lands.
- Voice: the whyline is text, not the hero's voice; muting the voice does not hide it, and it never speaks.
- Unlock ladder: visible from the start, no new row. Policy sheet: no row.
- Class generic: no class or skill name in code; the table has none (V golden for a Sorceress and a Paladin, `TestStatusCarriesTheReason` on the bot rig).
- Session record: `summary.json` gets `why_chain` (code to count), never the text; nothing is written per line, so a named run (`D2BOT_RUN`) pays no output cost (V field is omitted when the source is nil).

## Not done

- declinereason (a journal line per declined order) is a separate card; the codes here are the shared vocabulary.
- Stance numbers: `heal_trip` shows the bot's resume line (engage life), not the stance drink line; a stance-aware "drink at 50%" needs the combat knob passed as a parameter (U).
- Not seen in a real browser or a live run.
