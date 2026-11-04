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
| d2d branch and commit | only if a line carries `branch`/`commit` keys, else n/a | none today |
| casts that made a missile/effect | n/a | not logged |
| casts cut before the action frame | n/a (only buffs: recast count) | not logged for attacks |
| mana spent | n/a | not logged (the cast book exists in memory, `st.book`) |
| kills credited per skill, hits per cast | n/a | not logged |
| armor seconds absent | n/a | not sampled |
| Lightning cast length 13 or 19 ticks | n/a | not logged |

Log-side follow-ups: emit a `cast result` line (skill, made effect, cut), mana per cast, the kill's last skill, an armor-state sample on the minute line, a `build` line with branch and commit, and the Lightning cast ticks.

Hook: `d2dsoak report [-melee SKILLNAME] LOG...` (cmd/d2dsoak/report.go) runs `ParseSkills` per log and prints `Text(path, melee)`. `-melee` is the basic-attack skill name from the skill data (no numeric id); without it the swing line prints n/a. Covered by `TestReportSkills` (fixture log, V offline).
