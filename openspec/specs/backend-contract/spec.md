# Backend contract specification

## Purpose
The seams that keep decision code backend-agnostic, so the same bot runs on the original Game.exe (Wine, /proc/pid/mem reads, input injection) and on the d2d reimplementation (control socket), and what each backend must provide. Source: docs/decisions.md 2026-10-30 rule 1, docs/d2d-backend.md, docs/d2d-bot.md, docs/d2d-wire.md (when merged).

## Requirements

### Requirement: Decision code stays backend-agnostic
New decision code MUST reach the game only through `env.Env` (`Read d2.StateReader`, `In input.Controller`, `Mem mem.Source`, `World *d2map.Tracker`, `Clock`, `Emit`) and the package seams, and SHALL NOT import `internal/d2dctl`, `internal/d2dsrc` or Game.exe specifics. Source: docs/decisions.md 2026-10-30 rule 1; internal/env/env.go (V read).

#### Scenario: New behaviour package
- **WHEN** a behaviour package needs the player's life
- **THEN** it calls `env.Snapshot()` or `Env.Read.Player()` and compiles without any d2d import

### Requirement: StateReader interface
Every backend MUST implement `d2.StateReader` with exactly the methods `Snapshot`, `Player`, `UI`, `Units`, `UnitStats`, `ReadStatList`, `ReadBaseStatList` and `RawStatList`, and `*d2.Reader` and the d2d snapshot source SHALL both satisfy it (compile-time assertion). Source: internal/d2/statereader.go (V read); merge 0fbb310 (d2diface).

#### Scenario: Interface drift
- **WHEN** a method is added to `StateReader`
- **THEN** both implementations fail to compile until updated

### Requirement: Input client
A backend MUST provide an `input.Controller`; the Game.exe backend uses the `d2input` helper over its per-instance port (47012 + N) and the d2d backend uses `internal/d2dinput` over the control socket, with the same semantic actions. Source: AGENTS.md; docs/d2d-backend.md section 2 (S); merge 3c0f26b (d2dinput).

#### Scenario: Instance selection
- **WHEN** `D2BOT_INSTANCE=i2` is set
- **THEN** `input.NewClient("")` targets the helper of instance i2 and no other

### Requirement: d2d control socket is single-client
At most one party SHALL drive a d2d control socket at a time, there is no authentication, and a bot MUST NOT attach to a d2d it did not start unless `--d2d-attach` is given. The managed start writes `<sock>.owner` and a stranger's d2d is never driven or quit. Source: docs/d2d-wire.md socket-safety section (V offline tests TestBackendFlags, TestManagedRefusesStrangerSocket, on branch d2dwire).

#### Scenario: Stranger socket
- **WHEN** `--managed` finds an answering socket without an owner marker
- **THEN** the start fails with an error and the other d2d is untouched

### Requirement: Per-instance sockets and paths
The default d2d socket MUST be per instance (`<run dir>/d2d-<D2BOT_INSTANCE>.sock`), never the shared lab socket, and each soak SHALL own its bot binary path (`~/d2lod-live/soak24-iN/d2bot`). Source: docs/d2d-wire.md, docs/soak24.md section 2 (V read).

#### Scenario: Two soaks
- **WHEN** soaks run on i4 and i5
- **THEN** they use different binaries, logs, ports and sockets

### Requirement: Backend gaps are named
Behaviour the d2d backend cannot provide yet (loot pickup, stat map, skill list, poison state) SHALL be wired as an explicit "not yet" stub that logs a warning, never as silent success. Source: docs/d2d-wire.md ("Loot NOT wired"), docs/d2d-backend.md section 1.2 (gap column).

#### Scenario: Missing capability
- **WHEN** the leveler needs base stats on d2d and `Mem` is `noMem`
- **THEN** it takes the "unreadable" path and logs it, and does not invent values

### Requirement: Original game remains the acceptance target
Anything verified only on d2d MUST be tagged V-d2d and is not an acceptance result until the fidelity-gate spec passes or the original game confirms it. Source: docs/decisions.md 2026-10-30; plan-v2 section 11.

#### Scenario: Probe result
- **WHEN** a d2d probe says the free Rogue is granted without a window
- **THEN** the doc tags it V-d2d and the original-game probe stays an open item
