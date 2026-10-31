# Replay specification

## Purpose
What a d2d replay file holds, how it is recorded without touching the run, and what playback must guarantee so that a replay doubles as the determinism oracle for the lockstep fork. Source: docs/d2d-fork.md section 5, docs/d2d-loop.md phase 0, user decision 2026-10-30 ("Determinism + replay"), evidence in docs/d2d-replay.md.

## ADDED Requirements

### Requirement: Replay file content
A replay file SHALL contain only what re-creates the run: format version, d2d build hash (sha256 of the binary), the start recipe (seed, class index, character name, start screen, hardcore/classic flags, non-path d2d arguments) or the sha256 of the start save, the ordered control verbs with the virtual tick before each, state-hash checkpoints every N ticks, a trailer (ticks, verbs, durations) and metadata (bot version, note). Source: user decision 2026-10-30; docs/d2d-replay.md.

#### Scenario: Header and body
- **WHEN** `info` reads a file
- **THEN** it prints the version, build hash, recipe, checkpoint interval, verb count, tick count and durations without starting a game

### Requirement: No game data or secrets in a replay
A replay file MUST contain no game data, save bytes, secrets, file paths or hostnames. Source: user decision 2026-10-30 (shareable replays); AGENTS.md (no secrets in the repo).

#### Scenario: Paths and hosts stay out
- **WHEN** a run is recorded with a socket under the user's home and screenshots to an absolute directory
- **THEN** the file holds neither the socket path, the home directory, the screenshot paths nor the host name (the `screenshot` verb is dropped, paths are never written)

#### Scenario: Versioned and compressed
- **WHEN** a file is written
- **THEN** it is gzip-compressed JSON lines whose first line names the format version, and a reader refuses an unknown major version with a clear error

### Requirement: Passive recording
Recording SHALL be a tap between the client (bot or script) and d2d: every line is forwarded unchanged and answered with d2d's reply; the tap records mutating verbs and step counts and performs its own read-only checkpoint reads (`state`, `debug players`, `monsters`) between client commands. The client's behaviour MUST NOT change (no injected input, no rewritten lines, read verbs pass untouched and are not stored). Source: docs/d2d-view.md (tap design), docs/d2dctl.md (one client).

#### Scenario: Bot through the tap
- **WHEN** a bot dials the tap's socket instead of d2d's
- **THEN** every reply it receives is d2d's own and the file lists exactly the mutating verbs it sent, in order, with their tick numbers

#### Scenario: Upstream restart
- **WHEN** the d2d behind the tap exits and a new one listens on the same socket (managed bot restart)
- **THEN** the tap redials, writes a restart marker with the tick, and keeps recording

### Requirement: Lockstep playback verifies checkpoints
Playback SHALL start a d2d of the same recipe and seed in lockstep, send the recorded verbs in order, compute the checkpoint hash at every recorded checkpoint tick and compare it with the file: `OK` with the checkpoint count when all match, otherwise the first divergent tick, checkpoint index and both hashes. Source: docs/d2d-fork.md section 5 (determinism method), docs/d2d-replay.md.

#### Scenario: Different build
- **WHEN** the sha256 of the d2d binary used for playback differs from the header
- **THEN** the mismatch is printed before playback starts and a later divergence is attributed to the build

#### Scenario: Restart marker
- **WHEN** playback reaches a restart marker
- **THEN** it stops there, reports the checkpoints verified so far and says that restarts are not replayed

#### Scenario: Same build, scripted run
- **WHEN** a scripted walk and fight recorded on lab5 is played back on lab5 three times
- **THEN** every checkpoint hash matches (3/3 runs OK)

#### Scenario: Divergence
- **WHEN** a checkpoint hash differs
- **THEN** playback stops, prints the first divergent tick with the expected and the observed hash, and exits non-zero

### Requirement: Checkpoint hash
The checkpoint hash SHALL be the detloop journal checkpoint: sha256 over the replies to `dump player,stats,items,quests,ui,merc`, `monsters` and `ground` (payload lines each with a newline, the ok rest, a zero byte, per verb in that order), named `detloop-journal-v1` in the header, so the determinism harness and the replay tool confirm each other. Source: agent/detloop docs/d2d-determinism.md (journal checkpoint), docs/d2d-loop.md phase 0.

#### Scenario: Algorithm named
- **WHEN** the header names a hash algorithm the player does not implement
- **THEN** playback refuses with the algorithm name instead of comparing wrong values

### Requirement: Render from a replay
`render` SHALL play back like `play` and, every stride of virtual time, request a screenshot into the output directory; with ffmpeg on PATH it also writes a video at the requested frame rate (speed factor = stride x fps), and it never installs anything. Source: docs/d2d-view.md (screenshot verb, lab5 draws on request).

#### Scenario: Frames without ffmpeg
- **WHEN** ffmpeg is not on PATH
- **THEN** the frames are written, no video is produced and the message says so

### Requirement: Size
A replay file SHALL stay under 1 MB per virtual hour of a full bot run (read verbs are not stored; verbs are short text; gzip). Source: brief 2026-10-30; measured in docs/d2d-replay.md.

#### Scenario: Measured
- **WHEN** a bot run of at least 10 virtual minutes is recorded
- **THEN** the file size extrapolated to one virtual hour is under 1 MB
