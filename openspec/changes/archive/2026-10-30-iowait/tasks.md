# Tasks

```yaml
change: iowait
status: archived
agent: iowait
branch: agent/iowait
owns:
  - docs/process/worker-iowait.md
exclusive: []
gates:
  - "read-only: nothing on the worker host or GitHub changed"
merge: b0e57e5
result: "orphan find / on CIFS, baloo writes (docs only, measured on the worker host)"
```

## 1. Work

- [x] 1.1 Measure and write findings
