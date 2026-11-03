#!/bin/bash
# verifyb2 (the worker host): read-only look at the batch 2 fork branches. Makes own clones
# under ~/d2d-lab/vb2 (a clone does not write to the fork checkout) and writes the diffs and
# commit metadata to $D2_SCRATCH. No push, no commit in the fork.
set -e
F=$HOME/d2d-lab/fork
V=$HOME/d2d-lab/vb2
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}
mkdir -p "$V" "$S"
F=$F/src
git -C "$F" rev-parse local/batch2-minmana local/batch2-teleport
if [ ! -d "$V/src/.git" ]; then
  git clone -q --no-checkout "$F" "$V/src"
  git -C "$V/src" fetch -q "$F" local/batch2-minmana:refs/heads/mm local/batch2-teleport:refs/heads/tp
fi
for b in mm tp; do
  git -C "$V/src" log --format='%H|%an <%ae>|%cn <%ce>|%s' 3e073888..$b > "$S/log_$b.txt"
  git -C "$V/src" log --format='=== %h%n%B' 3e073888..$b > "$S/body_$b.txt"
  git -C "$V/src" diff 3e073888 $b > "$S/diff_$b.txt"
  git -C "$V/src" log -p --stat --format='=== %h %s' 3e073888..$b > "$S/patches_$b.txt"
done
git -C "$V/src" ls-remote "$F" 2>/dev/null | grep -E 'batch2' || true
git -C "$F" branch -r --contains 82f8d3e4 2>/dev/null || true
git -C "$F" branch -r --contains d2572d6c 2>/dev/null || true
wc -l "$S"/diff_*.txt
echo done
