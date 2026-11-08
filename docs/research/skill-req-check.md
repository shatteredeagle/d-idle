# Skill requirement check (TASK-132 skillreqcheck)

> status: result note, 2026-11-08. No code changed. Source of the claims: `docs/research/fun-ideas.md` section 12 (the SME's schedule and the UNVERIFIED list) and `docs/research/class-early-game.md`.

## Method (V)

The table values were read from the worker host's real install tables (`/tmp/d2mpq`, the same install `internal/data/real_test.go` uses) through `data.DB.SkillByName(name)` (`ReqLevel`, `MaxLevel`, `ReqSkills`). A throwaway test file (not committed) printed each row:

    labrun orion -- sh -c 'go test -count=1 -run TestZZDump -v ./internal/data'

Caveat (U): whether that install carries a patched Skills.txt is not known to this note (`docs/buildv3.md` speaks of "patched Skills.txt"). The values below are the ones the bot's own `Rule` reads, so they are the ones that matter for the leveler; they match classic 1.14d as the author remembers it, except the Energy Shield prerequisites (see the end).

## Table: SME claim against the table (V)

The SME claims are the clvl values in the section 12 UNVERIFIED paragraph (the schedule rows themselves are events, not levels).

| Skill | SME claim (clvl) | Table ReqLevel | ReqSkills | Match |
|---|---|---|---|---|
| Frozen Armor | 6 | 1 | none | NO, wrong |
| Ice Bolt | 6 | 1 | none | NO, wrong |
| Warmth | 12 | 1 | none | NO, wrong |
| Inferno | 12 | 6 | none | NO, wrong |
| Ice Blast | 12 | 6 | Ice Bolt | NO, wrong |
| Fire Ball | 18 | 12 | Fire Bolt | NO, wrong |
| Nova | 18 | 12 | Static Field | NO, wrong |
| Lightning | 18 | 12 | Charged Bolt | NO, wrong (the repo's v3 note already had 12) |
| Fire Wall | 12 | 18 | Blaze | NO, wrong |
| Frost Nova | 6 | 6 | none | yes |
| Static Field | 6 | 6 | none | yes |
| Telekinesis | 6 | 6 | none | yes |
| Blaze | 12 | 12 | Inferno | yes |
| Glacial Spike | 18 | 18 | Ice Blast | yes |
| Teleport | 18 | 18 | Telekinesis | yes |
| Fire Bolt, Charged Bolt | 1 | 1 | none | yes |

Nine of the SME's level claims are wrong; all nine match the author's memory of classic 1.14d exactly. None of this changes the event schedule (events gate availability, levels gate the point), but any Codex text or unlock file must print the table values, not the SME's. Extra rows read for the repo: Chain Lightning 18 (Lightning), Lightning Mastery 30, Enchant 18 (Warmth, Fire Ball), Meteor 24 (Fire Ball, Fire Wall), Energy Shield 24 (Teleport, Chain Lightning).

## buildv3 against the table (V: read `internal/plandata/build-fallback-lightning-v3.json`)

- Rows: Charged Bolt clvl 2-5, Frozen Armor and Charged Bolt at 5, Frost Nova at 6, Charged Bolt 7-10, Telekinesis 11, Lightning 12, Charged Bolt 13-17, Teleport 18, Chain Lightning from 19. Every row's ReqLevel is at or below its clvl (Frost Nova 6 at clvl 6, Telekinesis 6 at 11, Lightning 12 at 12, Teleport 18 at 18, Chain Lightning 18 at 19), consistent with the "no row refused" claim of `docs/buildv3.md`.
- Discrepancy (doc only): the buildv3.md prose table places Frozen Armor at 6 and Frost Nova at 7; the JSON has Frozen Armor at 5 (with the Den point) and Frost Nova at 6. The JSON is the truth.
- No Static Field, Warmth or Energy Shield in v3 (V, JSON). Static Field and Warmth are cheap to unlock early (6 and 1) so an unlock event cannot force a delay; they are display-only under v3.

## Static Field and Warmth timing

- Static Field needs clvl 6; Warmth clvl 1, no prerequisites. The SME's events (Bishibosh / Cold Plains for Static Field, Blood Raven for Warmth) are therefore never blocked by level: the earliest legal clvl is 6 and 1, and Bishibosh is met at clvl 7 to 8 (S). Keep them free-choice-point candidates only (ADR 0018 item 6); the build table does not spend on them.
- Frozen Armor and Ice Bolt are level 1: the SME's "free from clvl 6" is a level claim that is wrong; keep them free with no event (the bot needs Frozen Armor for survival).

## Teleport timing and the unlock event

- Teleport needs clvl 18 and Telekinesis (clvl 6, spent at 11 by v3). v3 spends it at 18; the Catacombs are run at clvl 14 to 17 (S).
- Under the SME's event (Andariel killed) the clvl-18 point would flow to the next legal row (Chain Lightning, legal from 18) and Teleport would wait until Andariel dies, which also loses the proven nav hops (`docs/capabilities.md` line 36, clvl 18 live). The SME also says Teleport must not be needed to reach Andariel.
- Recommendation: unlock event `waypoint:catacombs_2` (Catacombs level 2 waypoint, lit at about clvl 16 to 17, S). Because the level requirement is 18 the unlock can never make Teleport available earlier than v3 spends it, and in the normal path the waypoint is lit before clvl 18, so no v3 row is refused. Edge: a hero that reaches 18 before lighting the waypoint sends the point to Chain Lightning (reorder, not a loss; nothing is banked). Nothing the bot needs to reach Andariel depends on Teleport: the fallback is walking, which the nav keeps (U for the Catacombs walking time at clvl 16).
- Waypoint-lit event: U, no such event is emitted today (fun-ideas.md section 14); the table is readable on both backends.

## Other findings

- Energy Shield prerequisites in these tables are Teleport and Chain Lightning (level 24); the author remembers the classic prerequisite differently (U). Irrelevant to v3 (no Energy Shield).
- Not checked: other classes' ReqLevels. TASK-143 reads them from the same tables and fails a unit test (not this note) if a name is unknown.
