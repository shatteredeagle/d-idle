# Legal and risk note before the first outside copy of d.idle (TASK-257)

> status: research note for the user, 2026-11-10. NOT LEGAL ADVICE: written by an AI agent from public texts and our own docs; a lawyer (or at least the GPL reviewer ADR 0003 still lacks) should read it before a public release. Target: classic 1.14d, not D2R. Tags: V = fetched or read this run (source named), S = secondary, U = unverified or from memory. Nothing in the repo or the fork was changed.

## 1. GPL-3.0-or-later when a copy leaves the house

- Our repo is GPL-3.0-or-later (V: LICENSE, ADR 0010). The d2d reimplementation (psi29a/d2decomped) is GPL-3.0-or-later (V: ADR 0010 read of its LICENSE/LEGAL.md). Using or hosting it privately triggers nothing; giving a friend an image or tarball is "conveying" (U: GPLv3 s.6 from memory; docs/d2d-container/gpl.md carries the analysis).
- What the friend must be able to get: the Corresponding Source of exactly the binary shipped, and the licence text. Three code groups:

| Part | Whose | Obligation |
|---|---|---|
| d2lod-bot Go code (`/opt/d2lod`, own layer, talks to d2d over a socket) | ours, GPL | source = this repo at the image's commit; keep `LICENSE` in the image |
| d2d binary in the image = the d.idle fork (lockstep/time-scale/step/dump patches, lab verbs flag-gated) | upstream GPL + our GPL patches | the fork source at the exact commit incl. the lockstep patches, build flags, vendored pins; publishing `d2d-didle` and tagging each image to a commit satisfies it (decisions.md: published 2026-11-07, user go; image still needs a go at distribution) |
| Third party in the image: CLI11, crashpad, FFmpeg, StormLib, SDL3, OpenAL, libcurl, zlib, bzip2, Ubuntu runtime packages, the Freepik icon compiled into d2d | their own licences | a notices file for each (gpl.md lists them); Ubuntu keeps copyright files under `/usr/share/doc`; the Freepik terms are U |

- THIRD_PARTY.md (V: read) covers the repo only: zlib explode port, D2DX (never bundled), Kolbot and Tyniann (ideas only; Tyniann text still UNVERIFIED). It does not yet list what the image adds; the image needs its own notices file (gpl.md says `SOURCE.txt` still reads "NOT FOR DISTRIBUTION").
- Lockstep patches: "private until upstream wants them" (ADR 0010 addendum) is incompatible with shipping the fork binary. Once a binary containing them leaves, the recipients may demand that source and may republish it under GPL. The patches may stay unpublished only if no binary goes out. Decision already taken 2026-11-06/07 (publish the fork); consequence: nothing in the patches may be secret or contain private data (the fork-publication scan, research/fork-publication-scan.md).
- Image layers: source offer must match the layer's binary; every layer ships (docker history/save exposes all), so no secrets, tokens, game files or caches in any layer, including deleted ones. The container build already aborts on `.mpq`/`game.exe` in its context (V: docs/d2d-container/licensing.md). A built image is a convenience copy: rebuilt from a tagged commit, not from the dev tree.
- Our Go code has no third-party dependencies (V: ADR 0010), so no dependency licence enters it. Our GPL covers our code only; it gives nobody rights to Blizzard material.
- Not distribution: streaming output, a web page, running it for yourself. A friend who only watches `/watch` gets no copy. Giving a friend SSH or a login to your own box is also not conveying (U).

## 2. Blizzard EULA and fan-project posture

What the text says (V: https://www.blizzard.com/en-us/legal/fba4d00f-c7e4-4883-b8b9-1b4500a402ea/blizzard-end-user-license-agreement, "LAST REVISED March 21, 2024", fetched 2026-11-10). This is Blizzard's current platform EULA; the exact text shown when a given 1.14d install was set up (retail disc, Battle.net download) may differ, and an older D2 EULA is not publicly fetchable here (U).
- 1.C "License Limitations": no copying, reverse engineering, modifying, decompiling or "creat[ing] derivative works based on or related to the Platform"; no cheating, and "bots; i.e. any code and/or software, not expressly authorized by Blizzard, that allows the automated control of a Game ... e.g. the automated control of a character in a Game"; no "hacks" (modifying the software in any way not authorised). Blizzard may suspend or revoke the licence for violations.
- Blizzard keeps rights in "all recordings, Game replays", the game's art and "titles, methods of operation", and "the right to create derivative works" except under its Fan Policies.
- Legal FAQ (V: https://www.blizzard.com/en-us/legal/ page "Legal FAQ", Diablo III context, old): fan sites may show screenshots and some assets for "home, noncommercial and personal use only" with notices kept, no hosting of "hacks and cheats", and "We do not allow add-ons or expansions for Diablo" (S: search snippet of the same FAQ). Video Policy (V: https://www.blizzard.com/en-us/legal/dd76b654-f2c4-4aaa-ba49-ca3122de2376/blizzard-video-policy): community videos with Blizzard content allowed if non-commercial, viewers not forced to pay, T-rated; partner-program ad revenue on YouTube-type sites is explicitly tolerated (the list is old and names no Kick or Twitch: U whether it applies).
- Enforcement history: the Eighth Circuit in Davidson & Associates v. Jung (the bnetd case) upheld Blizzard's EULA reverse-engineering ban and rejected the DMCA interoperability defence (S: eWeek and law-firm summaries from search results). Blizzard has pursued bnetd-style emulators and Battle.net cheat tools; I found no instance of it pursuing an offline, own-install tool (U, absence of evidence).

Reading of our posture (my judgement, not a ruling):

| Item | Risk | Why |
|---|---|---|
| Reading your own install at runtime (tables, MPQs, art) | low | personal use of a licensed copy; nothing copied out of your machine; the tool is a reader. Interoperability reading is the usual defensible ground, but the reverse-engineering clause is read broadly (U) |
| The bot automating the game (1.C "bots") | real, but offline | the clause targets Battle.net play; ours is offline single player, no Battle.net (AGENTS.md). Remains a EULA breach in the letter for the game.exe backend (U); d2d is a clean reimplementation, not the Platform. Never run it on Battle.net or with a Battle.net account |
| Memory writes / lockstep hook on game.exe | higher under "hacks" | offline, lab-only, but a friend copy must not ship game.exe tooling; the friend build is the d2d container (decisions.md) |
| Shipping/committing art, tables, saves, MPQs, DC6, screenshots of art-rendered pages | high, not ours to give | Blizzard owns it; GPL does not license it. Already forbidden by tools/ci/check.sh and the tests (V: THIRD_PARTY.md, paperdoll-art-build.md) |
| Blizzard names ("Diablo II", "Blizzard") in a public repo, image name or registry listing | low to medium | nominative use to say what it needs is normally fine (U); Blizzard's trademark guidelines allow marks only non-commercially and in line with an activity policy (V: https://www.blizzard.com/legal/38fd0408-8431-469a-99bc-2cd9eb9462c8). Do not name the image or domain "diablo...", do not use Blizzard logos, add a "not affiliated" notice (S: the Legal FAQ asks for notices and dislikes product names in domains) |
| Charging for it, tip-jar or paid tier on the stream | raises everything | Blizzard's fan rules require non-commercial use; our GPL allows selling but Blizzard's does not allow monetising its content (V: Video Policy) |

Fan-project posture that fits all of this: free, source open, "bring your own game files", no Blizzard assets in anything we ship, no Battle.net, a visible disclaimer, and compliance when Blizzard asks (a takedown request is the realistic risk, not litigation; U).

## 3. Art on the streamed view (Kick overlay)

- Today the stream overlay (`/watch?mode=overlay`) never loads art: `web/inventory.js` line 221 returns false for `mode=overlay` before any art request (V: read in this worktree), and the doc promises it stays art-free until the user decides (docs/paperdoll-art-build.md). That is the default asked for by TASK-257; nothing needs changing.
- If art were turned on for the stream, the viewer sees Blizzard artwork rebroadcast to the public. Streaming gameplay is the most tolerated use Blizzard has (Video Policy: non-commercial, free to view; V) and the same art appears in any D2 stream, so risk is low, but it is outside "your own use at home": it is a public performance of their art under their video policy, and monetised channels (Kick subscriptions, ads) cut against the non-commercial rule (U whether Kick monetisation counts; the policy names only a few old partner sites).
- Recommended: keep the overlay art-free for the friends build and a monetised channel; allow art only on the owner's private, token-gated `/watch/inventory` page. Never record or share screenshots of the art-rendered page, the cache directory or the data volume.

## 4. Checklist before the first friend copy

Must be true (agents can check):
1. Friend build is the d2d container only; no game.exe/Wine tooling, no lab or cheat verbs enabled (flag-gated off by default).
2. The fork source is published (the public `d2d-didle` fork) and the image is built from a tagged commit; the image carries `LICENSE`, the GPL text, a source offer naming repo URL and tag, and `SOURCE.txt` no longer says NOT FOR DISTRIBUTION.
3. A notices file for every third-party package and the Freepik icon (or the icon patched out).
4. `docker history` and a layer listing of the final image show no `.mpq`, DC6, art PNG, save, token, password, private hostname or home path (`go run ./cmd/leakcheck -mode fail`, tools/ci/check.sh).
5. Art stays runtime-only and off in the overlay; no art in screenshots or docs.
6. README states: not affiliated with or endorsed by Blizzard; bring your own legally owned Diablo II 1.14d files; offline only; GPL-3.0-or-later; no warranty.
7. No Battle.net code or account path (AGENTS.md); image/repo name has no Blizzard marks.

The user must decide:
- Go for distribution of the image (decisions.md needs the user's go at that time; ADR 0003 needs the grill amendment named there).
- Friend delivery: image file, private registry, or run it for them (no conveying). Private first is the lowest-risk route (U).
- Whether the lockstep patches really go public now (it follows from shipping the fork binary).
- Who reviews GPL compliance (ADR 0003: none named) and whether a lawyer reads this note.
- Art on a public stream: default off (recommended), or allowed for a non-monetised channel only.
- Whether the channel is monetised and whether to ask Blizzard (legal@/fan-policy contact; U exists) before going public.
- Tyniann licence text (THIRD_PARTY.md) and the Freepik terms: verify or remove.
