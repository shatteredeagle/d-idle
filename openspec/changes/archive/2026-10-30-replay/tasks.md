# Tasks

```yaml
change: replay
status: archived
agent: replay
branch: agent/replay
worktree: ~/code/d2lod-wt/replay
owns:
  - cmd/d2dreplay/**
  - internal/d2dreplay/**
  - docs/d2d-replay.md
  - docs/probes/replay/**
  - openspec/changes/replay/**
exclusive: []
gates:
  - "go test ./internal/d2dreplay ./cmd/d2dreplay passes (format round trip, tap records verbs and checkpoints, player detects a divergence at the first mismatching checkpoint)"
  - "scripted walk+fight run: record then play 3 times, 3/3 checkpoint hashes match (live, lab5)"
  - "full-bot run: record then play once; result reported honestly (OK or first divergent tick)"
  - "replay file under 1 MB per virtual hour (measured on the recorded runs)"
  - "tools/openspec.sh validate --all --strict passes; changecheck 0 warnings"
merge: 7a14a3e
result: "replay recorder/player/render (internal/d2dreplay, cmd/d2dreplay): *.replay.gz format, hash detloop-journal-v1, scripted run played OK 3/3, full-bot 36/36 until first restart; blocked across restarts on the relaunch recipe"
```

## 1. Design
- [x] 1.1 change card, ADDED spec `replay` (file content, passive recording, lockstep playback, divergence report, render, size, no secrets), validate strict

## 2. Library (`internal/d2dreplay`, stdlib, tests with the d2dctl fake)
- [x] 2.1 format: header, verb entries, checkpoints, restart marker, trailer; gzip JSON lines; writer flushes; reader tolerates a truncated tail
- [x] 2.2 tap: one client, forward every line, record mutating verbs with tick numbers, checkpoint reads every N ticks, redial on upstream drop (restart marker)
- [x] 2.3 player: replay verbs in lockstep, hash at every checkpoint, stop at first divergence, optional wall-gap reproduction, hooks for render
- [x] 2.4 launcher: d2d from the recipe (seed, class, name, args) in a private home, wait for ping, quit at the end

## 3. CLI (`cmd/d2dreplay`)
- [x] 3.1 `record`, `play`, `info`, `render` (frames; ffmpeg video when present, never installed)

## 4. Proof (live, the worker host, lab5, under the lab lock, throwaway character)
- [x] 4.1 scripted walk + fight (det.py scenario): record, play x3, hashes 3/3
- [x] 4.2 short full-bot run through the tap: record, play, report OK or first divergence
- [x] 4.3 render a short stretch to frames (+ video if ffmpeg)

## 5. Docs
- [x] 5.1 docs/d2d-replay.md, demo file under docs/probes/replay/, card result
