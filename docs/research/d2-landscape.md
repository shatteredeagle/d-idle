# Diablo II reimplementation / decompilation / hosting landscape (agent D2LANDSCAPE, 2026-10-30)

Question (user): are there other decompilation projects that fit us better than d2d (psi29a/d2decomped) or D2MOO
(D2MOO is analysed in depth by agent D2MOO-FIT; here it only gets a row). Scope: classic D2 LoD, mainly 1.14d, 1.13c, 1.10f.

Tags: **VERIFIED** = read by me at the URL on 2026-10-30 (GitHub API metadata, README or file; dates and counts are as of that day).
**SECONDARY** = search snippet or someone else's description. **UNSURE** = memory or inference. Nothing was built or run: no
project below was executed by me, so every "runs headless / is deterministic / is faithful" statement is the project's own claim
unless stated. Nothing was copied into this repo. Our context read first: docs/d2d-spike.md, d2d-fork.md, d2d-fidelity.md, d2d-gaps.md,
adr/0002, spec backend-contract (and fidelity-gate: 20 % rule, docs/d2d-backend.md section 7).

## 1. Verdict in five lines

1. **No project is a better drop-in replacement for d2d as the fast, deterministic rules simulator.** d2d is the only one that plays all
   five acts, is checked against Game.exe by its author, builds on Linux, and (with our lockstep fork) runs 90x to 370x real time with a control socket.
2. **Strongest complement: jaenster/d2-dedicated-server** (MIT, Zig): runs the ORIGINAL 1.14d game logic headless in Linux containers
   (Game.exe under wine32, or the Mac build natively, no wine, no display). That is the fidelity oracle and a possible no-GUI product
   backend, but real-time only, needs a realm stack (redis/postgres) and a network client, and is one developer's three-month-old work.
3. **Second complement: jaenster/libd2** (MIT source, Zig, Go bindings): a map generator checked cell for cell against the retail engine
   (11.1M cells per seed), plus item drops; its rules engine is NOT validated against the original. Use as an independent map/drop oracle.
4. **Specs to cross-check d2d's three open gaps**: IgorKonovalov/diablo-2-recompilation (docs only, tagged claims, RNG call order).
   Its economy spec already names a tome price term d2d lacks (section 5). Prior art for our reader: Kolbot/d2bs (injected DLL calling the
   game's own functions) and the Nugwa628 codex (Game.exe function maps, 4,100 behaviour checks; read-only, licence unclear).
5. **Switching cost**: to any engine other than d2d the bot's backend (d2dctl/d2dinput/d2dsrc) is rewritten, about 2 to 4 weeks for a
   packet-level backend (UNSURE, my estimate), and the lockstep fork would have no equivalent (no fast-forward). Not recommended now.

## 2. Survey table (every project found)

Counts from the GitHub API on 2026-10-30 (VERIFIED unless noted). "Contrib" = contributors returned by the API (cap 100).
Fidelity column: how the project itself validates against the original.

| Project | Lang | Licence | Last push | Stars / contrib | Target | Status and coverage | Oracle / tests vs original |
|---|---|---|---|---|---|---|---|
| **psi29a/d2decomped (d2d)** https://github.com/psi29a/d2decomped | C++ | GPL-3.0 | 2026-10-30 | 24 / 4 (2 real: 1275 and 79 commits); created 2026-09-26 | 1.14d | All 5 acts, 27 quests, NM/Hell, SP; client + devctl socket; dedicated server "not started" | author says drops, AI, placement "checked against the original"; ours: docs/d2d-fidelity.md (3 gaps) |
| **ThePhrozenKeep/D2MOO** https://github.com/ThePhrozenKeep/D2MOO | C++ | MIT | 2026-10-30 | 196 / 18 | 1.10f (README) | Reimplementation of the game DLLs as DLL replacements plus patching (D2.Detours), Windows build; ordinal-by-ordinal verification "slow process" | patched ordinal by ordinal into the real game; no standalone sim. In-depth: D2MOO-FIT |
| **jaenster/d2-dedicated-server** https://github.com/jaenster/d2-dedicated-server | Zig | MIT | 2026-10-29 | 31 / 1; created 2026-06-05 | 1.14d (+1.06b..1.13c via game DLLs) | Realm (BNCS/MCP/BNFTP) + headless game servers hosting the original engine; Helm chart, e2e suite | runs the original code, so fidelity is by construction; 34 clientless e2e scenarios |
| **jaenster/libd2** https://github.com/jaenster/libd2 | Zig (+ Rust, .NET, npm/wasm, Go, C bindings) | MIT for source, "NOASSERTION" on GitHub; baked blobs derived from Blizzard data | 2026-10-30 | 21 / 1; created 2026-07-02 | 1.14d | DRLG, item drops, d2s read/write, D2GS packets, pathfinding, "game" rules engine (combat, skills, missiles, monster AI, GameInstance) | DRLG cell-exact vs retail dumps; save byte-exact; item "traced through the binary"; game/net NOT checked against a live server (its docs/VERIFICATION.md) |
| **jaenster/d2-clientless** https://github.com/jaenster/d2-clientless | Zig | none stated | 2026-10-26 | 12 / 1 | 1.14d | Clientless Battle.net/MCP/D2GS client, parses the live S2C packet stream; test client of the dedicated server | validated against real Battle.net (its claim) |
| **jaenster/aether** https://github.com/jaenster/aether | C++/Zig/TS | none | 2026-10-19 | 6 / 2 | 1.14d | Scriptable bot framework: injected DLL, SpiderMonkey 60, TS scripts, WebSocket daemon | n/a (bot) |
| **IgorKonovalov/diablo-2-recompilation** https://github.com/IgorKonovalov/diablo-2-recompilation | Python (docs) | none found (no LICENSE: API 404) | 2026-10-27 | 8 / 1; created 2026-10-11 | 1.14d | Documentation only, no code: specs for RNG, items, monsters, DRLG, combat, skills, missiles, anim, quests, economy; "monsters/ai" not started | each claim tagged by evidence and tied to runs on the real game; checkers in repo |
| **Nugwa628/d2-114d-codex** https://github.com/Nugwa628/d2-114d-codex | asm/py | NOASSERTION; ships Ghidra decompiler output of Game.exe | 2026-10-30 | 1 / 0; created 2026-10-30 | 1.14d | ~260 specs, 13,085 decompiled functions, ~4,100 recorded behaviour checks, a real-Game.exe-under-Wine-on-Xvfb runner | behaviour checks recorded on the real game |
| **gravestench/dark-magic** https://github.com/gravestench/dark-magic | Go + Lua | MIT | 2026-10-30 | 20 / 4; 1500 commits by one; created 2023-07 | 1.13c and 1.14d (classic mode out of scope) | Clean-room rewrite, Go engine, "one authoritative renderer-independent deterministic simulation", client/server/realm, labs (combat, missile, mapgen); explicitly not complete; largely LLM-written with maintainer review (its README) | "evidence-driven"; data and probes pinned to 1.14d; no numbers vs original seen |
| **Virtastic/d2-web** https://github.com/Virtastic/d2-web | C++ to WASM | MIT | 2026-10-30 | 1 / 1 | 1.13c | Browser engine, "early alpha", Act 1 only | none stated |
| **OpenDiablo2/OpenDiablo2** https://github.com/OpenDiablo2/OpenDiablo2 | Go | GPL-3.0 | 2021-10-21, **archived** | 11,107 / 73 | LoD (1.10-era data) | Renderer, UI, first-act movement; split into AbyssEngine; no real game rules | none |
| **AbyssEngine** https://github.com/AbyssEngine/AbyssEngine | C | MIT | 2024-04-21 | 324 | n/a | Clean-room engine, explicitly not save/multiplayer compatible | none (not an accuracy project) |
| **eezstreet/OpenD2** https://github.com/eezstreet/OpenD2 | C | GPL-3.0 | 2021-05-31 | 604 / 7 | 1.10-1.13 era | Menu and TCP/IP lobby, gamecode "still being written", stalls on loading | none; stale |
| **collinsmith/riiablo** https://github.com/collinsmith/riiablo | Java | Apache-2.0 | 2025-01-13 | 1,109 / 2 | LoD assets | Playable remake with own changes (auto gold pickup) | not a fidelity project |
| mofr/Diablerie (Unity), FooSoft/lazarus (Go, archived), Quest/other renderers | C#, Go | MIT | 2022 | 1,242 / 8 | 1.14-ish (SECONDARY) | Renderers/early engines, abandoned | none |
| **pvpgn/d2gs109** https://github.com/pvpgn/d2gs109 and tesseract2048/d2gs, espenmjos/d2gs | C, Shell | other / MIT | 2018 / 2015 / 2026 | 42 / 95 / 25 | 1.09d, 1.13c | The classic D2GS: an exe that loads the ORIGINAL D2Game.dll to host games; pvpgn-server (GPL-2.0, 2026-08-17, 650 stars) is the realm | original code by construction; real-time, wine or Windows |
| **blacha/diablo2** https://github.com/blacha/diablo2 | TS | MIT | 2023-03-06 | 229 / 5 | 1.13c (D2 classic) + D2R | `@diablo2/map`: map dumps by driving the ORIGINAL D2 DLLs (d2-map.exe, Docker), packet sniffer, Huffman, bintools | the original is the generator |
| **blizzhackers/kolbot (+D2BS)** https://github.com/blizzhackers/kolbot | JS | none stated (repo has a README, API shows no licence) | 2026-10-21 | 294 / 49 | 1.13c/1.13d, 1.14d (SoloPlay patch.json names 1.14d, SECONDARY) | Injected DLL (D2BS, SpiderMonkey) + JS bot library; D2Bot manager; the main bot ecosystem | n/a, it is the original game |
| noah-/d2bs https://github.com/noah-/d2bs | C++ | none shown | 2021-04-06 | 118 | 1.13 | D2BS source: 240 pointer macros into D2Client/D2Common/D2Game... | n/a |
| ChaosMarc/PlugY, theBGuy/PlugY4Kolbot | C++ | GPL-3.0 | 2022 / 2026-10-24 | 230 / 5 | 1.13, 1.14d fork for botting | DLL-injection mod: shared stash, multi-instance, bot stash protocol | n/a |
| bolrog/d2dx, SlashGaming | C++ | GPL-3.0 / other | 2023 / 2026-09 | 551 / 145 | 1.14d | Renderer and resolution patches | n/a |
| dzik87/D2GridraTools | Python | none | 2022-01-26 | 10 | 1.14d | Ghidra scripts for 1.14d | n/a |
| nokka/d2s (Go, MIT, 2022), daidodo/diablo_edit (C++, MIT), halbu (Rust, D2R) | | | | | | .d2s parsers/editors | n/a |
| **DevilutionX** (diasurgical) | C++ | NOASSERTION (sustainable-use style) | 2026-10-28 | 9,966 | **Diablo 1** | Out of scope; the model of a successful faithful port | n/a |
| **d2r-mobile** (two repos, 4,896 and 342 stars), D2R-3D, d2rmm, MythosLoader, D2RMH | C/C++/TS | MIT/other | 2026-10 | | **D2R only** | Compatibility layers, mod managers, map reveal. **D2R is a different game**, not our target | n/a |
| Project Diablo 2 / Median XL tooling (pd2-tools, murkl/d2launcher, PD2 wiki) | TS/Shell | MIT | 2026-10 | 138 / 55 | mods | Trade, launchers, DB. Out of scope: no server code is public that we found | n/a |

Not found (VERIFIED by search, which is weak evidence): any other 1.14d decompilation that builds a runnable game; any Rust or Python engine
with real rules; any Reddit/HN discussion indexed for these (the web search returned none, so the "forums" part of the brief is NOT covered).

## 3. Per-project notes that matter to us

### 3.1 jaenster/d2-dedicated-server (the surprise)
- VERIFIED (README, STATUS.md, PERFORMANCE.md, native-vs-wine.md): `d2gs:1.14d` runs the retail 1.14d Game.exe under wine32 with a d2gs
  patched in (dbghelp proxy DLL, `--loaddll`), headless, no display; `d2gs:1.14d-native` runs the **macOS i386 Mach-O build of 1.14d** mapped
  directly on i386 Linux (no wine, no emulation), 4.4 MB image, about 22 MiB resident, 0.2 percent CPU idle; older versions (1.06b to 1.13c) use the
  game's own DLLs through a Zig host (`apps/d2host`) that supplies Fog.dll and D2Net.dll. All serve real clients end to end (login, character, create, join, "IN GAME").
- VERIFIED: tick loop is frame-paced (40 ms budget per game, `QSERVER_TickAllGames`); the original main loop "busy-spins by default" and
  d2gs paces it. Nothing in the docs mentions a fast-forward or virtual clock: **real time only** as far as I read (UNSURE: not searched in code).
- VERIFIED: up to 7 games per Game.exe (Fog pool limit); the native build ships capped at 1 game ("about half of a round's games are admitted" when raised, unexplained).
- VERIFIED: requires redis and postgres ("no filesystem mode"), a realm (`realmd`, our-own-BNCS), and a client that speaks D2GS. Characters come from redis
  (`D2GS_CHAR_SOURCE=memory|file` exists for native). A full Helm chart and compose files exist.
- Control channel = the game's own network protocol. `jaenster/d2-clientless` (Zig, 12 stars) speaks BNCS/MCP/D2GS with no game binary and receives the live
  packet stream ("character assignment, act load, unit spawns, stat updates, quests"). libd2 `net` and `client` packages (Go bindings via purego) decode and model it.
- Licence: MIT code. It needs Blizzard files (Game.exe and data, or the Mac Mach-O 1.14d binary: **we only have the Windows install, UNSURE where the
  Mac binary comes from**). Our use would be private, our own copy.
- Health: 1 contributor, created 2026-06-05, active daily, sponsor-funded, no stars to speak of. Bus factor 1.
- Why it matters: it is the first public way to run the **original** simulation without a GUI, Xvfb, input injection or memory scraping.
  That removes most of what makes our Wine container heavy (ADR 0002 context), but swaps it for a packet protocol and a realm.

### 3.2 jaenster/libd2
- VERIFIED: map generation compared cell for cell with retail dumps (seeds 1, 2, 17, 18, 777 all Nightmare; 2 blind holdout seeds; per-level CRC over
  200 seeds x 3 difficulties; zero differing cells). Normal and Hell are only CRC-checked per level.
- VERIFIED: `game` (rules engine) "checked against the rules and tables read out of the binary, not against a live server"; determinism only
  unit-tested; the planned `srvtrace` golden (log real DAMAGE inputs and outputs, replay) is a "standing follow-up". `net`: not against a live server.
- VERIFIED: Go bindings (`go get github.com/libd2/go`, Go 1.24, no cgo, purego, one native library per platform). Hosted DRLG endpoint exists
  (https://libd2.typeguru.nl, not used by me). Pure Zig, cross-compiles to Linux/wasm.
- Licence: MIT source; embedded blobs are Blizzard-derived (its own note). Private use is fine; do not redistribute the blobs.
- A real, fast (seed in, whole act out in under a second) map oracle, usable from Go. It does NOT replace d2d's monster/skill simulation.

### 3.3 d2d (baseline, for contrast)
- VERIFIED (README): all five acts and 27 quests play through; "Multiplayer with a d2d host" and "Dedicated server" not started; can JOIN an original-hosted game (Acts I to IV
  checked live). 1,275 of about 1,360 commits are one author, created 2026-09-26, so a month old. Our docs: lockstep fork 90x to 370x, deterministic with `--seed`,
  but monster HP/AI rolls differed per run in the unpatched build (d2d-spike section 3; the fork's `monsters` output was byte-identical).
- Open fidelity gaps (docs/d2d-gaps.md): tome price 250 vs 450, Sorc cast period 14 vs 13/19, wake radius. Section 5 shows how an outside spec can settle them.

### 3.4 gravestench/dark-magic (the other Go one)
- VERIFIED: targets 1.13c and 1.14d parity, one deterministic renderer-independent simulation shared by offline, listen, dedicated and realm play,
  Go engine with Lua rule policy, MIT, PostgreSQL realm, QUIC/UDP. Three years old but most activity looks recent; maintainer says it is not complete and is built mostly by LLM agents under review.
- It is the closest philosophical match (Go, deterministic, headless server, same toolchain as our bot) and the only one that promises replay/checkpoint.
  UNSURE how much gameplay exists; I did not read ROADMAP/issues. If it matures it could host our bot in-process (same language), but nothing in my reading shows
  numbers against the original, so it cannot meet the 20 percent gate today without our own measurement.

### 3.5 OpenDiablo2 / OpenD2 / riiablo / AbyssEngine / lazarus
- VERIFIED stale or not accuracy projects: OpenDiablo2 archived 2021 (Go, GPL, renderer-first), OpenD2 stalled 2021, AbyssEngine "not a reverse-engineering project",
  no save/multiplayer compatibility, riiablo (Java) changes gameplay on purpose. None simulates the real rules. Verdict: no.

### 3.6 D2MOO (placement only)
- VERIFIED: MIT, 18 contributors, pushed today, "based on the 1.10f version", reimplements the game DLLs and patches them into the original (D2.Detours);
  "we do not provide fixes for bugs"; build is Visual C++/Windows ("CMake presets"). Not a standalone runnable game; fidelity high where ordinals are done. See D2MOO-FIT.
- SECONDARY (search): d2emu.com's chest calculator is said to run the game's drop code and credit D2MOO. I could not verify this (the page lists nothing).

### 3.7 D2GS (PvPGN scene) and D2BS/Kolbot (prior art)
- D2GS: SECONDARY/known: `D2GS.exe` loads the original D2Game.dll and friends as a Battle.net game server; this is what jaenster's d2gs descends from
  (pvpgn/d2gs109 1.09d, tesseract2048/d2gs 1.13c, espenmjos/d2gs Docker/Ubuntu wine). It is a proven way to run the real logic headless, but real-time and needs the realm.
- Kolbot/D2BS: VERIFIED (noah-/d2bs D2Ptrs.h): an injected DLL that declares about 240 pointers (`FUNCPTR`/`VARPTR`/`ASMPTR`) as "DLL number + offset" into D2Client, D2Common,
  D2Game, D2Win, D2Net, Fog, Storm... and **calls the game's own functions** from a JavaScript (SpiderMonkey) API (`getUnit`, `getPath`, `getCollision`, `getRoom`, `getSkillByName`, ...).
  Kolbot adds 1.14d patch entries (SECONDARY, snippet: SoloPlay `patch.json` with "Version":"1.14d"); the D2Bot manager handles multi-client orchestration; PlugY4Kolbot adds
  multi-instance and a stash protocol. Aether (jaenster) is the 2026 TypeScript version on 1.14d.

## 4. Comparison against OUR six needs

Legend: Y = meets, P = partial or unproven, N = no, ? = could not establish. The "d2d (ours)" column is the baseline from our own docs.

| Project | (1) fast deterministic sim of real rules | (2) control channel | (3) Linux container | (4) fidelity to 1.14d, 20 % | (5) licence vs private code | (6) maintenance |
|---|---|---|---|---|---|---|
| **d2d + our lockstep fork** | **Y** 90-370x, seed-deterministic (V-d2d) | **Y** devctl socket, `dump` verb | Y native, headless, no Wine | P: 3 gaps, only 3 of many systems measured | P: GPL-3.0, fork stays private, no code in our repo (ADR 0003) | P: one author, 1 month old but 1,275 commits; we carry a patch |
| d2-dedicated-server (original engine) | N real time only (UNSURE, not searched in code); 7 games per process | P: D2GS packets via a clientless; no state-dump verb; we write the client | **Y** (wine32 image, or native i386 image); needs redis+postgres+realm | **Y** by construction (it IS Game.exe or its Mac twin) | **Y** MIT code; needs Blizzard files | P: 1 developer, 5 months, daily commits |
| libd2 (Zig) | P: DRLG yes; `game` sim "unit-tested deterministic"; speed unmeasured | P: C ABI + Go bindings, library not a process | Y (pure Zig, wasm) | **Y for maps** (cell-exact), P for items, N-unproven for combat/AI | Y MIT (+ Blizzard-derived blobs, private use) | P: 1 dev, 4 months |
| dark-magic | P: designed deterministic, completeness unknown | ? (server/realm exist; no state API read) | Y Go | ? no measurements seen | Y MIT | P: 1 dev + LLM agents, 1500 commits |
| D2MOO | N: patches the running original; Windows | N | N (Windows DLLs, MSVC) | Y where ordinals are done | Y MIT | Y 18 contributors, active |
| OpenDiablo2 / OpenD2 / riiablo / Abyss | N | N | Y | N | GPL / Apache / MIT | N archived or stalled |
| IgorKonovalov specs, Nugwa628 codex | n/a (documents) | n/a | n/a | **Y as a reference**: tagged claims, behaviour checks | docs without licence: read, do not copy | P: 1 dev each, weeks old |
| Kolbot/D2BS/Aether | N (original game, real time) | Y in-process JS API on the real game | N (Windows DLL, wine) | Y (is the original) | licence unclear, injected code | Y Kolbot 49 contributors |
| blacha map (original DLLs) | N | P CLI dump | P docker/wine | Y for maps (1.13c) | Y MIT | N last push 2023 |

Ranking for our purposes (best fit first):
1. **d2d (+fork)**: stays the fast bench and primary backend candidate. No other project offers needs (1)+(2)+(3) together.
2. **jaenster/d2-dedicated-server**: best complement; the original engine as oracle and as an alternative no-GUI container backend. Worth a spike.
3. **jaenster/libd2**: independent map and drop oracle in Go; also a reference implementation of the D2GS protocol.
4. **IgorKonovalov spec (and the Nugwa628 codex as read-only reference)**: source of "expected values with evidence" for our fidelity gaps.
5. **dark-magic**: watch; revisit when it has a measured parity claim.
6. **Kolbot/D2BS/Aether**: prior art for the Game.exe backend, not a backend for us.
7. D2MOO (other agent), then the stale engines (OpenDiablo2, OpenD2, riiablo, Abyss): no.

## 5. What each complement would give us, concretely

**a) Settling d2d's open gaps with an outside spec (cheap, do first).** I read two files of IgorKonovalov's spec (VERIFIED at
https://github.com/IgorKonovalov/diablo-2-recompilation/blob/HEAD/docs/specs/items/economy.md, tags in that repo: [C] confirmed on a run, [I] inferred):
- Tome of Town Portal price (gap a): the price function has a book branch, `P = Books[suffix 0].CostPerCharge x q + base.cost` (item type 18), and step D4 explicitly
  skips the quantity multiply for books. That is exactly the term d2d lacks (d2d-gaps.md (a): "no tome branch anywhere"). The tag in that spec is [I] (read from code, no
  priced tome in the logged run: "ear, body-part and tome bases were not priced in the run"), so it is a lead, not a confirmation. Confirm with one live purchase of 2 tomes (already proposed in d2d-gaps.md).
- Cast timing (gap b): `anim/timing.md` has the cast factor cap (175, confirmed) and the per-mode rate machinery; I did not extract the Sorceress numbers (UNSURE). Read it before patching the fork's `set_pmode`.
- Wake radius (gap c): the spec's `monsters/ai` is "not started", so no help there.

**b) Original engine as oracle (medium effort).** Run `d2gs:1.14d` (wine32) once in a container on the worker host with a Windows-installed 1.14d, drive one character with the
clientless through a scripted scenario (walk, kill Blood Moor, buy at Akara, die) and compare the resulting `.d2s`/packet stats with d2d's `dump`. That gives
numbers from the real engine without our Xvfb+memory reader and without opening a game window. Estimated 2 to 4 days (UNSURE). Real time, so it cannot replace soak speed.

**c) libd2 DRLG as map oracle (small).** Generate a seed's act, compare walkability/exits with d2d's `--seed` levels and with what our reader sees. Needs only the Go
module and our MPQ data (UNSURE whether it reads our MPQs or only embedded tables; it embeds "1.14d excel tables").

**d) Prior art for our memory reader (Kolbot/D2BS).** Our reader scrapes Game.exe memory from outside (docs/d2-1.14d-memory.md). D2BS instead lives in-process and calls the engine's own getters
(D2CLIENT_GetPlayerUnit, D2COMMON stat and path functions, `getCollision`, `getRoom`, `getPath`). Lessons, tagged:
- VERIFIED: the game keeps per-module function/variable tables addressed by "DLL id + offset"; for 1.14d everything is inside Game.exe, so offsets are single-binary addresses (jaenster aether/codex say so for 1.14d: "All of the game's code is in that one executable"; VERIFIED at the codex README).
- The codex's `re/exports-typed` (13,085 decompiled functions, about 1,250 named) is a ready address map for struct fields we read. **Legal: it ships decompiler output of Game.exe; licence NOASSERTION. Use it as a lookup for addresses and structure layouts only, never copy text or code, same rule as d2d (ADR 0003).**
- Kolbot's `getPath`/`getCollision` show the engine already has an authoritative collision and pathing API we could prefer to our own tracker (UNSURE if reachable from outside the process).
- A clientless packet client (jaenster) shows another channel that needs no memory reading at all (S2C stream: unit spawns, stat updates). Not usable on the Game.exe client for us, but usable on a d2gs server.

## 6. What switching would cost (honest)

| Path | What we rewrite | What we lose | Estimate (UNSURE) |
|---|---|---|---|
| Stay on d2d | nothing; fix the 3 gaps in the fork | n/a | 1 to 3 days for the gaps |
| Add d2gs original engine as a third backend (oracle) | a StateReader and `input.Controller` over D2GS (`net` + clientless logic; libd2 has Go bindings for decode) | the lockstep clock (no fast-forward) | spike 3 to 5 days; full backend 2 to 4 weeks |
| Replace d2d by d2gs for product | all of the above plus redis/postgres/realm in the NAS stack, character provisioning | speed of tests, our fork's `dump` | not recommended while real-time only |
| libd2 `game` as sim | a Go adapter over a C ABI; unknown completeness | proven fidelity (it has none yet for combat) | not recommended |
| dark-magic | unknown; would be a new `Env` backend | unknown | wait |

Because our decision code only talks to `env.Env` (spec backend-contract), a new backend is an adapter, not a rewrite of behaviours. The main structural fact: today's d2d backend uses
a rich text socket with a `dump` verb we added; the original engine offers packets, where e.g. exact stat values, item properties and the merc state arrive as client-side
messages (what the client knows, not the server's state). Whether that is enough for our StateReader methods (`UnitStats`, `ReadStatList`) is UNSURE.

## 7. Recommendation

1. Keep d2d as the primary fast backend (ADR 0002 stands; no better fit exists). Fix gaps a and b in the private fork when convenient.
2. Open a small spike "d2gs-oracle": get `d2gs:1.14d` and the clientless running on the worker host (docker, wine32 image) and measure (i) whether the server can be paced faster than 40 ms per tick,
   (ii) the packet stream's richness vs our StateReader, (iii) a Blood Moor XP/drop run against d2d's numbers. If (i) is "no", it stays an oracle only. This is the one finding that could change the product picture
   (no Xvfb, no input injection, no memory scraping, MIT licence instead of GPL).
3. Use libd2 DRLG as a cheap map oracle and keep an eye on its `game` package; use Igor's specs as tagged expectations in docs/d2d-fidelity.md.
4. Do not integrate OpenDiablo2, OpenD2, riiablo, AbyssEngine, Diablerie, lazarus.
5. Watch dark-magic; a measured parity claim would make it interesting (same language).

## 8. What I could not verify

- No project was built or executed; every "headless/deterministic/faithful" is the project's claim (VERIFIED as a claim only).
- Whether d2-dedicated-server can be time-accelerated, whether its native build needs a Mac 1.14d binary we can legally obtain, and its true multi-game stability.
- Reddit r/diablo2, r/D2Mods, Diablo2.io, Phrozen Keep, Blizzard forums, HN and the modding Discords: the search tool surfaced none of them (one Blizzhackers Discourse set of results, SECONDARY). A negative result here is weak evidence.
- Kolbot/D2BS exact 1.14d support state and licence (no licence file seen in the repos I listed; the d2bs D2Ptrs list I read is the 1.13-era noah- fork).
- D2MOO details (left to D2MOO-FIT), d2emu.com internals, dark-magic completeness, the Sorceress cast frames in Igor's spec, PlugY/D2GS licences beyond the API field.
- Contributor counts are API contributors; commit authorship was only checked for d2d and dark-magic.
