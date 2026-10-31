#!/bin/sh
# HISTORICAL (cut done 2026-10-31; GitHub main is primary): kept for the record, the ops repo push below is for the private ops repo only.
# Quiet-master cutover for work package A (docs/process/public-split.md section 8). READ-ONLY: it prints the ordered steps
# and verifies state; it never pushes, merges, deletes or edits. Run from the master checkout: tools/ops-cutover.sh [check|steps]
ops=${D2_OPS_DIR:-$HOME/code/d2lod-ops}
branch=${D2_SCRUBA_BRANCH:-agent/scruba}
top=$(git rev-parse --show-toplevel) || exit 2
cd "$top" || exit 2
bad=0
ok() { echo "ok    $*"; }
no() { echo "TODO  $*"; bad=1; }

check() {
	echo "== state ($top)"
	[ "$(git rev-parse --abbrev-ref HEAD)" = master ] && ok "on master" || no "not on master"
	[ -z "$(git status --porcelain)" ] && ok "master tree clean" || no "master tree not clean"
	if [ -d "$ops/.git" ]; then
		ok "ops checkout $ops ($(git -C "$ops" rev-list --count HEAD) commits)"
		git -C "$ops" remote | grep -q . && ok "ops has a remote" || no "ops has no remote yet (step 2)"
		[ -f "$ops/AGENTS.private.md" ] && [ -f "$ops/local.env" ] && ok "AGENTS.private.md and local.env present" || no "ops repo incomplete"
	else
		no "ops checkout missing at $ops"
	fi
	git rev-parse --verify -q "$branch" >/dev/null && ok "branch $branch exists" || no "branch $branch missing"
	if git merge-base --is-ancestor "$branch" HEAD 2>/dev/null; then ok "$branch is merged into master"; else no "$branch not merged yet (step 3)"; fi
	echo "== worktrees"
	n=0; dirty=0; unmerged=0
	for w in $(git worktree list --porcelain | sed -n 's/^worktree //p'); do
		[ "$w" = "$top" ] && continue
		n=$((n + 1))
		[ -n "$(git -C "$w" status --porcelain 2>/dev/null)" ] && { dirty=$((dirty + 1)); echo "dirty      $w"; }
		b=$(git -C "$w" rev-parse --abbrev-ref HEAD 2>/dev/null)
		git merge-base --is-ancestor "$b" master 2>/dev/null || { unmerged=$((unmerged + 1)); echo "unmerged   $w ($b)"; }
	done
	echo "other worktrees: $n, dirty: $dirty, branch not in master: $unmerged"
	[ "$dirty" = 0 ] && [ "$unmerged" = 0 ] || bad=1
	echo "== systemd user units that call moved scripts (kept working by the stubs)"
	grep -l 'd2lod-bot/tools/\(publish\|agentstatus\)' "$HOME"/.config/systemd/user/*.service 2>/dev/null || echo "none"
	echo "== leakcheck"
	go run ./cmd/leakcheck -mode warn 2>&1 | sed -n 1p
	[ $bad = 0 ] && echo "READY" || echo "NOT READY"
	return $bad
}

steps() {
cat <<EOS
== steps, in order (quiet master: no agent is working)
1. Preconditions: tools/ops-cutover.sh check shows no dirty or unmerged worktree (record abandoned ones in docs/needs-you.md).
2. Create the private repo <owner>/d2lod-ops on the forge (web UI or API; needs the maintainer's credentials), then, from $ops:
     git remote add origin <ssh url of <owner>/d2lod-ops> && git push -u origin HEAD   # current branch of the ops repo
   (history was extracted with git filter-repo from a clone of master 57dd902; later changes to moved files on master since then
   must be replayed: git log 57dd902..master -- <moved paths>; the board file docs/needs-you.md is edited in ops from now on.)
3. Merge $branch into master (git merge --no-ff $branch). Run: tools/ops.sh status ; sh tools/ci/check.sh --hygiene-only ;
   go build ./... ; go run ./cmd/changecheck -cli.
4. In the master checkout run: tools/ops.sh link  (CLAUDE.local.md, local.env, .claude/agents, docs/needs-you.md).
5. Existing worktrees: for each (git worktree list) first merge or rebase master into the branch, then tools/ops.sh link there:
     for w in \$(git worktree list --porcelain | sed -n 's/^worktree //p'); do git -C "\$w" merge --no-edit master && (cd "\$w" && tools/ops.sh link); done
   (run it by hand after reading the check output; this script does not do it). Agents with open branches: their rebase deletes the
   moved files, which is intended.
6. Other machines (the worker host): clone $ops there at ~/code/d2lod-ops, then git pull the public checkout (the pull removes the
   moved scripts; the stubs call ops.sh run, which needs that ops clone). Remote scripts that cat tools/lib-paths.sh stay public.
7. systemd: d2-agentstatus.service (agentstatus.py serve 8897), d2-publish.service (publish/run_publish.sh), d2-cards.service
   (agentstatus/publish-cards.sh) keep working through the public stubs, no unit edit needed on day 1. Check:
   systemctl --user restart d2-agentstatus && systemctl --user start d2-publish && journalctl --user -u d2-publish -n 3.
   Later (optional): point ExecStart at %h/code/d2lod-ops/tools/... with Environment=D2_REPO=%h/code/d2lod-bot, delete the stubs.
8. Claude Code: the user's ~/.claude/CLAUDE.md may import $ops/AGENTS.private.md once as the fallback for launch dirs without a link.
9. Then continue with packages B, C, D of docs/process/public-split.md.
EOS
}

case ${1:-check} in
check) check; r=$?; echo; steps; exit $r ;;
steps) steps ;;
*) echo "usage: ops-cutover.sh [check|steps]" >&2; exit 2 ;;
esac
