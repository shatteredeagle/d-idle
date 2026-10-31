# Tasks

```yaml
change: mpqrecheck
status: archived
agent: mpqrecheck
branch: agent/mpqrecheck
worktree: ~/code/d2lod-wt/mpqrecheck
owns:
  - docs/process/mpq-data-reading.md
  - docs/d2d-lab.md
  - docs/research/mpq-recheck-pr-claims.md
exclusive: []
gates:
  - "every PR data claim re-read with the patched tables and marked MATCHES or DIFFERS"
merge: cae3486
result: "mpq-cat serves 1.10 tables unless a lowercase patch_d2.mpq symlink exists; price and cast PR data claims all MATCH real 1.14d data; one cast PR text claim (no seqtrans column) is wrong and being fixed; sanity check: monstats.txt header must start Id<TAB>hcIdx"
```

## 1. Work

- [x] 1.1 Re-read books, misc, itemtypes, animdata, skills with patched and unpatched stacks and compare with the PR texts
- [x] 1.2 Write docs/process/mpq-data-reading.md and the d2d-lab.md note
