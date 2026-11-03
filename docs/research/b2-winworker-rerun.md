# Batch 2 r3 on the Windows build worker: partial re-run (2026-11-03)

> status: PARTIAL. MSVC 14.51, WSL parity stages and two damage-roll sets ran; mana, Teleport seed 9 and the real-`Fight::absorb` set (v4) did NOT run (below). Tags: V = seen in the run output of 2026-11-03.

Heads (V, bundle made from the private fork): `local/batch2-bundle-r3` 824da884, `local/batch2-mana-teleport` dc6618ae, `local/batch2-monsterdmg-pr` bbd339ad, base upstream main 005c7cab.
Machine: the Windows build worker, work window opened by the owner for the whole day; gate, low priority, 8 of 16 threads.

## MSVC 14.51 (VS 2026, 19.51.36260, same flags as upstream CI, WERROR off; apps off) (V)
| Ref | Build wall | `warning C` | Errors | ctest |
|---|---|---|---|---|
| base 005c7cab | 204 s (cold 103 steps) | 0 | 0 | 71 tests, 0 failed, 31 skipped (no game data in this lane) |
| bundle-r3 824da884 | 173 s | 0 | 0 | 73, 0 failed, 33 skipped |
| mana-teleport dc6618ae | 175 s | 0 | 0 | 73, 0 failed, 33 skipped |
| monsterdmg-pr bbd339ad | 173 s | 0 | 0 | 71, 0 failed, 31 skipped |
`test_monster_ai` builds (no C2026). The skipped tests include test_sk_mana and test_tele_spot (they need game files; the WSL lane has them). Command: `the parity script --msvc18 --bundle B --labels "b2r3base b2r3 b2mt b2md"`.

## WSL Linux parity (Ubuntu 26.04, g++ 15.2, clang++ 21, tidy 21; tools/parity.sh --worker) (V)
All stages rc 0 for all three branches (guard, g-cfg, g-suites, g-all, c-cfg, c-suites, run with game data, extra, tidy). Total wall: bundle-r3 497 s, mana-teleport 490 s, monsterdmg-pr 484 s (ccache hit 69 to 70 percent). The stage table does not print the warning counts; rc 0 of `extra` and `tidy` is the gate. Command: `the parity script --linux --bundle B --labels "bundle-r3 mana-teleport monsterdmg-pr"`.

## Damage-roll counted sets on bundle-r3 (emulator-free glue, base 005c7cab headers vs r3 headers) (V)
- mdmg2.py 20000 20261103: base differs (e.g. roll 159654 of 160000 frames, absorb life 102020 / pool 68173); r3: 0 differences in roll, reduce, absorb, both, negpct.
- v3_run.py 10000 314159: base differs (roll 79857 of 80000, reduce 39451, absorb life 27017 / pool 56202, both 70054 pool, exactred 7315); r3: 0 differences in roll, reduce, absorb, both, exact, exactred; "worse than base": none.

## NOT run (and why)
- Mana (base vs head), Teleport seed 9, v4 (real `Fight::absorb`): the build of the base/D1/head trees in WSL failed twice by my script (cmake without an explicit g++, apps off) and once by OOM (cc1plus killed) while three WSL jobs of different agents shared the 12 GB VM. The coordinator then ended the lane (the maintainer's #181 re-run covers it). The worker-host numbers of b2rebase2 (mana, Teleport seed 1) stand; Teleport seed 9 on the worker host was not run there either.
- Harness inputs (untracked, in the ops checkout) the ops checkout WSL harness directory: v3_*, v4_*, mdmg2*, pb2_*.sh, run-py.sh, b2r3_*.sh, sk_mana.py, sk_tele_spot.py; the runner script of that directory (tracked).

## Duration notes
MSVC four labels 12 min; WSL parity 25 min; the differential script ran into the queue behind other agents' Windows worker jobs (shared log files, then OOM) and needed 5 attempts.
