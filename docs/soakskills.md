# Soak report: per-skill fields (change soakskills)

Source: docs/research/sorc-act-gate.md section 2. Code: `internal/d2dsoakstat/skills.go` (`ParseSkills`, `SkillStats.Text`). Tests use fixture lines only (V offline, not verified live).

Audit result: before this change `d2dsoak` printed no per-skill field (only kill-time buckets, where "cast" is seconds, not a count). Message names were read from internal/combat (V by reading the source).

| Field | Printed | Log source |
|---|---|---|
| casts started per skill | yes | debug `combat decision act=cast skill=` (needs debug level) |
| buff casts, recasts after a cut | yes | `combat buff`, `combat buff cut by a hit, will retry` |
| casts per kill | yes, all skills together and per skill | casts / `killtime` lines (per-skill kills are not credited, so this is casts of that skill over all kills) |
| novas, monsters per nova | yes | `act=nova n=` |
| staff swings | yes, when the caller names the melee skill | casts of that skill |
| made effect, cut, mana spent, kills (last skill), armor seconds absent, build, cast length ticks | yes | see docs/logfields.md |
| hits per cast | n/a | not logged |

Log-side fields (change logfields, docs/logfields.md): `cast result`, mana per cast, the kill's last skill, an armor sample on the minute line, the `build` line and the cast length ticks are logged and printed (offline only). Still n/a: kills credited per skill by hit, hits per cast.

Hook: `d2dsoak report [-melee SKILLNAME] LOG...` (cmd/d2dsoak/report.go) runs `ParseSkills` per log and prints `Text(path, melee)`. `-melee` is the basic-attack skill name from the skill data (no numeric id); without it the swing line prints n/a. Covered by `TestReportSkills` (fixture log, V offline).
