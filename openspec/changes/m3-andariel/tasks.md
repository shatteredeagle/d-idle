# Tasks

```yaml
change: m3-andariel
status: deferred
agent: m3-andariel
branch: agent/m3-andariel
worktree: ~/code/d2lod-wt/m3-andariel
owns:
  - internal/boss/act1.go
  - internal/boss/andariel_d2d_test.go
  - docs/andariel.md
  - docs/probes/andariel/**
exclusive:
  - soak-slot:3
gates:
  - "3 of 3 fights at clvl boss.andarielMinLevel with the kit and a living Rogue: Andariel dead, character alive, antidote drunk on poison, `boss andariel` log lines with life minimum and potions used (raw in docs/probes/andariel)"
  - "1 fight at the same level without a merc reported (pass or fail, with the cause)"
  - "go test -p 1 ./internal/boss passes including TestAndarielFightOrder (antidote, Static Field to the cap, then Charged Bolt)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; cheat ledger has one entry per cheat with a reason; tools/agentreap.sh m3-andariel leaves nothing"
merge:
result: "research note only (agent andychain, no fight run): docs/research/andariel-chain-needs.md; effective Andariel floor is 20 without a merc (route guard), SME says 18 with merc and 21 without"
```

## 1. Work

- [ ] 1.1 Read docs/bossgate.md, internal/boss/act1.go, docs/sme/play-act-route.md section 4; write the fight order test first
- [ ] 1.2 Prepare the throwaway (cmd/d2cheat: level, gold, kit; merc claimed honestly or hired per merc-policy: no paid Act 1 hire before Blood Raven) on a hidden slot
- [ ] 1.3 3 fights with the merc, 1 without; docs/andariel.md; changecheck, leakcheck, validate; reap
