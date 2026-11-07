# Fork publication, phase 1: history scan and push plan (agent forkpub, 2026-11-07)

> status: phase 1 result. Nothing was created or pushed on GitHub. Phase 2 (create the fork, push) is the coordinator's, after reading this. Decision record: ADR 0003 "Amendment 2026-11-07".

Tags: V = verified (how), S = secondary, U = unverified. No secret value, private hostname, IP, path or email appears here: classes, counts and commit ids only.

## 1. Method (V)

* Scratch bare clone of the fork's private remote (the dev-box checkout was not touched) plus `git fetch` of upstream `psi29a/d2decomped` main into it as `upstream-main` (tip b52e419a4fac). Merge-base of the overlay lineage with upstream main is 52362d52cfa0; upstream is 656 commits ahead of it (V `git rev-list --count`).
* "Ours" = every commit reachable from any branch and not from upstream main (`git log --branches --not upstream-main`). Scanned: every added line of every one of those commits (`git log -p -U0`, so every blob version, not only tips), every commit message, the file list and blob sizes (`git rev-list --objects`), tags. Regexes: AWS keys, GitHub token shapes (ghp_/gho_/ghu_/ghr_/ghs_/github_pat_), private key blocks, Discord bot token shape, `password|secret|api_key|token` assignments, 4x4 CD key shapes, cookies, the private lab domain, `*.lan`, RFC1918 IPs, the project's machine and person names (the classes of `cmd/leakcheck` scan.go), home paths, lab paths, emails, UUIDs, 40+ character tokens, Blizzard file extensions. gitleaks and trufflehog are not installed on the box; `git filter-repo` is.
* The same sweep was re-run on the overlay lineage alone (`upstream-main..d2d.lab4-overlay`).

## 2. Scope (V)

| Set | Non-upstream commits | Dates |
|---|---|---|
| all branches of the private remote | 375 unique, on 120 of 142 branches (the rest equal upstream) | 2026-10-30 to 2026-11-06 |
| `d2d.lab4-overlay` (tip e66d0fdf88a0) | 15 | 2026-10-31 to 2026-11-06 |
| contained: `d2d.lab3-labverbs` (f75816c616c8, Gate 2 base) | 12 | |
| contained: `lab2-freezeresume` (ee5e06d8ef99, Gate 1 and Gate 2 fork) | 11 | |

Branch families (V `git for-each-ref`): `d2d.lab3-labverbs`, `d2d.lab4-overlay`, `lab*` and `lab2..lab6` (our patch lines), `local/*` (about 100 PR drafts and rounds; the PRs they became are all MERGED upstream, S `gh pr list`, 31 of ours), `fix/*`, `pr/*`, `stage/*` (CodeRabbit staging on the fork's own main), `d2dtown-vendor`, `p3-layout-verb`, `main` (2 commits, a review config for staging). Tags: 23, all upstream release tags (none reach beyond upstream main).

Contained in the overlay tip: lockstep, dump verbs, debug and setup cheat verbs, freeze/resume and Frozen Armor oracle tests, the `--lab-verbs` gate and the `--overlay-dir` table overlay (the 15 commits listed by `git log upstream-main..d2d.lab4-overlay`).

### Authors (V `git log --format`)

Across all 375 commits: 6 author names, 4 author emails, 3 committer names, 3 committer emails.

| Class | Commits (all branches) | In the overlay lineage |
|---|---|---|
| agent identity (name Claude, vendor noreply email) | 139 | 11 |
| the owner's GitHub noreply form (id+handle) as author | 227 | 4 |
| a placeholder `fork` author (vendor noreply or a `@local` committer) | 6 | 0 |
| **a personal mailbox address (consumer webmail) as author, three variant spellings of the owner's name** | **3** (c4fcb0a6, 3cbf2743, 3c48fdfd) | **0** |

The 3 personal-address commits sit on `stage/base-2`..`stage/bundle-4` (c4fcb0a6) and `lab2`..`lab5` (the other two), not in the overlay lineage. The overlay lineage carries no personal email: authors are the agent identity or the owner's noreply form.

## 3. Findings (V: regex sweep over all added lines and all messages, results above)

| # | Class | Count | Where | Severity | Proposed action |
|---|---|---|---|---|---|
| F1 | Personal mailbox address in commit metadata | 3 commits | c4fcb0a6, 3cbf2743, 3c48fdfd | medium (private info) | do not publish those branches, OR rewrite with the mailmap below on a scratch copy. Not needed for the overlay lineage |
| F2 | `Claude-Session:` trailer = link to a private claude.ai session | 81 trailer lines in 81 commits (5 in the overlay lineage) | commit messages | low-medium (a private-session URL; not a credential, but not for strangers) | strip with `git filter-repo --message-callback` on the scratch copy for everything published, or accept (user decision) |
| F3 | Agent `Co-Authored-By: Claude ... <vendor noreply>` trailers | about 140 | commit messages | info | accept (public vendor noreply address, normal) |
| F4 | Credentials, tokens, keys, passwords, cookies, CD keys | 0 | | none | none |
| F5 | Private hostnames (the lab domain, `.lan`, machine names), IPs, home or lab paths, person names | 0 in added lines and messages (V: zero hits in all classes of the leakcheck set, both sweeps) | | none | none |
| F6 | Blizzard-owned data (binary types .mpq .dc6 .dcc .d2s .d2i .d2x .dll .exe .tbl .dt1 .ds1) | 0 files ever added by us (V: added-file list has only .cpp .hpp .py .md .txt .inc .yaml .cmake); the regex hits are 90 added LINES that name `d2data.mpq`, `field.ds1`, `level.ds1` in test code as strings (the tests read the tester's own data) | tests | none | accept |
| F7 | Game data tables committed in full (Misc.txt style) | 0 (V: no .txt data files were added; the one `.inc`, below, is not a table) | | none | none |
| F8 | `tests/mon_ais_pace_cases.inc`: 840 numeric test vectors recorded from game.exe thinks (a state-vector dump, seed 20261102) | 1 file, 1 branch family (`local/monsterpace*`, not in the overlay lineage) | | low (game-derived numbers, not Blizzard text or code; S: facts, but a lawyer's call) | keep the branch out of phase 2 unless the user wants it |
| F9 | `.coderabbit.yaml` on the fork's `main` | 1 | `main` | none | accept (no private content, V read) |
| F10 | Words in comments: "private fork", "Local patch (private fork)" | 6 files in the overlay tip (devctl debug*.cpp/hpp, devctl_dump.hpp, lockstep.hpp) | source comments | low (wording becomes false) | reword in a follow-up commit on `didle/main`, not by rewriting history |
| F11 | 40+ character tokens | 230 hits all file paths or the session URL id | | none | none |

Result: no credential, no private host/IP/path, no Blizzard-owned file in the history (V). The only private-info items are F1 (3 commits, outside the overlay lineage) and F2 (the session trailers).

Not covered (U): the bot-side docs and research files that may mention the user's private matters live in this repo, not in the fork, and are scanned by `cmd/leakcheck`; the fork history itself has no such docs beyond `docs/research/re/*.md` (decompile notes, upstream's own files, modified by us in 2 commits, regex-clean).

### Proposed mailmap plan (nothing rewritten)

Scratch copy only: `git clone --bare <bare> scratch.git && cd scratch.git && printf '%s\n' 'Owner Name <owner-id+handle@users.noreply.github.com> <personal-address>' > ../mailmap` (one line per personal address; use the owner's real noreply form from his GitHub settings, not written here) then `git filter-repo --mailmap ../mailmap --message-callback 'import re; return re.sub(rb"\nClaude-Session: [^\n]*", b"", message)'`. Rewriting changes every commit hash from the first touched commit on. The overlay lineage needs neither (no personal email; session trailers only if the user wants F2 gone, which rewrites its 5 trailer commits and all descendants: tip hashes change, so the pinned f75816c616c8 and ee5e06d8ef99 in the tester pack would no longer exist in the public repo).

Trade-off the user must decide (open decision D1): publish the overlay lineage UNREWRITTEN (hashes stay as the Gate 2 provenance cites them; the 5 session URLs are public) or rewrite F2 (cleaner, hashes change, provenance docs re-pinned).

## 4. Licence and notices (V: `git ls-tree`, `git show`)

* LICENSE present at the root (GPL-3.0 text), the README License section says GPL-3.0-or-later. Upstream sources start with `// SPDX-License-Identifier: GPL-3.0-or-later`.
* Our 10 added files: 7 carry the SPDX line. **Gaps (3):** `apps/d2d/lockstep.cpp`, `apps/d2d/lockstep.hpp`, `apps/d2d/devctl_dump.hpp` have no SPDX header (lockstep.hpp opens with "Local patch P2 (private fork)"). Fix: one follow-up commit on `didle/main` adding the line and rewording.
* No NOTICE file exists upstream or in the fork; none is required by GPL-3.0 for the fork itself. A NOTICE is only for our tester pack (docs/tester-licence.md, TASK-28).
* Vendored/fetched inputs (CLI11, sentry-native, FFmpeg) keep their own licences; the pinned vendor list is in docs/tester-licence.md.
* The overlay tip is 656 commits behind upstream main (V). A GitHub fork keeps the shared history, so branches on the old base push fine; a rebase onto current upstream is optional and is a separate job (docs/forkrebase.md).

## 5. Push plan (commands, not executed)

Branch set to publish (all reachable from the three named tips; nothing else, V by `git branch --contains`):

| Public name | Source | Commits over upstream base | Purpose |
|---|---|---|---|
| `didle/main` | `d2d.lab4-overlay` e66d0fdf88a0 | 15 | the d.idle fork (default working branch) |
| `didle/lab3-labverbs` | `d2d.lab3-labverbs` f75816c616c8 | 12 | Gate 2 binary base (provenance) |
| `didle/lab2-freezeresume` | `lab2-freezeresume` (contains ee5e06d8ef99) | 11 | Gate 1 and Gate 2 fork |

Tags: `didle-gate2-lab3` at f75816c616c8, `didle-gate2-lab2` at ee5e06d8ef99, `didle-overlay-1` at e66d0fdf88a0. The `local/*`, `fix/*`, `pr/*`, `stage/*`, `lab`..`lab6` families are NOT published: they are PR rounds already merged upstream or private staging (F1 lives there). Upstream-PR candidates that are already pushed upstream need no copy.

```
# 1. create the fork (needs the user's go; phase 2)
gh repo fork psi29a/d2decomped --fork-name d2d-didle --clone=false
# 2. from the scratch bare clone (the private remote is never altered)
cd <scratchpad>/d2d-fork.git
git remote add didle https://github.com/shatteredeagle/d2d-didle.git
git push didle d2d.lab4-overlay:refs/heads/didle/main \
              d2d.lab3-labverbs:refs/heads/didle/lab3-labverbs \
              lab2-freezeresume:refs/heads/didle/lab2-freezeresume
git tag didle-gate2-lab3 f75816c616c8 && git tag didle-gate2-lab2 ee5e06d8ef99 && git tag didle-overlay-1 e66d0fdf88a0
git push didle didle-gate2-lab3 didle-gate2-lab2 didle-overlay-1
# 3. root README: add docs/fork-publication/FORK-README.md as README-DIDLE.md on didle/main (and SPDX fix commit), then set didle/main as the default branch in the fork settings
```

The work in the fork checkout for steps 3 (SPDX and README commit) is a new commit on `d2d.lab4-overlay` in the private remote first (push the same day, AGENTS.md), then pushed to GitHub as `didle/main`.

Tester pack source tarball (V command shape; run in the scratch bare clone):

```
git archive --format=tar.gz --prefix=d2d-didle-f75816c616c8/ -o d2d-didle-f75816c616c8.tar.gz f75816c616c8253930d88277b90936e3eabe3bc8
sha256sum d2d-didle-f75816c616c8.tar.gz
```

Use the overlay tip (e66d0fdf88a0, adds `--overlay-dir`) once the Gate 2 binary is rebuilt from it and pinned. The drafts are docs/fork-publication/FORK-README.md and docs/tester-licence.md.

## 6. Open decisions (the user's)

* D1 rewrite or not (F2 session trailers); recommendation: publish unrewritten, the URLs are inert without his login, hashes stay as cited.
* D2 which extra branches, if any, beyond the three (recommendation: none).
* D3 GPL reviewer named or waived (ADR 0003 original: none named); the amendment records the publication, the user's conditional go covered a clean scan.
* D4 the SPDX and wording commit before the first push (recommended yes).
* D5 F8 test vectors: keep out (recommended).

## 7. Recommendation

Publish after one small fix: add the 3 SPDX headers and reword the "private fork" comments in one commit on `didle/main`, then push the three branches and three tags above, rewriting nothing. The history has no credentials, no private hosts or paths, no Blizzard data.

## Places that still say "private fork" (grep, S; wording fixed only in the ADR record and docs/d2d-fork.md, the rest listed)

Operating docs and rules that stay TRUE as written (the dev remote on the forge stays private, same-day push): AGENTS.md, docs/forkcheats.md, docs/forkdbg.md, docs/forkarmor.md, docs/forkfreeze.md, docs/forknext.md, docs/forkrebase.md, docs/upstreamsync.md, docs/lab-verbs.md, docs/overlay.md, docs/cheats.md, docs/d2d-lab.md, docs/d2d-lab6.md, docs/START-HERE.md, docs/capabilities.md, docs/builder.md, the docs/d2d-*.md family, docs/research/pr-*.md and docs/research/*diff*.md (historical), tasks 5, 8, 42, 59. These say "private" about the dev remote or about the past; leave them until the publication happens, then a coordinator sweep can say "d.idle fork (GitHub: d2d-didle)". docs/plan/STATUS.md and docs/BACKLOG.md, docs/INDEX.md, CHANGELOG.md are generated or coordinator-owned.
