# Tester image on ghcr.io: publish prep (TASK-120)

> status: PREPARED, 2026-11-08. NOTHING is published: no workflow has run, no package exists, no tag or environment was created. This is Tier 1 of docs/research/self-update-design.md. Tags: V verified (how named), S secondary, U unverified.

## 1. What exists

| Item | File | State |
|---|---|---|
| Workflow | `.github/workflows/tester-image.yml` | written, inert (only `workflow_dispatch`) |
| Compose | `deploy/d2d-container/compose.tester.yml` | `image: ${D2_IMAGE:-${D2D_IMAGE:-d2dbot:tester}}`; unset = the local build tag, nothing changes for local builds |
| Static tests | `deploy/d2d-container/tester_image_test.go` | `TestTesterImageWorkflow`, `TestComposeImageVariable` (no secret, dispatch only, environment gate, pinned actions, no push without confirm) |

The workflow has two jobs. `build` always runs: free disk, leakcheck, the Go static tests, the entrypoint dry run (`smoke.sh` with `DRYRUN_ONLY=1`), `context.sh` (public inputs, sha256 checked), an amd64 image build (loaded, NOT pushed), a scan of the built image for game data (`*.mpq`, `game.exe`, `*.d2s`, `d2*.dll`), a check that the licence files and both binaries are present. `publish` runs only when the input `confirm` equals `publish`, then waits for the required reviewer of the environment `tester-publish`; it logs in with `GITHUB_TOKEN` (permission `packages: write`, only in this job), rebuilds from the same cache and pushes the tags `X.Y.Z`, `latest`, `d2d-<fork commit>`, `sha-<bot commit>` with the labels `org.opencontainers.image.source` (this repo), `.licenses`, `.revision`, `.version`, `.title`, and `io.didle.fork.source` / `io.didle.fork.commit`. Provenance and SBOM attestations are off (`provenance: false` keeps the registry page to one plain manifest); add `id-token: write` and `attestations: write` later if wanted. No secret other than `GITHUB_TOKEN` is used (the test enforces it).

Differences from the design doc (2.2): `workflow_dispatch` only (no `tester-v*` tag trigger until a tag ruleset exists); the fork is fetched by `context.sh` from the public tag `didle-gate2-lab3` (pinned commit check inside the script) instead of a second `actions/checkout`, so there is one pin, not two. Action versions are major pins with a comment; switch to commit SHAs for stricter pinning (the test accepts both).

## 2. Dry run on the worker host (the workflow's commands, no GitHub)

Commands (worker host, nice 19, throwaway tag, image and context removed afterwards):
```
IMAGE=d2dbot:ghcrprep-dry PROFILE=tester CTX=/tmp/ghcrprep-ctx BUILD_JOBS=4 nice -n 19 deploy/d2d-container/smoke.sh
deploy/d2d-container/context.sh /tmp/ctx2 && nice -n 19 docker build --no-cache --build-arg BUILD_JOBS=4 -t d2dbot:ghcrprep-cold /tmp/ctx2
```
Results (V, run 2026-11-08 on the worker host, 16 cores):
- `smoke.sh` PASS with `PROFILE=tester`: tester dry run, build, ping, create + save, persist after restart, first-run wizard and watch token (mode 600), finished wizard skipped. That run reused Docker layers, so its 16 s build time is NOT a cold number. Image 410 MB.
- Cold build (`--no-cache`, nice 19, 4 jobs, vendored archives already cached, so download time excluded): context 2 s, `docker build` 370 s, image 409.7 MB (V). Both containers and the throwaway image were removed; the build cache was pruned.
- The build context is 255 MB (fork source, vendored archives, our Go binaries).
- One defect found and fixed: a transient download failure of the FFmpeg archive made `context.sh` report "sha256 mismatch" (the curl error was swallowed). `context.sh` now retries (5 times, all errors) and says "download of URL failed". The hash itself is right (V: sha256 of a fresh download equals the pin).
- NOT proven: the hosted runner (4 vCPU, about 14 GB free disk before the cleanup step, S GitHub docs), `type=gha` cache behaviour, the ghcr push, the labels on a real registry page. U until the first run.

Runner estimate (U): the same compile on 4 vCPU without the nice 19 handicap should take about 6 to 12 minutes plus the cache export; the job limit is 120 min. Disk: context 255 MB, a few GB of build trees in the build stage, final image 410 MB; the cleanup step frees about 25 GB, so disk is not expected to bind (U). If it does the job fails with "no space left" and nothing is pushed.

## 3. What the user must do (in this order)

1. Merge this branch (the merge pipeline does it). The workflow file needs a token with the `workflow` scope; the merge result says whether the push was accepted.
2. GitHub repo settings, Environments: create `tester-publish`, add yourself as required reviewer, optionally restrict it to the default branch. Create it BEFORE the first dispatch; without it the publish job would run unreviewed once `confirm=publish` is typed.
3. Decide package visibility. A package pushed by a workflow starts private (S). Friends can pull only if the package is public (Package settings, Change visibility) or each gets read access; friends without GitHub accounts need public. The label `org.opencontainers.image.source` links the package to this repo.
4. First run as a dry run: Actions, tester-image, Run workflow, version `0.0.1`, confirm empty. Read the time and disk numbers in the log.
5. Real run: the same form with `confirm` = `publish`; approve the environment prompt. Then go through section 6.
6. Later, optional: a tag ruleset for `tester-v*` and a `push: tags` trigger (the design doc wants it; not added because it would run the workflow without the ruleset).

## 4. How friends pull and update

First time, compose file plus `.env` (or exported variables):
```
D2_IMAGE=ghcr.io/shatteredeagle/didle-tester:latest   # or :0.1.0 to pin a release
D2_GAME_DIR=/path/to/your/Diablo-II
D2_WATCH_ORIGIN=http://localhost:8891
docker compose -f compose.tester.yml pull
docker compose -f compose.tester.yml up -d
```
Update: `docker compose -f compose.tester.yml pull && docker compose -f compose.tester.yml up -d`. The character and the watch token live in the `d2dbot-data` volume and survive. Local build users set nothing: with `D2_IMAGE` unset the file keeps `d2dbot:tester`. The old `D2D_IMAGE` still works as a fallback (docs/d2d-container/README.md still names only that one; not edited here, not owned). The image is amd64 only; Apple silicon would run it under emulation (U, probably too slow).

## 5. GPL source offer for the image page (from docs/tester-licence.md sections 3 and 4)

```
This image contains d2d, a GPL-3.0-or-later program, built from:
  repository : https://github.com/shatteredeagle/d2d-didle
  tag/commit : didle-gate2-lab3 -> f75816c616c8253930d88277b90936e3eabe3bc8
  (the label io.didle.fork.commit and the tag d2d-<commit> name the exact commit of each image)
The bot and the build recipe (Dockerfile, entrypoint, compose) are in
https://github.com/shatteredeagle/d-idle at the commit in the label
org.opencontainers.image.revision. The list of vendored inputs with sha256 is in
/usr/share/licenses/d2d/SOURCE.txt inside the image, next to the GPL-3.0 text and upstream LEGAL.md.
If those locations are unavailable, write to <contact: fill at cut> and a copy of the
Corresponding Source will be sent on request for at least three years from the date you
received the image, at no more than the cost of the medium.
No Blizzard game data is included; you supply your own Diablo II 1.14d files.
Diablo II is a trademark of Blizzard Entertainment.
```
Open: the in-image `SOURCE.txt` written by `context.sh` still has the line "NOT FOR DISTRIBUTION until the corresponding source is offered"; docs/tester-licence.md says to drop it at the cut. This prep does not remove it (a licence-text decision). The open items of docs/tester-licence.md section 6 (GPL reviewer, Freepik icon, the user's go for the first outside copy) gate the first publish.

## 6. Checklist for the first publish

- [ ] docs/tester-licence.md section 6 answered (reviewer waived or named; the go for the first outside copy)
- [ ] "NOT FOR DISTRIBUTION" line and the contact placeholder in `context.sh`'s SOURCE.txt resolved
- [ ] environment `tester-publish` exists with a required reviewer
- [ ] dry run (confirm empty) green; time and disk noted
- [ ] real run approved; tags `0.1.0`, `latest`, `d2d-f75816c6...`, `sha-...` exist
- [ ] `docker buildx imagetools inspect ghcr.io/shatteredeagle/didle-tester:0.1.0` shows linux/amd64 and the labels
- [ ] package visibility set as decided; the page shows the source repo and the source offer
- [ ] clean-machine test: `pull`, `up -d`, the watch screen opens, a character is created from the friend's own game files
- [ ] the image digest noted in the release notes

## 7. Risks

- Runner disk and time: the fork build is the long step; a cold build is measured above, the hosted runner is not. The job limit is 120 min. The `type=gha` cache (10 GB per repo, evicted after 7 days unused, S) should keep later runs short.
- Limits: Actions minutes for a public repo are free (S). Pulls of public images have no published ghcr limit but abuse may be throttled (S). A private package counts against the account's package storage and transfer quota (S).
- Transient downloads: ffmpeg.org failed once during the dry run (now retried). If it is down at publish time the run fails before anything is pushed.
- Bad `latest`: every real run moves `latest`. Friends who pin `:X.Y.Z` are safe; the rollback is the previous version tag.
- A public package lets anyone pull; the image holds no game data and no secret (the scan step checks the first).
- The d2d binary is built fresh in the workflow from the public tag: same commit and flags as the Gate 2 evidence, different toolchain host (U: compare behaviour once, as the design doc says).
