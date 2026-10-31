# Hooks (change card `hooks`)

Tags: V = run in the hooks branch (hook fed sample stdin, fake `go` shim for the failing case, `jq empty`, timing), S = Claude Code docs knowledge, U = not tested.

## Install, bypass, remove
- Install: `tools/install-hooks.sh` sets `core.hooksPath=tools/hooks` in the shared `.git/config`, so the main clone and every worktree use it (relative path: each worktree runs its own checkout of the hooks). It does not touch `~/.gitconfig`.
- Bypass one push: `D2_SKIP_HOOKS=1 git push forge master` (or `origin main`) (prints a loud note). There is no other bypass; do not use `--no-verify`.
- Remove: `tools/install-hooks.sh --remove` (unsets `core.hooksPath`).

## pre-push (`tools/hooks/pre-push`)
- Acts only when a pushed ref is `refs/heads/master` or `refs/heads/main` (private repo vs public repo); any other push exits 0 at once (V, sample stdin).
- Runs `go run ./cmd/changecheck -cli`: it exits 1 for check groups whose date in `openspec/.strict.yaml` has passed (cards/overlap/spec/owners/status from 2026-11-03, merge/diff from 2026-11-06), with a message naming the group. Before the date the warnings are printed but do not block.
- Then `tools/openspec.sh validate --all --strict` only if `node` is on PATH, else a note. No tests, no whole-repo vet (the head box has 4 GB).
- Writes `openspec/.last-check` (gitignored) as `DATE N warnings rc=R`; the session brief shows it.
- A failing check blocks the push (V, shim `go` returning 1 gives exit 1).

## Strict dates (`openspec/.strict.yaml`)
`cards_overlap_fatal_from`, `merges_diff_fatal_from` (YYYY-MM-DD) and `allow_merges` (old merge hashes that name no change, never reported). `changecheck -cli -today YYYY-MM-DD` injects the date. A missing file means never strict. Plain `changecheck` (without `-cli`) stays warn-only; `-strict` still means any warning.

## Session brief (`tools/session-brief.sh`)
POSIX sh, about 7 ms (V, `time`), no network, no go. Prints `needs you: N (now: ids)` from `docs/needs-you.md` and `changecheck: ...` from `openspec/.last-check`. Wired in `.claude/settings.json` (private overlay, `tools/ops.sh link` links it from the ops repo) as the `SessionStart` command hook (`$CLAUDE_PROJECT_DIR`). SessionStart stdout is added to the session context (S); whether the user also sees it in the UI is U. The dispatcher refreshes `.last-check` by running `changecheck -cli` through the pre-push hook on its master push. Remove the hook by deleting the `hooks` key of `.claude/settings.json`.
