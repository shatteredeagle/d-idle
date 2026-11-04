## Why

SKPASS finding B2 (docs/research/sk-passives-trace.md): game.exe's FUN_00644180 adds stats 127, 83, 188, 97 (cap 3 on own-class skills), 126, 107 and the skill shrine's +2 to a skill's cast level (damage, mana cost, duration, passives); synergies read base points only. The bot's `Skill.Effective()` read base plus stat 107 only, so mana cost and the known-skill gate read low with +class, +tab or +all-skills gear or a shrine.

## What Changes

- internal/d2/skilllevel.go: `SkillBonus` (pure rule table), `Player.Base`, `Player.Effective`, `SkillRef`.
- internal/combat/skills.go: `KnownSkills` casts at `Player.Effective`; `SkillRefOf` resolves own class, tab and element from the data.
- No change to synergy, leveler or display callers (audit in docs/fixskilllvl.md).

## Impact

Offline only, not verified live; the d2d backend exposes none of the stats (its bonus stays 0). Overlap (declared): internal/d2/** and internal/combat/skills.go are owned by main; this card adds new files in them and edits skills.go (small edit). No spec names this, so no delta.
