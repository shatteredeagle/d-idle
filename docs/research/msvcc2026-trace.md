# MSVC2026: `test_monster_ai` fails MSVC C2026 (string too big)

> status: research, 2026-11-02, agent msvcc2026 (card msvcc2026). Docs and `tools/emu/msvcc_*` only. Tags: V read (read in the source), V run (run on the worker host), I inferred, U unverified.

## Answer

- **Real, latent upstream problem, not an artefact of our toolchain (I, resting on V read + V run).** The Windows job of upstream CI builds the test targets, and four raw-string literals included by `test_monster_ai.cpp` are larger than MSVC's 16380-byte limit per literal piece.
- **Whether upstream CI is red on this is U**: `gh run list` was denied on this runner and no CI log is stored on the worker host. The Windows build worker's MSVC (14.44, VS 2022 17.14, `/W4 /permissive- /bigobj`) is the same family as the CI image's.
- **Minimal fix proven byte-identical (V run)**: split the long raw strings into adjacent raw literals of at most 12000 bytes. Four compiled constants, 0 mismatches. The private branch and the MSVC rebuild were NOT done (Limits).

## Questions

**Q1. Is the target built on the upstream Windows job?** Yes (V read).
- `.github/workflows/ci.yml`: matrix entry `windows-latest` / "Windows (MSVC)" (line 48); configure with `-DCMAKE_BUILD_TYPE=RelWithDebInfo` and no `-DD2D_BUILD_TESTS=OFF` (lines 218-228); step "Build" is `cmake --build build --config RelWithDebInfo -- -k 0` (line 232), i.e. the `all` target.
- `CMakeLists.txt` lines 121-124: `D2D_BUILD_TESTS` defaults ON (`_d2d_tests_default` ON unless `D2D_FLATPAK`); lines 137-139: `add_subdirectory(tests)` when ON.
- `tests/CMakeLists.txt` line 24: `file(GLOB_RECURSE ... test_*.cpp)` makes every test an executable, so `tests/monsters/test_monster_ai.cpp` is a target of `all`.
- Compile flags (`CMakeLists.txt` lines 71-77): `/W4` and `/bigobj`; nothing relaxes string limits.

**Q2. Which literals exceed the limit?** (V run, `tools/emu/msvcc_len.py` on the upstream-main tests directory, fork checkout)

| File | include site (`tests/monsters/test_monster_ai.cpp`) | raw literals | sizes (bytes) | over 16380 |
|---|---|---|---|---|
| `tests/pet_ais_cases.inc` | line 249, one `std::string_view` | 1 | 62345 | 1 |
| `tests/shadow_master_cases.inc` | line 610, `std::array<std::string_view, 3>` | 3 | 59449, 59285, 16762 | 3 |
| `tests/shadow_init_cases.inc` | (not in the reported errors) | 2 | max 13396 | 0 |

- Every other `*_cases.inc` in `tests/` (23 files) holds no raw literal and quoted lines of at most 508 bytes (`qmax` column of the script output): the `mon_ais*` files are one quoted line per array element and `test_monster_ai.cpp` line 371 even says "not constexpr: their lengths pass the step limit". So the author knew the compiler limits for those; the two big raw-string files are the only ones in the other style (I).
- The MSVC error lines (`shadow_master_cases.inc` 20, 36, 51, 80, 96, 111, 203; `pet_ais_cases.inc` 35, 68, 100) lie inside the oversize literals (line 203 is in the third literal of 205 lines; V read of the file). MSVC reports where the 16380th byte of each piece is passed (I; the exact reporting rule is U). The Windows build worker log is the source of these line numbers (S: the build-parity doc of the private ops repo, "FINDING at the BASE").
- The total of each literal stays below the 65535-byte limit MSVC applies after concatenation (62345 is the largest), so the split pieces concatenate to the same sizes (I: the 65535 limit is from MSVC documentation as remembered, not re-read here; U that 62345 passes after the split on MSVC 14.44).

**Q3. Is the fix byte-identical?** Yes (V run, `tools/emu/msvcc_split.py`, `tools/emu/msvcc_run.sh`, worker host).
- Rule: inside each `R"(...)"` split at line ends (the newline stays with the piece before it), greedily up to 12000 bytes, emit the pieces as adjacent literals separated by a newline outside the literal. Commas between literals and the comment line stay untouched.
- Result: pet 1 literal to 6 pieces (max 11977), shadow_master 3 to 13 pieces (max 11912) and, if one wanted every raw literal below the limit with margin, shadow_init 2 to 3 pieces (max 11933; not required).
- Proof 1 (python): the concatenated pieces of every original literal equal its bytes: 6 literals checked, 0 differences; SHA-256 prefixes of original vs split concatenation equal for all three files (edf81d10024e3a1c, ea3efa3bb76186ce, 5b46c4aebf0a5d3f).
- Proof 2 (compiled constants, g++ `-std=c++20`): a probe includes the original and split `.inc` side by side in the same shapes as the test (one `string_view`, one `array<string_view,3>`) and compares the compiled constants: 4 constants, lengths 62345, 59449, 59285, 16762 equal, FNV-1a hashes equal, **mismatches 0**.
- The patch is a mechanical rewrite of two files; no `.cpp` change, no behaviour change (the test text constants are the same bytes). The generator replaces the checked-in data, so also add a comment line in each `.inc` such as "pieces under MSVC's 16380-byte limit (C2026)" and make `tools/emu/shadow_master.py --dump` and the pet dump emit the pieces (U: those generator scripts were not read here; if they rewrite the files the split is lost).

## Deviation list (upstream vs MSVC, not game rules)

- **D1. Raw-string literals over 16380 bytes in `tests/pet_ais_cases.inc` and `tests/shadow_master_cases.inc`** (V read + V run for the sizes; the MSVC failure itself is S, the Windows build worker log). Rule on the upstream side: GCC and Clang accept them (the Linux and macOS CI jobs); MSVC does not. Differential: the g++ probe above covers the byte identity of the fix; the failure itself can only be confirmed by an MSVC build (Windows build worker via the ops repo's the parity script, gate permitting): U.
- **D2. Upstream Windows CI result at 3e073888 is unknown** (U). If it is green, either the CI MSVC treats raw literals differently (e.g. a newer compiler on the image) or the job is not required; if red, upstream already knows. Needed: `gh run list --repo <upstream> --workflow ci.yml` (GET), Windows job log, grep `C2026`.

## Bot findings

- **B1.** None: our bot has no copy of these test files and no MSVC target (the test files are upstream's; this repo has no `tests/*_cases.inc` of this kind). Not applicable.

## PR candidates

- **No PR candidate yet, nothing posted.** The brief allows an upstream-bound branch only for a real or latent upstream problem (this is one), but the MSVC verification and the throwaway worktree could not be done from this runner (Limits), and the policy here is private remote only, no PR. Patch description (for whoever creates `local/batch2-msvc2026` in a throwaway worktree under the lab checkout; numeric noreply identity for author and committer, subject at most 72 characters, body at most 3 lines, last line `Agent: Claude Code`, no co-author lines, private remote only, no force-push):
  - Subject: `tests: split raw string cases under MSVC's 16380-byte limit`
  - Body: `C2026 string too big in test_monster_ai (pet_ais, shadow_master).` / `Adjacent raw literals of at most 12000 bytes; same bytes (probe).` / `Agent: Claude Code`
  - Files: `tests/pet_ais_cases.inc`, `tests/shadow_master_cases.inc` replaced by the output of `tools/emu/msvcc_split.py` (scratch `msvcc/new/`).
  - Then: the parity script with a label for the branch from the ops repo (add a job per branch, see the build-parity doc), expect `test_monster_ai` to build and `warning C` count to stay 0.

## Limits and dead ends

- **No MSVC run.** The Windows build worker is reached over ssh; ssh is denied on this runner, so the parity script was not run (and its gate was not bypassed). The claim that MSVC fails is therefore S (the worker's earlier log), not V run here.
- **No git on the fork checkout from this runner**: `git -C` through `labrun` is denied and worktrees cannot be created, so the branch `local/batch2-msvc2026` does not exist. The generated files are in the worker scratch only (`/tmp/d2agent-msvcc2026/msvcc/new/`, not kept).
- **`gh run list` denied**: upstream CI status is U (D2). No CI logs are stored in the lab checkouts (searched the `pr/` and `fork/` output directories for the CI job name: only workflow files).
- **Dead end**: looking for an MSVC switch instead of a source fix. `/Zc:` has no option for the per-literal limit (I, from memory of the compiler's documented limits); not tried.
- The `shadow_init_cases.inc` literals (max 13396) are below the limit and are not part of the fix.

## Drill recipe (what to run later)

1. Windows build worker (gate allowing): ops repo the parity script (check mode), then a label for the split branch; expect `tests/monsters/test_monster_ai.cpp.obj` built, 0 `C2026`.
2. On the worker host: `tools/emu/run-on-worker.sh msvcc_run.sh` regenerates the split and prints the 4-constant comparison (expect `mismatches 0`).
3. For D2: `gh run list --repo <upstream> --workflow ci.yml --limit 20`, open a Windows job log from a run after `3e073888`, grep `C2026`.

## Matrix-cell and ledger updates

- No game rule is involved: no matrix doc row changes and `docs/coverage/ledger.tsv` is not touched (no rule row proven, so `tools/coverage/coverage.py --write` has nothing to regenerate).

## Glossary

- Missing term (for the report, not edited here): "C2026" (MSVC's per-literal string length error).

## Shared-file lines for the merger

See `openspec/changes/msvcc2026/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG).
