# Run invariants and holdout seeds (card gen-invariants, 2026-10-31)

> status: current. Code: `internal/d2dsoakstat/invariants.go`, `invreport.go`; spec: soak-gates (Per-seed invariants, Death and exit invariants, Invariants verdict, Holdout seeds). Why: docs/research/bot-generality.md section 3 (invariants versus competence). Raw validation: docs/probes/geninv/validation.txt.

"The run is sound" is checked apart from "the run is good". `d2dsoak run|extract|compare` read each seed's bot log in one streaming pass (200 MB takes seconds) and print counters and a verdict per seed, per group and for the set. The fidelity verdict and the exit codes are untouched: the invariants verdict is an extra block before the OVERALL line and a field `invariants` in `-json` and `verdict.txt`. A script that wants soundness reads `INVARIANTS:` or the JSON, not the exit code.

Clock: the minute lines (one per virtual minute on d2d). Events are placed between them by wall timestamp. Tags: M measured on logs, D derived from a spec or knob (a test pins `StallWarnMin` to `bot.stallWarnMin`), J judged.

| invariant | counter and definition | violated when | provenance | how it fails |
|---|---|---|---|---|
| HANG | pace intervals with wall time but no virtual advance; `combat no progress, holding` lines | any | D (the loop and combat watchdogs exist to stop it) | lockstep stops stepping, or a fight livelocks |
| STALL | `campaign stalled: no progress` warns and `campaign stalled, exiting` errors | any | D (bot-loop ladder) | the ladder fired: the run had been stuck 30 min |
| WEDGE | `no route from level` and `no eligible reachable leveling area` lines; longest span of one storm (pause under 5 min, no non-town Entered) | span over 15 min | M: honest waits 3 min in 10 logs (8 m2early, navsoak and navsoak2 before the wedge); wedges 4689 and 18293 | the town-exit wedge of navsoak |
| PROGRESS | longest stretch with no kill or xp gain, non-town area, quest done, gold spent, clvl | over 30 min | D (same window as the ladder's first rung); honest max 12 min | any silent stall, also with a moving clock |
| DEATHS | deaths per 1000 kills (from 150 kills), 3 deaths in 30 min with no clvl gain or 5 in 60, deaths with no progress within 10 min (log long enough) | over 15 per 1000; streak; any unrecovered | D (death-rule FAIL-1, FAIL-3), J (10 min) | death spiral, or the bot stays dead |
| RESTARTS | `d2bot start` beyond the first, supervisor launches beyond the first. Reported only: fresh games (minute `restarts`), chicken exits | any bot restart | D | crash loop. Fresh games and chickens are normal play |
| EXIT | how it ended: limit, open (no marker, not judged), early, crash; recovered panics | early, crash, panic | D (soakfix 95 percent of the requested hours; supervisor or wrapper exit code) | the run stopped short |

Verdict per seed: PASS, VIOLATED(codes) or NO-DATA (under 2 minute lines). Set and group: VIOLATED when any seed is, PASS only when all are, else NO-DATA.

Holdout convention: seeds 16 to 23 are holdout. They are never used for tuning, fitting, diagnosis or picking a knob; a seed that shows a symptom is a tuning seed from then on and leaves the list. Tuning seeds are 1 to 8 (baselines-early). `-holdout LIST` overrides (default 16-23, empty disables). The report prints a `tuning` and a `holdout` group (hours, kills, deaths, XP/h, counters, verdict) and says `group holdout: none ran` when none did. Competence on holdout against tuning is the transfer gap; this card only reports it.

COVERAGE line: seeds (tuning, holdout), class and difficulty (from run.json: the start class index and Normal; `unknown` for plain logs), act (from the area ids of the minute lines), engine (d2d when `d2d backend up` is logged), clvl range, virtual hours. The study's gen-claims card owns the README wording; this is the data.

Reading it: HANG or STALL means look at the log tail; WEDGE with PROGRESS is the town wedge signature; EXIT early on a seed with otherwise clean counters is a harness or lab problem. Known limits: a wedge that keeps killing or entering areas is invisible to WEDGE (PROGRESS covers silence only); minute lines are the clock, so a log without them is NO-DATA; unrecovered deaths are not judged in the last 10 minutes of a log.
