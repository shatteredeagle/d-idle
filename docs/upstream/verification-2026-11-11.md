# Independent verification of upstream PR candidates 1 and 2 (2026-11-11)

> status: WIP (agent prverify). Verdicts below are filled as the runs finish.

Verifier: d2-reviewer (Opus), singly, read-mostly. Material: `docs/upstream/pr-queue.md` and `docs/upstream/drafts/*.md` on branch `agent/prprep` (tip 49cfd0f1), the private fork on the worker host. Tags: V read (disassembly read here), V run (run here), S / U.

## Setup (V run)
- Upstream main moved since the prep: `962ffce2` (2026-11-11, #414 trace docs and an Imbue include fix), not `b6b59ae9`. Both branches rebase onto it with no conflict (detached worktrees `~/d2d-lab/wt-pv-light`, `~/d2d-lab/wt-pv-cold`, script `~/d2d-lab/pv/pv_setup.sh`): Lightning `9e756c76` (oracle) + `4b80e1e2` (claim); Cold `8c327675` (oracle) + `37bd27ad` (claim).
