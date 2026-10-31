# Tasks

```yaml
change: m1-audittests
status: archived
agent: m1loot
branch: agent/m1loot
worktree: ~/code/d2lod-wt/m1loot
owns:
  - internal/town/shop.go
  - internal/town/chores.go
  - internal/town/basicsaudit_test.go
  - internal/town/basicsfix.go
  - internal/town/town_test.go
  - internal/env/tuning_town.go
  - docs/knobs.md
  - docs/basics-audit.md
exclusive: []
gates:
  - "go test -p 1 ./internal/town passes; TestAuditS2 asserts no shop visit at gold 120 with a scroll and a full belt; TestAuditS3 asserts 1 portal scroll and at least 150 gold kept at gold 300; TestAuditS7 asserts the unidentified ring under 100 is sold as is; TestAuditS10 asserts one repair attempt per (gold bucket, lowest durability); TestAuditS11 asserts 950 banked at gold 1500 clvl 5"
  - "TestAuditPurchaseOrder (new): at gold 100 to 190 with an empty belt and no scroll, potions are bought first up to 3 and a scroll only when gold >= scroll price plus 2 potions; the clvl >= 4 with >= 100 gold rule no longer buys a scroll ahead of potions"
  - "`! grep -n PinsCurrent internal/town/basicsaudit_test.go` (no pinned gap remains)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: ee0205e
result: implemented offline (sim rig, not verified live); see proposal Result and docs/loot-charms.md
```

## 1. Work

- [x] 1.1 Read the archived basicsfix card and docs/basics-audit.md rows 2, 3, 7, 10, 11; list what basicsfix left open
- [x] 1.2 Flip TestAuditS2, S3, S7, S10, S11 to the SME end state; fix residues in shop.go or chores.go
- [x] 1.3 Change the merged scroll rule (`town.tpFirstLevel` 4, scroll at clvl >= 4 with >= 100 gold left) to the D3 order: potions first up to 3, scroll only when gold >= scroll price plus 2 potions; test at gold 100 to 190; write the four town-basics requirements from the merged code (numbers from docs/knobs.md)
- [x] 1.4 changecheck, leakcheck, validate; report which audit rows flip to C (m1-liveproof edits the table)
