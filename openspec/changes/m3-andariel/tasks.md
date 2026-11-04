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
  - "series at the fight floors (15 with a living merc, 17 without, life floor = N from the diagnostic run (step 2/4)) with the kit (8 healing, 3 antidotes, 4 mana, tome with at least 2 charges): 5 clean wins in a row (or 7 of 8 with a cause for the loss), Andariel dead, character alive, antidote drunk on poison, writes after start: 0, `boss andariel` log lines with life minimum and potions used (raw in docs/probes/andariel); includes handicapped variants (merc at 50 percent life, kit at the gate minimum, tome at 2 charges); retreat by portal at 30 percent (escape) or 12 percent (chicken), one re-attempt after a refill, two retreats end the attempt, any death resets the series"
  - "the no-merc case at least 3 times, reported (pass or fail, with the cause); test that a dead merc flips the floor between 15 and 17 mid-run"
  - "go test -p 1 ./internal/boss passes including TestAndarielFightOrder (antidote, then Charged Bolt; Static Field only if a measured drill shows it helps) and TestAndarielScriptHasNoStaticFieldOpener unchanged"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; cheat ledger has one entry per cheat with a reason; tools/agentreap.sh m3-andariel leaves nothing"
merge:
result: "research note only (agent andychain, no fight run): docs/research/andariel-chain-needs.md; effective Andariel floor is 20 without a merc (route guard), SME says 18 with merc and 21 without"
```

## 1. Work

- [ ] 1.1 Read docs/bossgate.md, internal/boss/act1.go, docs/sme/play-act-route.md section 4; write the fight order test first
- [ ] 1.2 Prepare the throwaway (cmd/d2cheat: level, gold, kit; merc claimed honestly or hired per merc-policy: no paid Act 1 hire before Blood Raven) on a hidden slot
- [ ] 1.3 the series (with the merc, handicapped variants, at least 3 without a merc; setup may be goto Catacombs 4 plus a given merc and kit, declared); docs/andariel.md; changecheck, leakcheck, validate; reap
