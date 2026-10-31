# Tasks

```yaml
change: d2d-container-soak
status: proposed
agent: d2d-container-soak
branch: agent/d2d-container-soak
worktree: ~/code/d2lod-wt/d2d-container-soak
owns:
  - docs/d2d-container-soak.md
exclusive:
  - soak-slot:1
gates:
  - "24 h soak report.txt says PASS with exit 0"
  - "memory curve and periodic d2d restart interval recorded"
merge:
result:
```

## 1. Work

- [ ] 1.1 template and first run by a non-agent per docs/container.md (step 9, user go)
- [ ] 1.2 24 h one-slot soak with restart and kill injection, memory curve (step 10)
- [ ] 1.3 apply the spec delta when this merges
