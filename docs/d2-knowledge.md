# D2 knowledge index (for the d2-sme agent and anyone asking "does this make sense to play")

Target: classic Diablo II LoD 1.14d, offline single player. Not D2R.

| question | read |
|---|---|
| memory layout, offsets, what the bot can see | docs/d2-1.14d-memory.md |
| researched knowledge base (mechanics, builds/gear, human play patterns) | docs/sme/ |
| class plans, skill orders | docs/classes/ |
| endgame builds and gear targets | docs/builds-endgame.md |
| how the bot decides (goals, gates, farm runs) | docs/knobs.md, docs/deathspiral.md, docs/stallfatal.md, PLAN.md |
| what happened in live runs | docs/g1-log.md, docs/soak-analysis.md, docs/liveruns*.md |
| safe levelling and resist thresholds the bot uses | internal/resource (GearGate, FieldGate), internal/task gate code |

Ask the SME with the d2-sme agent type; it tags claims SRC / EXP / UNSURE. Anything it states about mechanics still gets verified live before it becomes bot behaviour.
