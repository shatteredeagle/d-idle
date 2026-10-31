# Tasks

```yaml
change: whisperlook
status: archived
agent: whisperlook
branch: agent/whisperlook
worktree: ~/code/d2lod-wt/whisperlook
owns:
  - docs/research/d2d-whisper-lan-oracle.md
exclusive: []
gates:
  - "no GitHub posting; web/GitHub GET only"
merge: e02481a
result: "whisper = ijl11.dll side-load adding a token-protected control socket in the original game; d2dproxy = TCP MITM on port 4000 logging packets; net-join = headless d2d client; verdict: packet/state oracle, not a soak oracle; first demo 3-5 agent-hours"
```

## 1. Work

- [x] 1.1 Read whisper and d2proxy in psi29a/d2decomped
- [x] 1.2 Write the oracle assessment and experiment card sketch
