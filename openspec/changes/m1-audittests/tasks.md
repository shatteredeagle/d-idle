# Tasks

```yaml
change: m1-audittests
status: deferred
agent: m1-audittests
branch: agent/m1-audittests
worktree: ~/code/d2lod-wt/m1-audittests
owns:
  - internal/town/shop.go
  - internal/town/chores.go
  - internal/town/basicsaudit_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/town passes; TestAuditS2 asserts no shop visit at gold 120 with a scroll and a full belt; TestAuditS3 asserts 1 portal scroll and at least 150 gold kept at gold 300; TestAuditS7 asserts the unidentified ring under 100 is sold as is; TestAuditS10 asserts one repair attempt per (gold bucket, lowest durability); TestAuditS11 asserts 950 banked at gold 1500 clvl 5"
  - "`! grep -n PinsCurrent internal/town/basicsaudit_test.go` (no pinned gap remains)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read the archived basicsfix card and docs/basics-audit.md rows 2, 3, 7, 10, 11; list what basicsfix left open
- [ ] 1.2 Flip TestAuditS2, S3, S7, S10, S11 to the SME end state; fix residues in shop.go or chores.go
- [ ] 1.3 Check the merged purchase order against decision 5; write the four town-basics requirements from the merged code (numbers from docs/knobs.md)
- [ ] 1.4 changecheck, leakcheck, validate; report which audit rows flip to C (m1-liveproof edits the table)
