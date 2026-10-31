# d2d container: risks, effort, questions

Effort in focused agent-hours (U, estimates; the existing plan puts the container recipe at 8 to 12 h after the A/B report, S docs/d2d-backend.md phase 4). Tags V/S/U as in design.md.

## Effort per step
| # | Step | Hours | Blocked by |
|---|---|---|---|
| 1 | Vendor bundle script (git archive of fork, CLI11, sentry-native with recursive submodules, FFmpeg tarball with hash check), `.dockerignore`, context assembler | 2 | none |
| 2 | First test build of the d2d stage on the worker host (g++ 15 / C++26 proof, Qt-only-for-configure, offline flags). Fixes if g++ 15 fails: switch to g++-16 from the 26.04 archive (V apt shows g++-16 exists) | 3 to 5 (mostly waiting) | step 1 |
| 3 | Optional fork patch: Sentry optional and launcher skipped (shrinks build, no Qt, no crash code) | 2 to 3 | step 2 |
| 4 | Runtime image: notices, licences, `SOURCE.txt`, non-root, tini, healthcheck; smoke test with MPQs (`ping`, `state`, create character, save) | 2 | step 2 |
| 5 | `d2bot` entry that drives d2d (backend selection, Lifecycle wiring, `D2D_MODE`, Pacer vs wall clock, per-slot sockets under `/run/d2d`) | 12 to 20 (phases 1 to 3 of the plan are the bulk) | d2d backend phases |
| 6 | `d2fleet` wizard for d2d (no prefix/patch/import steps; check MPQs; create character via d2d `--start-screen charcreate`; slots) | 6 to 10 | step 5 |
| 7 | Viewer source for d2d (`screenshot` verb to MJPEG) and the optional 8432 port | 4 to 6 | step 5 |
| 8 | d2d soak driver in the container (sidecar samples from `dump`/`state`, no /proc/pid/mem, game-time profile for lockstep) | 8 to 12 | steps 5, 6 |
| 9 | Template, load to Tower, UI-visible apply, first run by a non-agent per docs/container.md checklist | 3 | steps 4 to 6 and the user's go |
| 10 | One-slot 24 h soak in the container, restart/kill injection, memory curve (d2d growth, periodic d2d restart interval) | 4 active, 24 wall | steps 8, 9 |
| 11 | GPL package: notices file for all dependencies, SOURCE.txt, licence files (only before any sharing) | 3 | user answers |

Total to a NAS-runnable image with a human-verified smoke test: steps 1 to 4 and 9 about 12 to 15 h; the product (steps 5 to 8, 10) another 35 to 50 h, mostly the backend work that is planned anyway.

## Risks
1. C++26 toolchain (U): never built here with g++ 15 (the worker host used 16.2.1). Mitigation: step 2 first; g++-16 exists on 26.04 (V).
2. Offline vendoring of Sentry submodules (U, not tried); recursive submodules, shallow flags. Mitigation: optional-Sentry patch (step 3).
3. Fork drift: rebases onto upstream (policy: monthly at most, V d2d-backend.md 4.2); the image pins a commit, so drift costs only when we update.
4. Memory growth of d2d per first-visited level (about 100 MB, V) over days: needs periodic d2d restart, never tested beyond 20 min (V) at real time; no multi-hour d2d run exists (U).
5. d2d single-thread cost: 1 core per slot, same as Wine; the NAS core count and load (U which cores are free) caps the slots; `--cpus` is a quota, not pinning.
6. d2d fidelity gate (A/B vs Game.exe) not done (S plan); the container is only worth it if it passes.
7. Viewer: d2d has no window; frame source must come from a verb (V `screenshot` exists, performance U).
8. Port clash on the NAS: d2idle holds 8431 (V); two containers cannot both publish it.
9. The NAS UI rule: everything must come from the template; `tpl-apply.sh` recreates and starts the container (V nas-container.md): stop again if unwanted.
10. GPL/notice obligations before sharing (gpl.md); Freepik icon terms (U).
11. NAS kernel ignores swap limits (S): a memory limit is a hard kill, so size `--memory` with margin.
12. HEALTHCHECK and the NAS: whether the Docker tab shows health status (U); harmless if not.

## Minimal questions for the user
1. Host port: stop or remove the Wine `d2idle` so `d2dbot` takes 8431, or give d2dbot another host port (suggest 8434), meanwhile? And do you want the optional viewer port 8432 (LAN only), or keep the viewer proxied on the dashboard port?
2. Default mode: real time for the live container and lockstep only in a separate soak container (my recommendation)? Is a lockstep soak verdict acceptable as G1/G2 evidence, or must the gate run in real time?
3. Where do the MPQs live on the NAS: reuse the staged Wine game dir (`/mnt/user/appdata/d2idle/game`) by copying only the MPQs to `/mnt/user/appdata/d2dbot/game`?
4. OK to spend effort on the optional fork patch (Sentry optional, launcher skipped), accepting a slightly larger fork diff, to drop Qt and crash code from the build?
5. GPL: who reviews, which licence for our code, and may the fork be published when an image is shared (gpl.md questions 1 to 6)? None needed while everything stays private.
6. Go-ahead for the first test build on the worker host (about 15 to 25 min, niced as far as docker allows, about 10 GB scratch)?
