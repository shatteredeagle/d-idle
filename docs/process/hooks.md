# Hooks (change card `hooks`)

> status: current since OpenSpec removal stage 5 (2026-11-04): changecheck and `openspec/` are gone; the pre-push hook runs the warn-only leak scan only. Current truth: docs/decisions.md.

Tags: V = run in the hooks branch (hook fed sample stdin, fake `go` shim for the failing case, `jq empty`, timing), S = Claude Code docs knowledge, U = not tested.

## Install, bypass, remove
- Install: `tools/install-hooks.sh` sets `core.hooksPath=tools/hooks` in the shared `.git/config`, so the main clone and every worktree use it (relative path: each worktree runs its own checkout of the hooks). It does not touch `~/.gitconfig`.
- Bypass one push: `D2_SKIP_HOOKS=1 git push forge master` (or `origin main`) (prints a loud note). There is no other bypass; do not use `--no-verify`.
- Remove: `tools/install-hooks.sh --remove` (unsets `core.hooksPath`).

## pre-push (`tools/hooks/pre-push`)
- Acts only when a pushed ref is `refs/heads/master` or `refs/heads/main` (private repo vs public repo); any other push exits 0 at once (V, sample stdin).
- Runs the warn-only leak scan (`go run ./cmd/leakcheck -mode warn`). It never blocks. No tests, no whole-repo vet (the head box has 4 GB).
- A failing check blocks the push (V, shim `go` returning 1 gives exit 1).

## Session brief (`tools/session-brief.sh`)
POSIX sh, about 7 ms (V, `time`), no network, no go. Prints `needs you: N (now: ids)` from `docs/needs-you.md`. Wired in `.claude/settings.json` (private overlay, `tools/ops.sh link` links it from the ops repo) as the `SessionStart` command hook (`$CLAUDE_PROJECT_DIR`). SessionStart stdout is added to the session context (S); whether the user also sees it in the UI is U. Remove the hook by deleting the `hooks` key of `.claude/settings.json`.
