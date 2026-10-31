# Decisions (user, with date)

Index of the decision records in [adr/](adr/README.md) (the dated entries below stay as the verbatim source):

- [ADR 0001](adr/0001-container-end-goal-native-dev.md) (2026-10-29, accepted): The end goal is a container; native on the worker host stays the dev path
- [ADR 0002](adr/0002-d2d-primary-backend-gated.md) (2026-10-30, superseded by 0009): d2d becomes the likely primary backend; the original game stays the acceptance target
- [ADR 0003](adr/0003-d2d-fork-private-gpl.md) (2026-10-30, accepted): The d2d fork stays private on the forge; no d2d code in this repo; nothing goes upstream without the user
- [ADR 0004](adr/0004-den-first-and-early-gear-policy.md) (2026-10-30, accepted): Den of Evil first, free skill point, no early vendor gear
- [ADR 0005](adr/0005-death-rule.md) (2026-10-30, accepted): Deaths are judged by streak, recovery share and rate, not by a fixed count
- [ADR 0006](adr/0006-free-rogue-merc.md) (2026-10-30, accepted): Take the free Rogue (Blood Raven reward); never pay Kashya first
- [ADR 0007](adr/0007-freeze-lift-and-verification-phases.md) (2026-10-30, accepted): Lift the feature freeze for M0 to M9; G2 is a stability and economy soak
- [ADR 0008](adr/0008-no-kill-injection-without-permission.md) (2026-10-30, accepted): Agents do not kill the soak game process unless a permission rule allows it
- [ADR 0009](adr/0009-d2d-primary-backend.md) (2026-10-30, accepted, grill 0001): d2d is the primary backend now; Game.exe is frozen, stays the acceptance reference, retires at the d2d container's 24 h soak; exit from d2d is break glass only
- Licence: GPL (same as d2d, GPL-3.0-or-later) - user, 2026-10-30 ([ADR 0010](adr/0010-licence-same-as-d2d.md))
- Repo going public soon; lockstep patches stay in the downstream fork until upstream wants them - user, 2026-10-30 ([ADR 0010 addendum](adr/0010-licence-same-as-d2d.md))

## 2026-10-29: the end goal is a container; native on the worker host stays the dev path
- **Product:** the container (deploy/container, d2fleet wizard, bot-owned game per slot) is the supported way to run D.idle, and the cross-platform answer (Linux, the NAS, Windows/Intel Mac via Docker). Docker and virtualization are assumed on the host: no setup docs for them.
- **Development:** native on the worker host (tools/live.sh, game.sh, soak24.sh, managed d2bot) continues for as long as it makes rapid iteration possible (no container rebuild per change). It is an internal dev mode, not a supported product path.
- **Rules that follow:**
  - Do not add native-only user features or polish host scripts for their own sake; new runtime behaviour must work in the container too (explicit mounts only, no host assumptions).
  - The container is verified with the same stamped binaries: first on one slot after the 24 h soak (docs/review-strategy.md section 3), by a non-agent (docs/container.md checklist). Windows/WSL2 and Apple Silicon are unproven (docs/soak-hosts.md, docs/quickstart.md).
  - A Windows-native port is NOT planned; revisit only if the container is a bad experience on Windows.
- **Open container items:** viewer for the installer (needs a second port: user decision pending), soak driver inside the container, first stranger test.

## 2026-10-30: likely move to d2d (native reimplementation) as the primary game backend; decision gated on the spike
- **Context:** https://github.com/psi29a/d2decomped (d2d) is an open-source GPL-3.0 reimplementation of the D2 LoD engine (1.14d as ground truth), native, with a local control channel and debug verbs; no Wine/Proton. The user: 'we'll likely switch to using d2d primarily if it runs containerized better, which I expect it will'. Spike running: docs/d2d-spike.md (D2DSPIKE), notes research: docs/sme/d2decomped-findings.md (D2DHARVEST).
- **Direction (tentative until the spike answers):** d2d becomes the primary backend IF it builds and runs headless in a container, fidelity is good enough for the bot's behaviours and the control channel can carry what the bot needs. The original game under Wine stays the acceptance/reference target (and the bot's current proven path) until d2d proves out. The user's earlier plan stands: first a solid foundation and bot behaviour, then revisit the harness/controller for d2d armed with what we learned.
- **Rules that follow now:** (1) keep the bot's behaviour logic backend-agnostic: new code goes behind the existing interfaces (internal/mem snapshots, internal/input client), no new Wine/Game.exe-specific assumptions in decision code; (2) do not invest further in Wine/Proton-specific container work (prefix handling, winedevice reaping, sandbox for Wine) beyond keeping the current paths working; (3) d2d is GPL-3.0: talk to it over its socket as a separate program, never copy its code or notes verbatim into this repo (own summaries with file-path citations), decide our own license at distribution time (see the license discussion: no change needed while private); (4) probes that d2d can answer cheaply (death/corpse/gold, quest persistence, merc hire, NPC menus) go through d2d first, then are confirmed on the original game when cheap.

## 2026-10-30 (user answers, morning)
- d2d fork lives in private forge repo `<owner>/d2d-fork` (branch main, two commits on upstream e36922e: P2 lockstep/time-scale/step/clock, P1 dump verb). GPL reviewer: none named; the user has no idea who reviews, so no distribution happens until that is decided (repo stays private).
- Mephisto quit-safety: test it when we get there (no dedicated probe slot now).
- Fresh character created: Fresha (Sorceress, softcore expansion, level 1) on i5, verified by d2savecheck. Not started as a soak yet: G2 runs on i4 and the worker host idle is under the health guard.
- Wine container work is PAUSED (the d2idle container on the NAS is not redeployed or extended until the d2d decision; keep it running as is).
- 2026-10-30 (user): any ports the d2d container needs may be used (no per-port approval). Still: UI-visible only (NAS template), nothing deployed until the d2d primary decision and the container work resumes.
- LAN/networked play allowed (local original host <-> d2d client; no official Battle.net) - user, 2026-10-30
