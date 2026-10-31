# Public-readiness audit (agent PUBLICPREP, 2026-10-30)

Read-only audit of the d2lod-bot repo before it goes public (ADR 0010 addendum). Scope: working tree at `master` 17fffea plus
`agent/licensedecide`, full history (`git log --all`: 2003 commits, 438 refs, 28 MB packed, `refs/stash` included), commit
messages, CI, `.claude/`. Nothing was pushed, changed on a remote or made public. Tags: V = verified by the command named,
U = my judgement. No secret value is quoted anywhere in this doc: path:line and type only.

## 1. Findings

| # | class | count (V) | examples (path:line) | severity | verdict |
|---|---|---|---|---|---|
| S1 | Secrets, working tree | 0 real | `tools/publish/tests/test_publish.py:TOKEN` (fake bearer, test), `tools/kick/kick_common.py:16-17` (account *key names*, no values), `internal/party/party.go:251` (generated LAN game password `pw<n>`) | none | fine |
| S2 | Secrets, full history | 0 real | same fixtures as S1; 0 private-key blocks, 0 ssh pubkeys, 0 GitHub/Slack/AWS/OpenAI/Stripe/JWT prefixes, 0 `.env`/`.pem`/`.netrc`/`id_rsa` ever added, 0 `?token=<value>`, 0 cookie values, 0 CD keys (16/26-char shapes near "key": 18 hits, all Go struct `Key:` fields) | none | fine |
| S3 | Secret *handling* code | n/a | `internal/api/api.go:27,210`, `cmd/d2fleet/main.go:146-171`, `.forgejo/workflows/ci.yml:77-79` (`secrets.RELEASE_TOKEN` by reference) | none | fine (token read from files/env, never literal) |
| P1 | LAN hostnames `*.lan.birbhub.com` | 49 files / 96 lines (history: 111 added lines) | `AGENTS.md:38`, `tools/live.sh`, `tools/watch.sh`, `tools/sentinel.sh`, `docs/boardhost.md`, `docs/soak-hosts.md`, `tools/ha-stream/d2idle_stream.yaml:3`, `.claude/agents/d2-researcher.md` | medium | decide: scrub in the public copy (domain is private DNS; reveals home topology) |
| P2 | Private IPs | 11 files / 16 lines (`192.168.*`), 4 files (`10.*`, test fixtures) | `docs/boardhost.md:4,15,20`, `docs/ci.md:60`, `docs/d2d-fork.md:176`, `docs/tower-container.md:17`, `docs/process/orion-compute.md:7,12` | medium | must scrub |
| P3 | Home paths `/home/jesse` | 17 files / 30 lines | `tools/live.sh`, `tools/soak/layout.sh`, `docs/launch.md`, `docs/d2d-spike.md`, `cmd/d2dview/view_test.go` | low | must scrub (replace with `$HOME`/`~`) |
| P4 | Unraid/Omlette share paths | 27 files / 53 lines | `docs/tower-container.md`, `deploy/nas/README.md`, `AGENTS.md:40` | low | decide (generic `/mnt/user/appdata` is fine; `Omlette` is a private drive name) |
| P5 | Third person's name | 4 files / 9 lines (history 16) | `docs/process/forgejo-runners-plan.md:47,49,61,82,100`, `docs/process/venus-d2d-worker.md:28` (also return date), `tools/publish/tests/test_publish.py:24` (sanitizer fixture) | high | must scrub (names a housemate, his PC and schedule) |
| P6 | Owner identity | `shatteredeagle` 13 files; `jesse` 70 files; `jesse@orion...` 21 lines; GitHub noreply email 4 lines | `tools/publish/push_pages.sh:2,6`, `docs/research/upstream-prs.md:5-6`, `docs/d2d-fork.md`, `docs/ci.md` | low | fine (owner is the publisher; GitHub handle and Pages repo `shatteredeagle/d-idle` are already public). `jesse@orion` ssh target lines go with P1 |
| P7 | Machine inventory | `orion/tower/herdr/venus/jade/onyx` by bare name: 16/22/2/9 commit messages, ~60 docs | `docs/process/forgejo-runners-plan.md` (full table: OS, cores, who owns), `docs/multi.md`, `docs/soak-hosts.md` | medium | decide: bare names are harmless once the domain and IPs are gone; the runner-plan table is the one doc to drop |
| P8 | Stream/Kick identity | `kick.com/diddle_bot` 20 lines | `docs/streamctl.md`, `tools/stream/*` | none | fine (public channel) |
| P9 | Commit messages | 0 hostnames-with-domain, 0 IPs, 0 home paths, 0 names (V `git log --all --format=%s%n%b`) | bare machine names only (P7) | none | fine |
| P10 | Authors | 2007 commits `Claude <noreply@anthropic.com>`, 2 `git stash` commits (`refs/stash`, 598782d) | | low | fine; drop `refs/stash` from any export |
| L1 | Extracted Blizzard tables | 2 files, 170 KB | `internal/gear/data.json` (skills/weapons/misc rows), `internal/gear/armor.json` (armor.txt rows); generators `internal/gear/gen*.py`; derived `internal/gear/builds/*.json`, `internal/plandata/*.json`, `docs/plan/data/*.json` | high | decide: THIRD_PARTY.md says tables are never committed, but these are a transformed subset (names, costs, reqs). Options: keep (common in open D2 tooling, facts not expression) or generate at setup from the user's MPQ (the code path exists: `internal/data/extract.go`) and gitignore the JSON |
| L2 | Game memory fixtures | 3 `.snap` (23+23+8 KB) + jsonl snapshots | `internal/d2/testdata/town.snap`, `internal/questmem/testdata/quests-i5.snap` | low | fine (decoded struct data, no code pages; U: confirm the .snap writer copies only unit/stat fields) |
| L3 | Game screenshots | 2 PNG | `docs/publish/scene-1280.png`, `scene-tile.png` | low | fine (stream scene; standard fair-use screenshot) |
| L4 | Decompiled/patched code | 0 C/C++/.patch/.diff files, 0 diff hunks in any .md (V), 0 Ghidra artefacts; d2d notes are own summaries with path citations | | none | fine; AGENTS.md patch rule holds |
| L5 | Third-party text | redivivus 6 files (numbers/paths only), Kolbot/Tyniann ideas only, zlib `explode.go` notice present | `THIRD_PARTY.md` accurate; Tyniann terms still UNVERIFIED | low | decide before go-live: verify Tyniann or drop the row |
| L6 | SPDX headers | 0 of 1058 Go files, 1 of 152 py/sh | policy `docs/process/spdx.md` says "not mass-edited" | low | fine (LICENSE at root suffices legally); optional one-shot header pass |
| L7 | Committed binaries in history | 5 blobs, 40 MB | `cmd/d2soak/d2soak` (0b06c6a), `combatlive` (166340c), `act2live` (5be9796), `townlive` (a21a663), `levelerlive` (3c81bd1); all removed next commit; 4 of 5 reachable from master | medium | decide: harmless (own Go binaries) but 40 MB of clone weight; filter-repo drops them |
| H1 | README | missing at root | draft: `docs/research/README-public-draft.md` | high | must add |
| H2 | AGENTS.md | one file mixes public rules with private ops | sections "The live games", "Memory limits on herdr", "Status board", d2board URL, Orion ssh, instance table | medium | split (section 4) |
| H3 | `.claude/`, `.forgejo/` | 8 + 1 files | `.claude/settings.json` (one local hook, fine), `.claude/agents/*` (procedure; `d2-researcher.md` names LAN hosts), `ci.yml:1` names the Orion runner | low | keep; scrub the two host mentions |
| H4 | Volume | docs 399 files/4.6 MB, openspec 289 files, 95 archived cards, 174 agent worktree branches | | low | fine for a working repo; a stranger needs README + docs/INDEX.md as the entry |
| H5 | Private-infra scripts | 32 of 165 `tools/` files | `tools/live.sh`, `watch.sh`, `sentinel.sh`, `soak/*`, `ha-stream/*`, `stream/*`, `ci/runner/*` | medium | decide: keep as "maintainer tooling" (documented as such) or exclude from the public copy |

Totals: secrets 0 (tree) / 0 (history); must scrub 4 classes (P2, P3, P5, H1); decide 7 (P1, P4, P7, L1, L5, L7, H5); fine 10.
Scanners: gitleaks/trufflehog not installed (not installed per brief); `git-filter-repo` is present at `~/.local/bin`.

## 2. Publish options

| option | what | effort (agent-h) | risk |
|---|---|---|---|
| (a) as-is | flip Forgejo to public, push the same history to GitHub | 1 (README, P5 edits) | P1-P7 stay in 2000 commits forever; 40 MB of binaries; 438 branches of agent WIP; cannot be undone once cloned |
| (b) scrub in place | `git filter-repo` on Forgejo's history (paths: L7 binaries; strings: P1 domain, P2 IPs, P3 home path, P5 name, P4 drive name), force-push, every worktree re-cloned | 4-6 + a half day of downtime for 170 worktrees and all agents | rewrite of a live multi-agent repo; AGENTS.md forbids force-push; filter-repo string replace cannot fix prose that *describes* the LAN (P7, runner plan) |
| (c) clean snapshot, fresh history | Forgejo stays the working remote with full history; an export script produces a sanitised tree and commits it to a new public repo (GitHub `shatteredeagle/d2lod-bot`), one squashed "snapshot" commit per release or per merge | 3-4 to build + test the exporter, 1 per release | two repos to keep in sync (one-way, scripted); public history has no blame; contributors send patches to the public repo and the coordinator ports them |

**Recommendation: (c)**, for the same reasons the status page already works this way (`tools/publish`: dry-run generator,
allow-list, sanitizer tests, gated push to `shatteredeagle/d-idle`). It keeps the private history intact (no force-push, no
agent disruption), makes "what is public" a reviewable, tested transform, and lets the public repo start with a clean README
and a clean `git log`. If history later needs to be public (for blame or GPL "corresponding source" of an old binary), (b) can
still be done on a copy. Private patches never left the tree, so the public snapshot is complete corresponding source for every
binary built from it (ADR 0010: the fork patches are in the *image*, which is not distributed).

## 3. Export / sanitise script design (`tools/publish/export_repo.sh`, not written)

Dry run by design, like `tools/publish/publish.py`: reads the Forgejo checkout, writes `~/d2lod-status/public-clone/`, pushes
only with `D2_PUBLISH_APPROVED=yes` and only after its own sanitizer check passes in the same invocation.

1. `git archive master` into a temp dir (tracked files only; `refs/stash`, branches, `.git` never leave).
2. **Drop list** (`tools/publish/export-exclude.txt`, one glob per line, reviewed in the repo): `tools/ha-stream/`, `tools/ci/runner/`,
   `tools/soak/layout.sh`, `tools/sentinel.sh`, `tools/watch.sh`, `docs/boardhost.md`, `docs/soak-hosts.md`,
   `docs/process/forgejo-runners-plan.md`, `docs/process/venus-d2d-worker.md`, `docs/process/orion-*.md`, `docs/tower-container.md`,
   `docs/multi.md`, `docs/g1-log.md` (decide), `openspec/changes/archive/` (decide: 95 folders of agent briefs), `.claude/agents/`.
3. **Rewrite list** (`export-rewrite.sed`, ordered): `*.lan.birbhub.com` -> `<host>.lan` with the bare machine name kept
   (`orion.lan.birbhub.com` -> `orion.lan`); `192.168.N.N` and `10.N.N.N` -> `<lan-ip>`; `/home/jesse` -> `~`; `/mnt/Omlette` ->
   `/mnt/cold`; `jesse@` -> `<user>@`; the housemate's name -> `a housemate`. Applied to text files only (`git grep -I` list).
4. **Split AGENTS.md**: public `AGENTS.md` = sections Ownership, Ownership of cards, Where you work, Facts and honesty, Doc rules,
   Commit often, Agent skills; private `docs/ops/AGENTS-private.md` (dropped by the exporter) = The live games, Orion capacity, Memory
   limits on herdr, Status board, Model routing. Done once in the private repo, then the exporter only drops the private file.
5. **README**: copy `docs/research/README-public-draft.md` to `README.md` (the private repo can carry it at the root too).
6. **Sanitizer gate** (`tools/publish/tests/test_export.py`, reusing the classes in `test_publish.py`): fail if any exported text
   file matches the private-infra regexes (`birbhub\.com`, `192\.168\.`, `/home/jesse`, `Omlette`, the name list, `ghp_`, `BEGIN .*PRIVATE KEY`),
   if any file is over 1 MB, or if any `.mpq/.d2s/.key` is present (reuse `tools/ci/check.sh --hygiene-only`).
7. **Commit**: in the public clone, replace the tree, `git add -A`, commit `Snapshot <date> from <short private hash>` with the
   CHANGELOG section of the day as body; push `main`. Idempotent (no commit when nothing changed, as `push_pages.sh` does).
8. **Back-port**: contributions to the public repo come in as PRs; the coordinator applies them to Forgejo with `git am` and the next
   snapshot carries them. The public commit message cites the public PR number so attribution survives.

## 4. User decisions needed

1. Option (c) (recommended) vs (a)/(b). If (b): accept the force-push and worktree reset.
2. L1: keep `internal/gear/*.json` (extracted table subset) in the public repo, or move to setup-time extraction.
3. P1/P7: scrub the `birbhub.com` domain only (bare `orion`/`tower` stay) or also the machine names and the lab docs.
4. H5: ship the Orion/Tower maintainer scripts as "maintainer tooling" or exclude them.
5. Target: GitHub `shatteredeagle/d2lod-bot` (new) vs making Forgejo itself public (Forgejo needs a public hostname: today it is LAN-only).
6. L5: verify the Tyniann licence or drop its row from THIRD_PARTY.md; optional SPDX header pass (L6).
7. Whether `openspec/changes/archive/` and `docs/archive/` go public (project history, large, harmless) or only live docs.

## 5. Go-live checklist (day of)

- [ ] Decisions 1-7 recorded in `docs/decisions.md` (coordinator).
- [ ] `docs/process/forgejo-runners-plan.md` and `docs/process/venus-d2d-worker.md`: remove the third person's name and schedule (P5) in the private repo too.
- [ ] AGENTS.md split merged (section 3.4); README.md at the root; CONTRIBUTING section in README says "agents-first, patches welcome, no game data".
- [ ] THIRD_PARTY.md: Tyniann row verified or removed; LICENSE present (agent/licensedecide merged).
- [ ] `tools/publish/export_repo.sh` dry run: 0 sanitizer hits, diff of the exported tree reviewed by the user once.
- [ ] `tools/ci/check.sh --hygiene-only` and `go build ./...` pass on the exported tree (stdlib only, no private paths needed).
- [ ] Public repo created empty, GPL-3.0-or-later shown by GitHub, default branch `main`, no Actions secrets, Issues on, Wiki off.
- [ ] First snapshot pushed with `D2_PUBLISH_APPROVED=yes`; clone it fresh on another machine and build.
- [ ] `docs/publish.md` and ADR 0010 addendum updated with the public URL; `tools/publish` status page links to it.
- [ ] Nothing about the d2d fork patches is in the public tree (`git grep -c lockstep` hits are design notes only, V today).
