# Decision dataset (schema v1)

> status: BUILT offline (TASK-106: `internal/watch/sessionrec.go`, `sessionexport.go`; how to use: `docs/session-record.md`). Not yet run live. Consumed by the learned-brain experiment (TASK-126). Tags: V verified, S secondary (fixture), U unverified.

## Why

Every time a decision point of the store (`internal/decide`, `docs/t0-slice.md`) fires, the person's answer (or the lack of one) is a labelled example of how he wants the bot to play. Those answers are scarce and cannot be regenerated, so they are kept from the first session on, whether or not a model is ever trained. Uses: a small model of the player (TASK-126), tuning the rule defaults, and feedback from friends.

## File and format

- One append-only JSONL file per session: `<data volume>/sessions/<session id>/decisions.jsonl`, written by the same command run as the session record. Never rewritten; a line is flushed when written.
- Line 1 of the file is a header `{"v":1,"kind":"header",...}` (schema version, session id, bot commit, catalogue version, start wall time). Every later line carries `"v":1`.
- Four line kinds: `header`, `fire` (the firing), `answer` (what the player did with the thought a firing opened: `of` = fire id, `by` `player` with `latency_ms`, or `timeout` when it faded) and `outcome` (follow-up, `of` = fire id, written N minutes later). A fire line is never edited, so `by` on it says who decided the action taken NOW (`rule`, `player` for a remembered answer inside the repeat cooldown, `default`), and the `answer` line says what the person said afterwards. Readers skip unknown kinds.
- A repeat of the same key and source inside 60 game seconds is not logged; the summary counts it (`fires_suppressed`).

## Fields of a `fire` line

| field | meaning |
|---|---|
| `id` | counter within the session |
| `session` | stable random session id (no account or character name) |
| `commit` | bot commit hash |
| `t_game` | game time (frames or ms, as the clock of the run counts) |
| `t_wall` | wall-clock time, UTC, RFC 3339 |
| `point` | point key, e.g. `t0.unique.fight` |
| `ctx` | the situation features exactly as the catalogue key carries them (banded): e.g. `areaclass`, `merc`, `kit`, `clvl`, `life`, `gold`, per the point's `contextKeys` |
| `class`, `build` | class and build ids from the plan table (data, not a character name) |
| `options` | the answers offered |
| `default` | the answer taken if nobody replies (the legacy answer via `AskWithDefault`) |
| `answer` | the answer used |
| `by` | `player`, `default`, `timeout` or `rule` (a learned rule answered) |
| `latency_ms` | time from the question shown to the answer (null when not asked) |

An `outcome` line: `{"v":1,"kind":"outcome","of":<id>,"window_min":N,"life_pct_min":..,"xp_per_h":..,"deaths":..}`; N is a knob of the recorder (U: start with 10).

## Privacy

- Stays on the machine of the person who plays. Nothing is uploaded by default and nothing talks to a network.
- No secrets, no account names, no character names, no host names or paths; the session id is random. The leakcheck scan covers the docs, not the data; the recorder must not write free text from the game.
- Export for friends' feedback is opt-in: an explicit command writes an anonymised copy (session ids re-randomised, wall time reduced to an offset) to a file the person then chooses to send.

## Size (S: fixture, replaces the estimate)

`TestDatasetSizePerHour` simulates one hour with a question every 20 seconds (the busy end) and a quarter of them answered: 63 logged firings, about 30 KB of `decisions.jsonl` per hour including header, answer and outcome lines (about 480 bytes per firing), plus about 3 KB of session record. A year of daily 2-hour sessions stays under 25 MB. A live number (V) replaces this once a session has run; each `summary.json` carries `decision_bytes_per_h` for it.

## Versioning

`v` is bumped when a field changes meaning or is removed; adding a field keeps `v`. A reader skips unknown fields and refuses a higher major `v`. Catalogue key changes are covered by `decide.MigrateKey`; the header carries the catalogue version so old files stay readable.
