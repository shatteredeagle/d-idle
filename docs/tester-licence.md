# Tester pack: GPL notices, SOURCE.txt and the written offer for the d.idle fork (TASK-28)

> status: DRAFT, phase 1 of the fork publication (2026-11-07). Not legal advice; "GPLv3 section" references are the author's reading (U) until a reviewer is named or the user waives one. Nothing is published or posted. Scan and push plan: docs/research/fork-publication-scan.md. Fork README draft: docs/fork-publication/FORK-README.md. Background: docs/d2d-container/gpl.md and licensing.md.

## 1. What ships with a tester image

| Item | Form | Status |
|---|---|---|
| d2d binary built from the d.idle fork | in the image | built from the pinned ref below |
| Fork source at that ref | public repo `d2d-didle` plus a tarball in the pack | tarball command below (V command shape, not run) |
| Vendored inputs (CLI11, sentry-native, FFmpeg Bink-only static, StormLib, SDL3, OpenAL Soft) with hashes | list in `licenses/` | hashes: U here, taken from the build context when the image is cut (the build prints them; FFmpeg 9.0.2 is hash-checked, S docs/d2d-container/README.md) |
| Build recipe | Dockerfile and CMake flags, in our source tarball | S docs/d2d-container |
| Our source tarball (the bot: Go) | `git archive` of this repo at the shipped commit, after leakcheck | U not cut yet |
| Licence files | GPL-3.0 text, upstream LEGAL.md, NOTICE, SOURCE.txt in `/usr/share/licenses/d2d/` | below |
| Ubuntu base image packages | pointer: the distribution's source repositories for the installed package versions (apt source, `dpkg -l` list in the image) | U to be generated at build |
| Game data | NEVER shipped; the tester brings their own 1.14d MPQ set | decision of 2026-11-06 |
| Artwork exceptions in upstream (Freepik icon, a freeware font) | question below | open |

Pinned ref: `f75816c616c8253930d88277b90936e3eabe3bc8` (`d2d.lab3-labverbs`, the Gate 2 binary base; contains `ee5e06d8ef99`, the Gate 1 and Gate 2 fork). Move to the overlay tip `e66d0fdf88a0` once a binary is built from it and pinned.

Source tarball (run in a clone of the published fork):
```
git archive --format=tar.gz --prefix=d2d-didle-f75816c616c8/ -o d2d-didle-f75816c616c8.tar.gz f75816c616c8253930d88277b90936e3eabe3bc8
sha256sum d2d-didle-f75816c616c8.tar.gz
```

## 2. NOTICE (draft text)

```
d2d (d.idle fork)
Based on d2decomped by psi29a and contributors, GPL-3.0-or-later.
Modifications: lockstep clock, dump and lab verbs (off unless --lab-verbs),
table overlay (--overlay-dir). Source: https://github.com/shatteredeagle/d2d-didle
at the commit named in SOURCE.txt.
Third-party components keep their own licences; see licenses/.
Diablo II is a trademark of Blizzard Entertainment. No Blizzard game data is
included; you supply your own copy.
```

## 3. SOURCE.txt (draft text, without the NOT FOR DISTRIBUTION line)

```
This image contains the program d2d, built from:
  repository : https://github.com/shatteredeagle/d2d-didle
  commit     : f75816c616c8253930d88277b90936e3eabe3bc8   (tag didle-gate2-lab3)
  tarball    : d2d-didle-f75816c616c8.tar.gz  sha256 <fill at cut>
               (made from the public tag with `git archive`, see docs/fork-publication/PUBLISHED.md)
Vendored inputs (name, version, sha256): <fill at cut from the build context>
Build recipe: the Dockerfile and CMake flags in the source tarball of the
bot repository, commit <fill at cut>.
The program is licensed GPL-3.0-or-later; the licence text is in this directory.
```

## 4. Written offer (draft text)

```
The Corresponding Source for the d2d program in this image is available at the
repository and commit named in SOURCE.txt. If that location is unavailable, write to
<contact fill at cut> and a copy will be sent on request for at least three years
from the date you received the image, at no more than the cost of the medium.
```

## 5. Freepik icon question (open, U)

Upstream LEGAL.md names the icon (Freepik licence) and a freeware font. The icon is compiled into the binary. Whether the Freepik terms allow redistribution inside a binary is unverified; options: confirm the terms, or patch the icon out in the fork. The publication of the source repository carries the same question (the files are upstream's, already public there).

## 6. USER DECISION list

1. ADR 0003 amendment: RECORDED 2026-11-07 (docs/adr/0003, "Amendment 2026-11-07"): public GitHub fork `d2d-didle`, lab verbs flag-gated and off by default. Done.
2. GPL reviewer: named or waived (open; ADR 0003 says none yet; the user's go of 2026-11-07 covers the publication of the source, not a review).
3. When the repo goes public: DONE 2026-11-07 as the standalone repo https://github.com/shatteredeagle/d2d-didle (record: docs/fork-publication/PUBLISHED.md).
4. Lab and cheat verbs: FLAG-GATED, not stripped (decided 2026-11-07). The test cheats never go into an upstream PR without approval.
5. The user's go on the day of the first copy of an image to anyone else.
6. Rewrite history or not (session-URL trailers): docs/research/fork-publication-scan.md, D1.
