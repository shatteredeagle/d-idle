#!/bin/bash
# verifyb2 (the worker host): suites, extra warnings on added lines (g++ and clang++), and the
# throwaway merge with upstream pull/178, for one tree built by vb2_build.sh.
#   vb2_ci.sh NAME OWN_TEST
set -u
here=$(cd "$(dirname "$0")" && pwd)
NAME=$1; OWN=$2
V=$HOME/d2d-lab/vb2; T=$V/wt-$NAME; B=$V/build-$NAME
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}; O=$S/ci-$NAME; mkdir -p "$O"
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
echo "== suites ($NAME)"
for t in test_rules test_game_rules test_monsters test_monster_thinks test_monster_ai test_combat test_game test_sk_ground test_skills "$OWN"; do
  f=$B/tests/$t; [ -x "$f" ] || f=$(ls "$B"/tests/*/"$t" 2>/dev/null | head -1)
  if [ -z "$f" ] || [ ! -x "$f" ]; then echo "$t missing"; continue; fi
  (cd "$B" && timeout 1500 nice -n 19 "$f" > "$O/run-$t.log" 2>&1); echo "$t rc=$?"
done
echo "== extra warnings on added lines"
git -C "$T" diff -U0 3e073888 HEAD -- '*.cpp' '*.hpp' > "$O/added.diff"
git -C "$T" diff --name-only 3e073888 HEAD | grep -E '\.cpp$' > "$O/changed.txt"
FLAGS="-fsyntax-only -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Wshadow -Wrange-loop-construct -Wconversion -Wno-error"
for cxx in g++ clang++; do
  : > "$O/w-$cxx.log"
  for f in $(cat "$O/changed.txt") components/game/fight/tick.cpp components/game/fight/strike.cpp components/game/fight/missiles.cpp; do
    python3 "$here/vb2_cmd.py" "$B/compile_commands.json" "$T/$f" > "$O/cmd.txt" || { echo "no command for $f" >> "$O/w-$cxx.log"; continue; }
    (cd "$B" && nice -n 19 $cxx $(cat "$O/cmd.txt") $FLAGS "$T/$f") >> "$O/w-$cxx.log" 2>&1
    echo "rc=$? $f" >> "$O/w-$cxx.log"
  done
  grep -c "warning:" "$O/w-$cxx.log" | sed "s/^/$cxx warnings in all lines: /"
  grep "^rc=" "$O/w-$cxx.log"
done
python3 "$here/vb2_added.py" "$O/added.diff" "$O/w-g++.log" "$O/w-clang++.log"
echo "== merge with pull/178"
U=$(git -C "$HOME/d2d-lab/fork/src" remote get-url upstream 2>/dev/null || echo https://github.com/d2d-project/d2d)
git -C "$V/src" fetch -q "$U" pull/178/head:refs/heads/pr178 2>&1 | tail -2
git -C "$V/src" log -1 --format='pr178 %h %s' pr178
git -C "$V/src" merge-tree --write-tree --name-only pr178 "$(git -C "$T" rev-parse HEAD)" > "$O/merge.txt" 2>&1; echo "merge-tree rc=$?"
head -3 "$O/merge.txt"
git -C "$V/src" diff --stat 3e073888 pr178 | tail -1
git -C "$V/src" diff --name-only 3e073888 pr178 > "$O/pr178-files.txt"
git -C "$T" diff --name-only 3e073888 HEAD | grep -x -F -f "$O/pr178-files.txt" | sed 's/^/both touch: /'
