# GPL and licence notes for a d2d container image

NOT LEGAL ADVICE. This is a factual summary by an AI agent for the user to take to a lawyer or decide on. Tags: V = read from a file I opened, S = secondary (our docs), U = unverified or from memory.

## State today
- Nothing is distributed. The fork is a private forge repo (`<owner>/d2d-fork`), binaries live on the worker host, the image does not exist, no image is pushed (V docs/decisions.md 2026-10-30; this task builds nothing).
- Private use of GPL software, including modified, creates no publication duty (U: the general GPL position; the project's own LEGAL.md and design notes say the same, S docs/d2d-fork.md section 8).
- The repo and any image stay private until the user decides otherwise; "who reviews GPL obligations" is unanswered (V decisions.md).

## What d2d is
- Upstream `psi29a/d2decomped`: every source file carries `SPDX-License-Identifier: GPL-3.0-or-later` (V top CMakeLists.txt first line, `LICENSE` and `LEGAL.md` present in the tree). Its LEGAL.md says no Blizzard files are in the repo and you supply your own game (V read).
- Our fork adds patches (P1 dump, P2 lockstep, 324 lines added; V d2d-fork.md). Those are derivative works and therefore GPL-3.0-or-later too (U: standard reading).
- Two artwork exceptions in upstream: the icon (Freepik licence) and a freeware font in the banner (V LEGAL.md). The icon is compiled into the binary (V apps/d2d/CMakeLists.txt reads `assets/icon/d2d.bmp`), so a notices file must carry its terms (U what they require).
- Third-party code in the tree or build: CLI11 (BSD-style, U), sentry-native and crashpad (MIT and Apache-2.0, U), FFmpeg 9.0.2 built Bink-only, static (LGPL by default, U, the configure line has no `--enable-gpl`, V), StormLib (MIT, U), SDL3 (zlib licence, U), OpenAL Soft (LGPL, dynamically linked, U), zlib/bzip2/libcurl (permissive, U). Upstream installs licence files for Sentry itself (V CMake `install(FILES ... LICENSE)`); a product image needs a complete notices file (not written).

## If an image containing the d2d binary is given to anyone (including one friend)
1. Provide the GPL-3.0 text and a copyright/licence notice with the binary (U: GPLv3 sections 4 to 6). Draft puts them in `/usr/share/licenses/d2d/` (Dockerfile.draft).
2. Provide the Corresponding Source for exactly that binary, or a written offer valid at least three years (U: GPLv3 section 6). "Corresponding Source" includes the build scripts: our fork commit (`c36b77b...`), the vendored dependency pins, the Dockerfile and the CMake flags. Easiest compliant form: publish the fork repo (a tag per image) and put its URL and the commit in `SOURCE.txt` inside the image; or ship the `git archive` tarball next to the image.
3. Our patches must be under GPL-3.0-or-later and say so (they are new files in a GPL tree; keep the SPDX header).
4. No further restrictions: the image must not forbid the recipient from modifying and rerunning the binary (U: "installation information" duties apply to consumer devices only, U).
5. Offering the source via a private repo does not satisfy the offer for recipients who cannot reach it: publish or attach.
6. Hosting a service where users only use the container over the network is not distribution under plain GPL (the AGPL differs; d2d is GPL not AGPL, V SPDX line). Giving someone the image or a download is distribution.

## Our own code (d2bot, d2fleet, ...)
- Today: no licence file decision (V docs/decisions.md says "decide our own license at distribution time"; `ls` of the repo root shows no LICENSE, V).
- The bot talks to d2d over a unix socket as a separate program started with `exec`; the common view is that this is "mere aggregation"/separate works, so the GPL does not extend to the bot (U, not legal advice; same position in docs/d2d-backend.md section 4.5, S).
- Weak points to ask about: (a) our fork's patches are GPL but sit inside the same binary the bot drives, which is fine; (b) the bot embeds knowledge of d2d's verbs and text formats, copying none of d2d's code or text (rule 3 of decisions.md); (c) putting both in one image: "aggregate on a storage medium" is allowed by GPLv3 section 5 (U), but keep separate layers and directories (the draft does) and state clearly which licence covers which path.
- Our code also depends on game knowledge and the MPQs; no Blizzard data is committed or imaged (V AGENTS.md rule, design.md section 3). Whether shipping a tool that automates Blizzard's game, even offline against a re-implementation, raises Blizzard trademark or EULA questions is outside what I can say (U; D2R EULA is not relevant, this is classic data run by d2d).
- Licence choice for our code is the user's: options are keep it proprietary/private (allowed if never distributed in an image with d2d as a derivative), a permissive licence (MIT/Apache) on the Go code, or GPL-3.0 for simplicity. Each has consequences for third-party contributions and for the explanation burden when both are in one image.

## Obligations checklist before the first shared image (not done)
- Decide who reviews. Decide our code's licence. Add LICENSE files and `NOTICE`.
- Build the notices file for every dependency in design.md 1.1 (CLI11, sentry/crashpad, FFmpeg, StormLib, SDL3, OpenAL, libcurl, zlib, bzip2, Ubuntu packages in the runtime stage, which carry their own copyright files under `/usr/share/doc/*/copyright`, U).
- Publish or attach the source for the fork commit, the vendored pins, and the build recipe; keep an image-to-commit mapping (tag).
- Confirm the Freepik icon terms allow redistribution in a binary (U) or patch the icon out in the fork.
- Keep Sentry crash reporting off and the DSN blank in the shipped binary (draft does), so no data leaves the user's host (V `--send-crash-reports` default off, main.cpp 1643).
- Never include MPQs, `Game.exe`, saves or the token in any layer.

## Questions for a lawyer or the user
1. Does a socket-connected, separately built Go bot stay outside the GPL when shipped in the same image as the GPL binary?
2. Is a source offer by pointing to a public Git tag of the fork enough, and for how long must it stay up?
3. Which licence do you want for D.idle itself, and may the bot code stay private while the fork is published?
4. Is an image shared with a few friends "distribution" for you (yes under the GPL), and are you comfortable publishing the fork and patches then?
5. Any exposure from automating Diablo II (a Blizzard title) with a bot, even offline against d2d, if the project is ever public? (Personal use is the current scope.)
6. Do we offer P1/P2 upstream (drafts exist, V d2d-fork.md section 9)? That would shrink the fork and the GPL surface to nothing of ours.
