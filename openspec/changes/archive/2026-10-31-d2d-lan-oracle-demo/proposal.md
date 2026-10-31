## Why

User decision 2026-10-30: LAN/networked play is in scope (no official Battle.net). The WHISPERLOOK research (docs/research/d2d-whisper-lan-oracle.md) found that a d2d client can join an original Game.exe host over LAN TCP (port 4000), giving a packet/state oracle for d2d fidelity (not a soak oracle).

## What Changes

- demo: d2d net-join into an original-hosted LAN game, packets logged through d2dproxy, town layout compared
- doc docs/d2d-lan-oracle.md
- starts after origprobes is finished; hidden instance only
