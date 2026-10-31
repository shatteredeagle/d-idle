# Redivivus (jaenster/redivivus): what it is and what it means for us

> status: research note, 2026-10-30, agent redivivus. Read-only GET fetches. Tags: V = fetched primary (URL), S = snippet or secondary, U = unverified or inferred. Not a recommendation to switch.

## 1. What it is

- **Not a reimplementation.** "Diablo II 1.14d recovered as C++ source that builds. Not a reimplementation: the shipped Game.exe, function by function." V https://github.com/jaenster/redivivus (repo description), https://redivivus.app/about
- Method: the 1.14d Game.exe and the Mac build are reverse-engineered in Ghidra databases; `ghidra-reconstruct` (via ghidra-mcp) regenerates C++ from them; crashes are fixed "usually in the Ghidra database". Most of the tree is regenerated, not hand-written. V https://redivivus.app/about, README
- Scope (V README): whole program, not only mechanics: client, D2Game (the embedded game server module), renderers (Glide, DDraw, D3D), sound, Storm/Fog libs, network layer, launcher, `D2Multi` (OOG launcher/chat/character select), `D2MCPClient`. Tree layout read from the contents API (V `gh api repos/jaenster/redivivus/contents`). Tree is generated, "some functions still read like a decompiler wrote them" (V README).
- **Author:** jaenster (Jan Stoots), contributors Nugwa628 (3 commits), plus issue/PR authors VTSTech, Niehztog, xrip, Nishimura-Katsuo. V `gh api repos/jaenster/redivivus/contributors`, `.../issues`. Credits blizzhackers (dzik), Niehztog's byte-match oracle, D2MOO for early struct layouts. V README.
- **License:** none. GitHub reports `license: null` and no LICENSE file exists in the tree (only thirdparty/libbacktrace/LICENSE). V `gh api repos/jaenster/redivivus`, tree listing. Legal default: all rights reserved, and it is derived from Blizzard's binary. Contrast d2d: GPL-3.0. V `gh api repos/psi29a/d2decomped` (spdx GPL-3.0).
- **Relation to d2d:** NOT shared code, NOT shared maintainers. Niehztog pointed psi29a at it (d2decomped issue #71, 2026-10-23); psi29a asked about collaborating, jaenster said "let's talk on Discord", psi29a gave his OpenMW-Discord handle. They live near each other (psi29a). No later outcome visible. V https://github.com/psi29a/d2decomped/issues/71. "Sister" is the user's framing; the relation is friendly contact, and d2d already cites it as a *secondary* source (see 3). U that any collaboration followed.
- **Maturity:** created 2026-10-23, last push 2026-10-30 (same day as this note), 43 commits (page count of per_page=1), nearly daily, essentially all by jaenster. 40 stars, 19 forks, 14 open issues, 8 PRs total (about half closed). No releases/tags (`releases` empty); CI: `build.yml` and `progress.yml`. No unit/replay tests found: the only test-like file is `asm_matching/msvc8/scripts/smoke_test.sh`. V GitHub API (repo, commits, issues, pulls, releases, workflows, tree). The project says work began 2021 (commit "recomp.fyi: start date 2021-03-26"). V commit log.
- **Platforms:** build is a mingw-w64 cross-compile to a 32-bit Windows `d2.exe`; you copy it next to your own Game.exe and run `d2.exe -w`. Needs your own 1.14 install, no game data in repo. Linux is only as a cross-compile host (Wine to run, U). No native Linux/Mac run. V README, docs/building.md.
- **State:** "It crashes." Starts, menus, single player create/walk/fight/area change. Open issues: Den of Evil completes instantly (#24), mobs not dropping items (#7, fix merged as #8), no sound (#9/#10), pick-up with full inventory crashes (#17), zlib alloc crash (#11). V README, issues.
- **Scope statement:** "single-player project... Online play on Battle.net is outside its scope." V README.

## 2. What it does differently

- **Fidelity by construction.** Accuracy claim is not tests but code identity: `asm_matching/` recompiles with VC++ 2005 SP1 and compares machine code per function. At version 1778: 5,689 of 14,812 functions (38.4%) and 393,360 of 2,555,443 bytes (15.39%) byte-identical, runtime excluded. V README, progress/report.json. So ~85% of code bytes are not yet verified identical, and "behaves like the original" is the first goal, byte-match second (about page).
- **No automation surface found.** No headless mode, control API, seeding, replay or determinism work visible. `D2Multi/McpConnect` and `D2MCPClient` look like the Realm MCP (Battle.net) client, not a bot hook (U, filenames only; a stray commit mentions a "control channel" build flag, purpose unread). Seed code exists but is just the original `Seed.cpp` (V tree).
- **Server side:** the D2Game module (monsters, AI, drops, levels, treasure classes) is in the tree as recovered original code (V tree: D2Game/Monster/*, GAME/Level.cpp, ITEMS/Items/Drop.cpp). A standalone realm server (D2Server) is only a "future idea". V https://redivivus.app/about.
- **Docs of formulas:** none of its own; the source IS the documentation. No damage/XP/spawn docs (V docs/ has only building.md and original-compilers.md). Names/types come from a Ghidra DB readable only through a GitHub-sign-in MCP server (https://ghidra.redivivus.app/mcp) (S about page; not accessed: needs login, and we do not sign in).
- **Performance/engine design:** a faithful port of the original architecture; not data-driven, no performance work. U/inferred.

## 3. Reuse for us

- **As a second oracle (reading only):** its D2Game source is a different reconstruction path from d2d's (d2d reimplements; Redivivus recovers). Where d2d and Redivivus agree on a mechanic that is a weak independent confirmation; where they differ it is a pointer to what to probe in Game.exe. d2d's own bar is the same: Redivivus "can confirm a reading, never replace one", tag `unverified (source: ...)`. V d2decomped AGENTS.md section 5 via `gh api`; our scan docs/research/upstream-pr-scan.md line 46. d2d already did this once (PR #115 "converted monster no target", comments tagged unverified, row in docs/research/re/unverified.md). V d2decomped issue #115 body, code search.
- **Fidelity gaps (monster density +25-30 %, champions per area, Fallen HP, per-kill time):** useful pointers, not answers. Relevant files to read for the numbers/formulas only: D2Game/Monster/MonsterRegion.cpp (region spawn/density), Monster/Create.cpp, GAME/Level.cpp, ITEMS/Items/Drop.cpp. We have not read them (U that they contain the density formula; D2Game is generated, may be imperfect and crashes were found in this area, see issue #7/#8 itemgen memset bug). Treat any value as a hypothesis to test with Game.exe probes (origprobes), per `docs/research/d2d-fidelity-riskmap.md`.
- **LAN/server oracle (whisper/d2dproxy):** no overlap. Redivivus does not host or speak a new protocol and its D2Game is a game-server module with no realm/D2GS wrapper yet. Our Game.exe-as-host oracle plan stands.
- **Container:** no. It is a Windows exe built with mingw, no headless mode, no Linux runtime.
- **Licensing:** no licence means we have no right to copy or reuse its source; reading is possible but it is a decompilation of Blizzard code. Our rule (no decompiled code in repo; addresses and numbers only) therefore applies doubly: record numbers/behaviour observed, cite path and commit, never paste code. GPL d2d cannot take its code either. Our private fork stays patches-only. We should not vendor, mirror or clone it into our repo (task rule), and a bare read of GitHub files is the extent of use. U: no lawyer view.

## 4. Risks

- **Ignoring:** low cost. We lose a cheap hint source for the D2Game mechanics; d2d's maintainer may fold its findings in anyway (he did in #115), so the signal arrives via d2d.
- **Depending:** high. No licence, no releases, one main author, daily regenerations (tree can change wholesale; function names churn, per its own rules "names change, addresses do not"), still crashes, 15 % byte-identical code, no automation hooks, Windows-only, and its scope excludes online play. Any claim taken from it is a reading of a reconstruction, not Game.exe.

## 5. Recommendation

**Watch, and cross-reference selectively; do not build or integrate.** Add a "secondary source" line for Redivivus in our evidence rules (same tag as d2d's: `unverified (source: redivivus path@commit)`), and when a fidelity gap (monster density, champion count, Fallen HP) has no Game.exe probe yet, read the matching D2Game files once on GitHub for the formula and turn it into a probe, never into a fact. Do not test a build now (crashy, Windows-only, no API, and Game.exe is already our oracle). Contact the author only if a specific question stays open after reading; psi29a already has a line to him and is the right intermediary. Not a switch.

## 6. Draft Discord questions (for the user, not posted)

1. "Does D2Game in Redivivus load and run the monster spawn density (MonRegion / area monster counts) the same way the original does, and has anyone compared it to Game.exe in play, e.g. mobs per outdoor area?"
2. "Is there any headless or scripted mode (or a plan for D2Server) that could run D2Game without the renderer, for deterministic testing?"
3. "What licence is intended for the tree, and is reading it to cross-check a separate GPL reimplementation (d2d) fine with you?"
