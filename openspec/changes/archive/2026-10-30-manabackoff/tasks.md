# Tasks

```yaml
change: manabackoff
status: archived
agent: manabackoff
branch: agent/manabackoff
worktree: ~/code/d2lod-wt/manabackoff
owns:
  - internal/resource/attack.go
  - internal/resource/types.go
  - internal/resource/tuning.go
  - internal/resource/policy_test.go
  - internal/env/tuning_resource.go
  - internal/combat/resource.go
  - internal/combat/fight.go
  - internal/town/shop.go
  - internal/town/spare_test.go
  - docs/earlygame.md
  - docs/knobs.md
exclusive: []
gates:
  - "a mana-starved caster in a big pack swings after the back-off cap instead of retreating for ever"
  - "a recovering caster buys mana potions up to the floor; non-casters and non-recovery behaviour unchanged"
merge: c31891d
result: "back-off cap resource.backoffMaxSteps 8 then melee, recoveryManaFloor 2 mana potions; Fresha 13% of an hour mana-starved; live effect pending"
```

## 1. Work

- [x] 1.1 Quantify the Fresha log
- [x] 1.2 Back-off cap with melee fallback, tests
- [x] 1.3 Recovery mana floor, tests
- [x] 1.4 Whole-repo tests on the worker host
