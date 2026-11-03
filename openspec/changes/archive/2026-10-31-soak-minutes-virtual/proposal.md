## Why

User decision 2026-11-01 ("virtual minutes"): in the d2d gate ladder every minute of the soak-gates criteria is a virtual (game) minute. The d2d lockstep clock runs about 10 times faster than the wall clock and the rules protect the bot's in-game behaviour (potion supply, progress). Gate 1 soak 2 (docs/probes/postfix-soak2.md) therefore fails the belt window (27 virtual minutes) and the no-progress rule (26 virtual minutes without a kill or XP), while d2dsoak printed INVARIANTS PASS because its PROGRESS limit is 30 and counts wider events.

## What Changes

- delta spec soak-gates: ADDED "Gate 1 minutes are virtual minutes" with a scenario per criterion; MODIFIED the two Gate 1 requirements (belt, progression) to say virtual minutes; no number changes (5 and 10)
- d2dsoak (internal/d2dsoakstat): new counters and violation codes BELT (more than 5 consecutive minute lines outside town with no healing potion in pack or belt) and NOKILL (more than 10 consecutive minute lines with no new kill and no new XP), printed with the limit; the existing PROGRESS invariant (30, wider events) is unchanged
- docs: postfix-soak2 decision section, review-strategy status line, d2d-soak and geninv notes, decisions.md line
- not changed: any number in the spec, the m24 and Game.exe wall-clock monitors, specs/** (the delta is applied at archive, when the d2dsoak alignment is merged and live-proven)
