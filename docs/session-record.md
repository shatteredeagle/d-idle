# Session record and decision dataset

> status: BUILT offline, fixtures only (TASK-106). Tags: V verified live, S fixture or code, U unverified. Nothing here has run against a live game yet; the live smoke is the checklist at the end.

One play session (a run of the bot with a person present) writes four files under `<instance state>/sessions/<id>/`, where `<id>` is the UTC start plus the store profile (`20261108T100000Z-human`). Directory mode 0700, files 0600. Nothing talks to a network.

| file | what | written |
|---|---|---|
| `session.jsonl` | the human-facing record, append-only | as it happens |
| `decisions.jsonl` | the decision dataset (schema v1, `docs/research/decision-dataset.md`) | as it happens |
| `summary.json`, `summary.md` | the metrics of the direction review, section 5 | at the stop |

Code: `internal/watch/sessionrec.go` (recorder), `sessionexport.go` (export), hooks `decide.Store.SetAskObserver` (every Ask, including rule answers, which no event shows) and the store's events and log. `cmd/d2bot/play.go` wires it to the bot (S).

## session.jsonl (v1)

Every line: `v`, `kind`, `t_s` (seconds since the start, wall), `t_wall` (UTC). Kinds: `start` (session, commit, class, build, catalogue; class and build are empty when the character is not read yet), `identity` (class and build, written once they are known and again if they change; TASK-178), `question` (thought opened: `thought`, `point`, `fire`), `answer` (`by` player, `latency_ms`), `fade` (`by` timeout), `rule` / `revoke` (key), `order` (`order_kind`, area, minutes), `notes` (the return screen's box, scrubbed of the access token, at most 2000 characters), `viewers` (open true/false: the video stream has a viewer), `death`, `sample` (every 5 minutes: life, XP and credits gained), `stop`.

## summary.json

`questions`, `answered`, `faded`, `rules_learned`, `rules_revoked`, `questions_per_10min` (index = ten-minute bucket of the GAME clock), `fires_by` (rule/default/player), `fires_logged`, `fires_suppressed`, `answer_latency_median_ms`, `orders`, `mean_sec_between_orders`, `stayed_in_tab_min`, `deaths`, `xp_per_h` (per game hour) and `xp_per_wall_h`, `credits`, `notes`, `first_10min` (timeline, at most 40 entries), `record_bytes`, `decision_bytes`, `decision_bytes_per_h` (per game hour), `decision_bytes_per_wall_h`, `decision_bytes_by_kind` (fire, outcome, answer, header). The two clocks: `duration_s` is wall, `game_duration_s` is the injected game clock and `game_per_wall` their ratio (about 20 under lockstep, 1 in realtime). Rates and the outcome windows use the game clock; the wall figures are given beside them (V `TestSummaryBothClocksLockstep`).

Dataset size (V from the paladinsmoke lockstep run, docs/paladinsmoke.md): 978 KB for 707 virtual minutes = 83 KB per game hour and about 1.6 MB per wall hour; 2035 logged firings. A fire line is about 370 bytes and its outcome line about 95 (V on the golden session), so fire and outcome lines are about 98 percent of the file (2035 x 465 = 946 KB) and answers are noise. The 30 KB figure assumed about 65 logged firings per hour; the Paladin run logged 170 per game hour because 60-second repeats of many distinct keys are all kept (25,822 repeats were suppressed). Nothing was reduced: `decision_bytes_by_kind` now reports the split for every session. Expect tens to a hundred KB per game hour, not 30.

Limits (U): `stayed_in_tab_min` counts the minutes the video stream had a viewer (the watch hub has no other connection count), so a tab without video does not count and nothing is counted when video is off. `build` is the leveler build id chosen at the class-aware start (`planBuild` in cmd/d2bot/class.go; empty for a class without one or before that start; U live), the class comes from the goal status. `xp_per_h` and `credits` come from the goal status and the d.idle ledger (entries added during the session).

## The play-session command

```
d2bot play [normal d2bot flags]        # e.g. d2bot play --backend d2d --managed ...
d2bot sessions export [-o FILE] [-name N]... SESSION_DIR
```

`play` sets the environment twins that the tester container sets (`D2BOT_LADDER`, `_WATCH`, `_WATCH_VIDEO`, viewer 0.0.0.0:8891, answers 0.0.0.0:8892, the page origin of this machine's LAN address, `D2BOT_LADDER_STORE=shared`, token file `<data>/watch.token`, 0600, created when missing) unless you set them yourself, prints `http://<lan ip>:8891/watch?t=<token>` once at start, and records the session; stop with Ctrl-C or SIGTERM and the summary is written. Every other flag is a normal d2bot flag; one way per option, a flag given twice keeps the last. With the d2d backend set `D2BOT_WATCH_VIDEO_SOURCE=tap` (as the container does); the default `x11` needs a private-display instance. The token is printed by `play` only, never by the log; the recorder never receives it except as a string to remove from free text (S: tests read every file for it). A supervised child does not print it again.

## Export (opt-in)

`sessions export` reads `decisions.jsonl` and writes an anonymised copy to stdout or `-o FILE` (never overwrites): the session id is re-randomised, wall time becomes `t_off_ms` from the start, only the schema's fields are kept, and any string that contains the OS user name, the host name, the profile of the directory name, a `-name`, a `/`, `\` or `@`, or looks like a token (24 or more id characters; a commit hash is exempt) becomes `scrubbed`. It prints how many values it replaced. Read the file before sending it; the command never sends anything. A dataset with a higher `v` than this build is refused.

## Realtime (TASK-33, G6)

The recorder uses the injected game clock only for `t_game` and answer latency, and the wall clock for everything else, so realtime and lockstep both work. Under realtime the game clock is the wall (`docs/tester-realtime.md` rows 5 and 6); the lockstep-only places listed there are not touched by the recorder (S).

## Built vs verified

- S (tests, no game): record and summary on a scripted 18-minute session with a fake wall clock and the real store, two classes (`go test ./internal/watch -run 'Session|Dataset|Export'`); golden files `internal/watch/testdata/session-*.golden`; `d2bot play` wiring through `startLadder` with both classes and the token check (`go test ./cmd/d2bot -run 'Play|SessionsExport'`).
- U, live smoke checklist (needs a hidden slot): `D2BOT_WATCH_VIDEO_SOURCE=tap d2bot play --backend d2d --managed ...` for 10 minutes with the page open on a phone; then check that `<state>/sessions/<id>/` holds the four files, `questions_per_10min` is plausible, `decision_bytes_by_kind` shows fire and outcome as the bulk, `grep` of the token over the directory is empty, and `d2bot sessions export` on it passes a read-through. Not run in this task.
