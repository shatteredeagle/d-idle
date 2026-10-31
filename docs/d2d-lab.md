# d2d lab binary (private fork, on the worker host)

> status: partly stale: the current lab binary is d2d.lab5 (docs/d2d-fork.md, docs/d2d-loop.md); the lab2/lab3 sections below are history. Slots: tools/d2dslot.sh (docs/process/worker-compute.md).

One binary with every verb our tooling needs: `~/d2d-lab/fork/bin/d2d.lab` (read-only, never overwritten; sha256 and provenance in `bin/d2d.lab.txt`).

- Fork repo: `~/d2d-lab/fork/src`, remote `forge` = `ssh://nas.example:2222/<owner>/d2d-fork.git` (private). Never push to the GitHub upstream.
- Branches: `main` = upstream e36922e + P2 (lockstep clock) + P1 (`dump`); `d2dtown-vendor` = main + `dump vendor`; `p3-layout-verb` = main + `debug layout` / `debug nearby`; `lab` = d2dtown-vendor + p3 cherry-pick (the one to use).
- Build: git worktree `~/d2d-lab/fork/wt-lab`, build dir `~/d2d-lab/fork/build-lab` (Ninja, Release, `-DD2D_WERROR=OFF`: GCC 16 gives a maybe-uninitialized false positive in town.cpp/ui.cpp under -Werror; deps reused from src/build via FETCHCONTENT_SOURCE_DIR_*; ffmpeg is rebuilt). Rebuild: `cd build-lab && nice -n10 ninja -j4 d2d`, then copy to a NEW name in bin/.
- `start.sh` now defaults to `bin/d2d.lab`; `D2D_BIN=<path>` overrides (old shared build `src/build/apps/d2d/d2d`, or `bin/d2d.pre-vendor`).
- Verified verbs (private run, `~/d2d-lab/fork/verify-lab.sh`, own socket in home-lab, `--no-save --lockstep`): `debug layout`, `debug nearby`, `dump`, `dump vendor`, `clock`, `step`.

## lab2 (current default)

`~/d2d-lab/fork/bin/d2d.lab2` (provenance and sha256 in `bin/d2d.lab2.txt`), branch `lab2` on the forge, worktree `~/d2d-lab/fork/wt-lab2`, build dir `build-lab2`.

- Base: our four patches (P2, P1, `dump vendor`, P3) were rebased with `git rebase --onto 3306e60 e36922e` (no conflicts, patches intact), then `fix/vendor-prices` (4884b1d) and `fix/cast-timing` (484e847) were cherry-picked from `~/d2d-lab/pr/{price,cast}`. `lab` and `p3-layout-verb` stay on the old e36922e base.
- Verified (private run, `verify-lab2.sh`, `meas-lab2.py`, `tome-lab2.py`): all verbs of `lab`; mean cast period Fire Bolt 13.0, Charged Bolt 13.1, Lightning 19.1 ticks; Akara tome (tbk) 450, Identify tome (ibk) 360 in `dump vendor`.
- `start.sh` defaults to `d2d.lab2`; `D2D_BIN=~/d2d-lab/fork/bin/d2d.lab` restores the previous one.

## Data reading trap (mpq-cat)

`mpq-cat` only finds the lowercase `patch_d2.mpq`; our data copy has `Patch_D2.mpq`, so it silently serves the old 1.10 tables (d2d itself and the Go reader are fine). Read tables through a scratch dir with a lowercase symlink and run the one-line sanity check (`monstats.txt` header must start `Id hcIdx`). Recipe, symptoms and checks: docs/process/mpq-data-reading.md; PR recheck: docs/research/mpq-recheck-pr-claims.md.

## lab3 (new verbs, agent D2DVERBS, 2026-10-30)

Branch `lab3` on the forge (`<owner>/d2d-fork`) = `lab2` + 3 patch commits; worktree `~/d2d-lab/fork/wt-lab3`, build dir `build-lab3`, binary `~/d2d-lab/fork/bin/d2d.lab3` (sha256, commit and verb list in `bin/d2d.lab3.txt`, mode 555). `start.sh` still defaults to `d2d.lab2`; try lab3 with `D2D_BIN=~/d2d-lab/fork/bin/d2d.lab3 ~/d2d-lab/fork/start.sh` (it becomes the default only after an end-to-end check on the Go side). Verb reference: docs/d2d-fork.md section "lab3 verb reference". Verification script (ours, on the worker host): `~/d2d-lab/fork/verify-lab3.py`, `verify2-lab3.py` (private socket, `--no-save --lockstep`).
