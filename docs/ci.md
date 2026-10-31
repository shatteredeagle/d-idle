# CI (GitHub Actions)

> status: CI is `.github/workflows/ci.yml` on GitHub (`shatteredeagle/d-idle`, branch `main`). The earlier self-hosted workflow and its runner moved to the private ops repo; the text below describes that earlier setup and its failure history.

Earlier workflow (self-hosted forge, now private): `.forgejo/workflows/ci.yml` (the forge at forge.example), served by the worker host runner (label `linux`, host executor as
the worker user, systemd user unit `forge-runner`, Nice=10, capacity 2). Status: V runs on the worker host since late October; the first ~390 runs
failed or were cancelled (see "Why it failed" below), reworked 2026-10-30 by CIFIX; the first run of the reworked file is U.

## When it runs
- push to the default branch (`main` on GitHub, `master` on the private archive), tags `v*`, every pull request, and manually (`workflow_dispatch`). NOT on other branch pushes (agent branches get
  their checks locally, `tools/hooks`, and at the PR/merge).
- `concurrency: ci-<ref>` with cancel-in-progress: a newer push cancels the older run of the same ref.

## Jobs
- `checks` (light, always, about 5 s on the worker host): `go run ./cmd/changecheck -cli` (cards, owners, merges; runs `tools/openspec.sh validate
  --all --strict` when node exists), gofmt, `go vet ./cmd/changecheck`, `tools/ci/check.sh --hygiene-only`.
- `test` (heavy, skipped when the diff to the base has only `docs/`, `openspec/` or top-level `*.md` files): `tools/ci/check.sh`
  (hygiene, gofmt, `go vet ./...`), `PAR=2 tools/ci/test.sh` (`go test -short -p 2 ./...` minus the quarantine list in
  `tools/ci/flaky.txt`, each quarantined test retried once; includes the knobdump drift test), `tools/ci/scripts.sh` (`bash -n` on
  every tracked `.sh`, `tools/winereap_test.sh`, `tools/container-checklist.sh`, `packaging/release.sh --dry-run`). All steps run under
  `nice -n 10`, job timeout 30 min.
- `release` (tag `v*`, after `checks` and `test`): `packaging/release.sh $TAG`, uploads `dist/*.tar.gz` and `.sha256` as a workflow
  artifact and, if the repo secret `RELEASE_TOKEN` (write:repository) exists, creates the forge release and attaches both files.

## Why it failed before (V)
The `Scripts` step ran `tools/winereap_test.sh`, which sets a fake `XDG_RUNTIME_DIR` but still took the REAL legacy lock
`/tmp/d2lod-game-i4.lock` that a running soak on the worker host holds for hours: every one of its ~40 `winereap.sh` calls waited 10 s
(reproduced on the worker host: minutes instead of 1 s, `i4: instance lock busy` in a loop). The test now sets `REAP_NOLOCK=1` except in the lock
test (which only needs a held lock to bail out or continue). Reproduced fix: scripts step 30 s, exit 0. The cancelled runs were
superseded or killed pushes, not a second cause (U: The forge run logs were not readable from the runner; the runner journal only
has fetch errors from the forge outages).

## Read results
Forge repo page, Actions tab (`/<owner>/d2lod-bot/actions`); a red `checks` is a card/owners/format problem, a red `test` shows
the failing step. Runner log: `ssh worker journalctl --user -u forge-runner -n 100` (step output is not in it).

## Run the same locally
```sh
go run ./cmd/changecheck -cli && gofmt -l $(git ls-files '*.go') && go vet ./cmd/changecheck && tools/ci/check.sh --hygiene-only   # checks
tools/ci/check.sh && PAR=2 tools/ci/test.sh && tools/ci/scripts.sh                                                              # test
```
On the head box (4 GB) do not run the `test` commands; use `labrun worker -- ...`. Keep the runner env in mind when a CI-only failure shows
up: `env -i HOME=$HOME PATH=/usr/bin nice -n 10 bash tools/ci/scripts.sh` on the worker host; real instance locks and soaks are live there.

## What the runner needs
- A forge-runner with label `linux` running jobs on the host. Go 1.27.3, bash, git, curl, python3, gzip/tar/sha256sum; node >= 20.19
  is optional (openspec validate is skipped without it). No docker.
- `actions/checkout@v4` and `actions/upload-artifact@v3` must be reachable (the forge mirrors them via its default actions URL).
- Registering the runner needs a registration token (admin action, not done here).

## Game tables absent
CI has no `/tmp/d2mpq`. V: on the head box (no `/tmp/d2mpq`), `go test -short -count=1 -p 1 <pkg>` for each of the 94 packages
printed no failure: every sim/real-table test skips cleanly. The `-race` and `-count=3` runs of the old workflow were dropped for speed; run them locally.

## The NAS runner notes
- Run 397 (push b87c2ee): `test` failed only in the Scripts step. Go tests and gate passed. `tools/container-checklist.sh` printed
  `FAIL 6 slot Wizbot ...` and `FAIL 7 slot restored once`. Cause (reproduced in the runner container as uid 99): d2fleet admission
  refused the slot, `d2fleet: admission refused (load): load average 5.8 is too high for another instance on 4.0 cores (limit 5.0)`.
  In a container load1 is the host's, and the NAS is busy (tests run niced -p 2 on 4 cores). Fix: tools/ci/scripts.sh sets
  `D2FLEET_MAX_LOAD_FACTOR=-1` (admission off) for that fake-mode HTTP test.
- Task logs: The NAS `/mnt/user/appdata/forge/gitea/actions_log/<owner>/d2lod-bot/<xx>/<task id>.log.zst` (`zstd -dc`); read via root ssh (docs/boardhost.md).
- Read-only log command (V, works from the head box): `ssh -i ~/.ssh/id_ed25519_nas -o IdentitiesOnly=yes -o BatchMode=yes <nas-user>@nas.example 'cd /mnt/user/appdata/forge/gitea/actions_log/<owner>/d2lod-bot; ls -t */*.log.zst | head; zstd -dc <xx>/<id>.log.zst | tail -60'`.
- Verified: workflow_dispatch of the fix (tasks 1187 checks, 1188 test) both succeeded on the NAS; dispatch API `POST .../actions/workflows/ci.yml/dispatches` works on this forge (204).
