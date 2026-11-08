# Decision dataset (schema v1)

> status: DESIGN (user decision 2026-11-07). Implemented by the session record (TASK-106, acceptance criteria 4 and 5); consumed by the learned-brain experiment (TASK-126). Tags: V verified, S secondary, U unverified. Nothing here is built yet; every number is U.

## Why

Every time a decision point of the store (`internal/decide`, `docs/t0-slice.md`) fires, the person's answer (or the lack of one) is a labelled example of how he wants the bot to play. Those answers are scarce and cannot be regenerated, so they are kept from the first session on, whether or not a model is ever trained. Uses: a small model of the player (TASK-126), tuning the rule defaults, and feedback from friends.

## File and format

- One append-only JSONL file per session: `<data volume>/sessions/<session id>/decisions.jsonl`, written by the same command run as the session record. Never rewritten; a line is flushed when written.
- Line 1 of the file is a header `{"v":1,"kind":"header",...}` (schema version, session id, bot commit, catalogue version, start wall time). Every later line carries `"v":1`.
- Two line kinds: `"kind":"fire"` (the firing) and `"kind":"outcome"` (follow-up, references the firing by `id`, written N minutes later; the firing line is not edited).

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

## Size estimate (U)

A `fire` line is about 300 to 500 bytes. The store deduplicates by key, so firings per hour are in the tens to a few hundred (the session record's questions-per-10-minutes number will say). At 300 firings per hour: roughly 100 to 150 KB per hour, plus outcome lines of about 100 bytes each (30 KB). A year of daily 2-hour sessions stays under 100 MB. TASK-106 replaces this estimate with a measured one.

## Versioning

`v` is bumped when a field changes meaning or is removed; adding a field keeps `v`. A reader skips unknown fields and refuses a higher major `v`. Catalogue key changes are covered by `decide.MigrateKey`; the header carries the catalogue version so old files stay readable.
