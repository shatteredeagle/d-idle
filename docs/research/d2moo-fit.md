# D2MOO: is it a better backend than d2d for us? (agent D2MOO-FIT, 2026-10-30)

Question: "look into D2MOO and is it better for us than d2d". Docs only; I made a shallow clone of
https://github.com/ThePhrozenKeep/D2MOO into /tmp (HEAD 8721f95, commit date 2026-10-30; deleted afterwards), read it, built nothing.
Tags: **VERIFIED** = I read it in the clone (path) or via the GitHub API (2026-10-30). **SECONDARY** = from our own docs or
other people's text. **UNSURE** = inference or not checked. Target classic 1.14d; D2R is not involved in D2MOO.
Licence rule from docs/decisions.md: nothing of D2MOO's code is copied here; everything is described in my own words.

## 0. Verdict in five lines

1. D2MOO is **not a standalone engine**. It is a Windows 32-bit C++ patch/replacement of the original game's DLLs (D2Common,
   D2Game, ...) that must run inside the original Game.exe. It cannot be hosted headless on Linux without a very large port.
2. It is **based on 1.10f**, not 1.14d. The 1.14d build option exists in CI but there are no 1.14d ordinal definitions or patches
   for D2Common/D2Game (only a debugger stub), so its 1.14d coverage is unproven.
3. It has no control channel, no server host, no main loop of its own, no network host, and its Storm/Fog layers are mostly
   stubs that call the original DLLs. Our needs (1)-(3) are not met; (4) is only partly (1.10f rules).
4. Licence is **MIT** (more permissive than d2d's GPL-3.0). Health is excellent right now (405 commits in Oct 2026).
5. **Recommendation: no as a backend (keep d2d); yes as a read-only formula/rules oracle** for cross-checking d2d patches.
   Concrete first payoff: it explains the tome price gap (section 6).

## 1. What D2MOO is (VERIFIED)

- README.md: "a re-implementation of the Diablo2 game coupled with patching facilities for modders ... we do not provide fixes
  for bugs (yet). The original game assets and binaries is required to use this project." Build deps: CMake, "Visual C++ (or any
  C++ compiler that can generate .DLLs on Windows)", the D2.Detours git submodule.
- README "Versions": "The project is currently based on the 1.10f version of the game. Feel free to submit patches for other versions."
- README FAQ: "Can I replace the game .dll files directly with the ones from D2MOO? NO! ... it only works with D2Common.dll (and is
  not bugfree)". Other DLLs are patched function by function through D2.Detours (`D2.Detours.patches/1.10f/{D2Common,D2Game,Fog}.patch.cpp`).
- The output is a set of DLLs (shared libs when the version is below 1.14, top-level CMakeLists line 29: `BUILD_SHARED_LIBS ON` only for
  `D2MOO_ORDINALS_VERSION < 1.14`; static libs otherwise) loaded by the original executable. Tooling around it: an imgui debugger
  (`-debug` flag), D2SE launcher ini generation (`cmake/D2MOO.D2SE.cmake`).

## 2. Source layout and completeness (VERIFIED, `source/`)

| Dir | .cpp files | Lines (cpp+h) | What it is |
|---|---|---|---|
| D2Game | 108 | 163 k | server game logic: AI (5 files incl. Baal), GAME (Game, SCmd/CCmd packets, Level, Event, Task, Arena, Clients), ITEMS, MISSILES, MONSTER (8), OBJECTS (4), PLAYER (11 incl. save, trade, pets), QUESTS (ACT1..ACT5 + Quests.cpp), SKILLS (one file per class + monster/item), UNIT (11) |
| D2Common | 138 | 156 k | rules and data: DataTbls (txt loading), Drlg (about 19 files: outdoors, maze, presets, per-biome), Path, Collision, Items, Monsters, Units (stats, FCR, speed), Skills, States, StatList, Inventory, QuestRecord |
| D2Win / D2Gfx / D2Lang / D2CMP / D2Sound | 20 / 7 / 9 / ... | 15 k / 3 k / 2.5 k | client UI/graphics (only partly); not needed for a headless server |
| Fog | 23 | 1.8 k | thin; `Fog.cpp` redefines `D2FUNC_DLL` as `D2FUNC_DLL_STUB`, i.e. calls into the ORIGINAL Fog.dll |
| Storm | 2 | 1.2 k | `Storm.cpp` stubs (`D2FUNC_DLL_NP_STUB`); `external/squall` (Lectem's unofficial Storm re-implementation, Unlicense, last push 2023-10-31 per API) is linked as `squall::Storm` |
| D2Net | 4 | 1.9 k | `Client.cpp`, `Server.cpp`, `D2Net.cpp` (size suggests a thin wrapper, UNSURE) |
| Game | 2 | 1.0 k | `Main.cpp`: command-line table and entry (credit to the galaxyhaxz/d2src base); D2MCPClient is a 34-line stub |

Completeness of D2Game (VERIFIED by grep): 134 `TODO` lines in D2Game, 1521 in D2Common (mostly "what is this field" notes), 0
`FIXME`, 0 "not implemented". Code for all five acts' quests, all classes' skills, missiles, monster AI, vendors
(`SUNITNPC_BuyItem`, gamble) and the game tick (`GAME_UpdateGamesProgress`, `GAME_ProcessNetworkMessages`) exists as
reconstructed functions. Some functions are kept as commented-out decompiler pseudo-code (e.g. `SUnitNpc.cpp` around lines
1367-1521), so completeness is about reconstructed coverage, not proven behaviour. UNSURE: percentage of ordinals actually
patched and verified; `D2.Detours.patches/1.10f/D2Common.patch.cpp` tracks it per ordinal (not counted).
Tests: doctest suites in D2Common/D2Lang (`source/*/tests`); the D2Common one is labelled `requires-original-dlls` in its
CMakeLists, so even the tests need the original game files.

## 3. Platform and build (VERIFIED)

- Top-level CMakeLists: warns "Diablo2 is 32bits only, 64bits build are unsupported. Invoke CMake with '-A Win32'".
- `CMakePresets.json`: presets DefaultGenerator and VS2022 (Visual Studio 17), ninja ("Ninja Multi-Config", needs the x86 native
  MSVC prompt per doc/AdvancedBuildAndRun.md), clang-tidy variants.
- `.github/workflows/main.yml`: `runs-on: windows-latest`, `ilammy/msvc-dev-cmd` arch x86, CXX=CL, matrix versions 1.00, 1.10f,
  1.14d (1.14d Release only, "unmaintained versions"). No Linux job.
- 175 of 350 .cpp files include `<Windows.h>`; `D2CommonDefinitions/CMakeLists.txt` says it cannot get rid of `<Windows.h>` in
  `D2BasicTypes.h`; the only clang mention is `clang-cl` (MSVC-compatible driver on Windows). Calling conventions
  `__stdcall`/`__fastcall` everywhere; D2Game links `winmm` (`timeGetTime`).
- Conclusion (VERIFIED for the facts, UNSURE for effort): a Linux build would need a portability layer for Win32 types and
  calling conventions, a 32-bit toolchain or a 64-bit port of struct layouts (static layout tests under `statictests/` pin the
  original 32-bit layouts), plus replacements for Storm, Fog, D2Net, D2Win, D2Gfx, D2Lang, D2CMP.

## 4. Can a headless server be hosted without the original executable?

What exists: server-side game logic (game objects, per-tick update, packet handlers, monsters, skills, items, quests, DRLG in
D2Common). What is missing for a standalone host:

| Missing piece | Evidence |
|---|---|
| a game-server host / main loop | `source/Game/Main.cpp` is the client command-line table (video, network, character options); `GAME_UpdateGamesProgress` is driven by the original server thread (UNSURE how it is started; the original does it) |
| MPQ loading + data tables | Storm is stubbed to the original; squall is a 2010-era reimplementation last touched 2023 (SECONDARY; may or may not read 1.14d MPQs, UNSURE) |
| Fog (memory, logging, excel parsing, async data) | redirects to original Fog.dll (`Fog.cpp` header) |
| network layer | D2Net 4 small files; the MCP client is a 34-line stub; the original's sockets would be used |
| player/save I/O host | `PlrSave*.cpp` reconstructed, but file I/O goes through an interface set by the host (ordinal `D2GAME_10029_SAVE_WriteFileInterface`) |
| any control channel | none: the only external hooks are DLL ordinals (D2Game.1.10f.def lists ordinals from 10001), D2.Detours patches and the in-process imgui debugger (`-debug` / `D2_DEBUGGER=1`) |

So the realistic way to "run" D2MOO's rules is inside the original Windows stack (under Wine, i.e. what we already do with
Game.exe), or by writing a new host that supplies about the six missing pieces above. That is a multi-month engine project, not an
adapter.

## 5. Licence, activity, users (VERIFIED, GitHub API and repo, 2026-10-30)

- `LICENSE`: **MIT**, "Copyright (c) 2020-2025 The Phrozen Keep community" (API: spdx MIT). Copying would be allowed with the
  notice, but our rule stays "describe in own words"; MIT removes the GPL-offer-source worry that d2d has.
  Submodules: D2.Detours (Lectem), doctest, imgui, stb, squall (Unlicense). UNSURE: licence of D2.Detours.
- Activity: pushed 2026-10-30; created 2020-12-25; 196 stars, 61 forks, 26 open issues, no GitHub releases; commits since
  2026-04-30 (git log): 430 (Jul 6, Aug 5, Sep 14, Oct 405). Top authors in that window: ResurrectedTrader 181, Mw95 173,
  Clement Gregoire (Lectem) 39. All-time top contributors (API): Lectem 980, Mw95 205, ResurrectedTrader 182, IAmTrial 123,
  AudriusButkevicius 44. The October burst is almost entirely "name locals and parameters" renames and no-op cleanups (commit
  subjects), i.e. readability work, not new features.
- Community: Phrozen Keep forums and Discord (README). Users: README targets modders (D2SE, D2.Detours patches); I found no
  documented use as a server for D2GS/PD2 (UNSURE, did not search outside the repo). "No bug fixes (yet)" means it is
  intentionally bug-compatible with the original, which is good for fidelity.
- 1.14d coverage: CI builds `1.14d` Release; `D2.Detours.patches/1.14d/` holds only D2Debugger files; `source/*/definitions/` has
  only `*.1.10f.def` ordinal files; D2Game has only about 19 version `#if` sites in total. Uber content (1.11+) is behind the
  off-by-default `D2MOO_WITH_UBERS`. VERIFIED for the files; whether D2Game semantics differ between 1.10f and 1.14d beyond
  those ifdefs is UNSURE.

## 6. What we can still get cheaply (oracle uses)

D2MOO is MIT-licensed readable source for the original rules (SECONDARY: our docs/sme/cast-timing-sources.md already used
Units.cpp as the strongest cast-speed source). Uses:

1. **Tome of Town Portal price (open d2d gap (a), docs/d2d-gaps.md).** `D2Common/src/Items/Items.cpp`,
   `ITEMS_CalculateTransactionCost` (starts about line 2060): for items of type book it sets the cost to the base cost plus
   quantity times the per-charge cost from `books.txt` (VERIFIED in the code near lines 2160-2165, 1.10f). That matches our
   live 450 vs d2d 250 hypothesis (a charge-dependent book term that d2d omits). The per-charge value is data, readable with our
   `mpq-cat`, so the rule can be confirmed offline without a live purchase (UNSURE: compute with the real numbers; the
   multiplier order is in the same function).
2. **Cast / animation speed** (gap (b)): already used; keep Units.cpp (`UNITS_IsSeqAnimSpeedModulatedByFCR`, FCR formula) as the
   reference for the d2d patch; it shows integer arithmetic and the Druid 208 base.
3. **Other checks**: monster AI target choice (`D2Game/src/AI`, `MONSTER`), skill damage/mana (`SKILLS`), drops (D2Common
   Items/Monsters), DRLG presets (D2Common/Drlg), vendor logic (`SUnitNpc.cpp`). Method: read the function, restate the formula
   in a table with a tag, compare with d2d's rules at the same point. About 1-2 agent-hours per mechanic.
4. **d2d's `tools/emu` oracle idea**: D2MOO cannot be run headless here, so use it as a *paper* oracle, not a differential test
   runner. A differential runner would need D2MOO's D2Common built and hosted (section 4: not available).
5. Caveat: the source is 1.10f. Where 1.11-1.14d changed rules (item/runeword balance, skill tweaks), D2MOO is the wrong
   reference; our Game.exe measurements and the 1.14d MPQ data win. Tag every cross-check "1.10f source".

## 7. Comparison with d2d on our six needs

Our d2d facts are SECONDARY (docs/d2d-spike.md, d2d-backend.md, d2d-fidelity.md, d2d-gaps.md; not re-verified now).

| # | Need | d2d | D2MOO | Verdict (confidence) |
|---|---|---|---|---|
| 1 | Headless, deterministic, 100x+ real time sim | yes: headless mode, lockstep 120-377x, deterministic, five acts | no: needs the original game processes; Storm/Fog stubbed to original DLLs; no time-scale loop; would need a new host | **d2d** (high) |
| 2 | Control channel for state read / actions | yes: devctl (about 60 % of what we read; patch set planned) | none (in-process DLL, ordinals, imgui debugger); we could write one but only inside Game.exe | **d2d** (high) |
| 3 | Linux container on the NAS | builds and runs on Linux | Windows-only 32-bit MSVC, CI windows-latest, Windows.h in half the files; large port | **d2d** (high) |
| 4 | Fidelity to original 1.14d within 20 % | measured against 1.14d; approximations documented; cast timing and prices needed fixes | closest to original CODE (bug-compatible) but 1.10f base, no 1.14d ordinals, unverified end to end | **D2MOO for formulas, d2d for a 1.14d-measured whole** (medium) |
| 5 | Licence | GPL-3.0; private fork fine until we ship a binary/container, then offer source | MIT | **D2MOO** (high) |
| 6 | Maintenance | active (per our earlier research) | very active (430 commits/6 mo, 3 main authors, Discord) but renames, no releases, 1.14d not a goal | **tie** (medium) |

Score: d2d wins 1-3 (the hard requirements); D2MOO wins licence and is the better formula reference.

## 8. Recommendation: no as a backend, yes as a paper oracle

- **Do not switch to or add D2MOO as a backend.** It cannot satisfy needs 1-3; "product backend on the NAS" would mean writing a
  new engine host and porting to 64-bit Linux.
- **Use alongside for verification only**: cross-check d2d patches and our fidelity gate against D2MOO functions (tome price, cast
  frames, drops, AI wake/aggro). Prefer live measurements on Game.exe when they disagree (D2MOO is 1.10f).
- Effort: oracle checks about 1-2 agent-hours per mechanic. A D2MOO-based headless backend: UNSURE, my estimate 3-6 agent-months
  (Win32/Linux portability, Storm/Fog/MPQ/D2Net replacements, server host, control channel, 1.14d fidelity work), far more than
  finishing the d2d patch set (4-6 agent-days per docs/d2d-backend.md).
- Main risks if someone tries anyway: (a) 32-bit layout assumptions vs 64-bit Linux; (b) Storm/Fog stubbed to original DLLs, so
  the data path is untested off Windows; (c) 1.14d semantics not reconstructed; (d) no upstream appetite for headless Linux
  (it is a modding/patching project); (e) reconstructed functions are not proven correct.

## 9. Could not verify

- Whether D2Game's tick runs correctly without Game.exe, or the exact list of what the original host supplies.
- How many ordinals are patched/verified (D2Common.patch.cpp status not counted) and real D2Game test coverage.
- Real `books.txt` per-charge numbers and whether the 450 live price follows from the rule in 6.1.
- D2MOO 1.14d behaviour (nothing was built; only repo files and the CI matrix were read). D2.Detours licence. Real-world users
  (D2GS, PD2).
- Commit history beyond the 50-commit shallow clone (counts above are from the clone's `git log --since`, plus the API).
