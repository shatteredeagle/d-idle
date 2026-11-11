set -u
# b2bundle: see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths below, as the other lab scripts).
T=$HOME/d2d-lab/wt-b2bundle; B=$HOME/d2d-lab/build-b2bundle; K=$HOME/d2d-lab/mdr2-tools; O=$HOME/d2d-lab/b2bundle-ci; mkdir -p $O
cd $T
BASE=3e073888
echo "== pr_guard per commit"
for c in $(git rev-list --reverse $BASE..HEAD); do echo "-- $(git log -1 --format='%h %s' $c)"; python3 tools/pr_guard.py $c~1 $c 2>&1 | tail -3; done
echo "== pr_guard range"; python3 tools/pr_guard.py $BASE HEAD 2>&1 | tail -3
echo "== presubmit --quick"
python3 tools/presubmit.py -C $T --base $BASE --quick 2>&1 | tail -14
echo "== warnings"
git diff -U0 $BASE HEAD -- '*.cpp' '*.hpp' > $O/added.diff
git diff --name-only $BASE HEAD | grep -E '\.cpp$' > $O/changed.txt
cat $O/changed.txt | tr '\n' ' '; echo
FLAGS="-fsyntax-only -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Wshadow -Wrange-loop-construct -Wconversion -Wno-error"
for cxx in g++ clang++; do
  : > $O/w-$cxx.log
  for f in $(cat $O/changed.txt) components/game/fight/tick.cpp components/game/fight/strike.cpp components/game/fight/missiles.cpp; do
    python3 $K/vb2_cmd.py $B/compile_commands.json $T/$f > $O/cmd.txt || { echo "no command $f" >> $O/w-$cxx.log; continue; }
    (cd $B && nice -n 19 $cxx $(cat $O/cmd.txt) $FLAGS $T/$f) >> $O/w-$cxx.log 2>&1
    echo "rc=$? $f" >> $O/w-$cxx.log
  done
  grep -c "warning:" $O/w-$cxx.log | sed "s/^/$cxx warnings (all lines): /"
done
python3 $K/vb2_added.py $O/added.diff $O/w-g++.log $O/w-clang++.log
echo "== clang-tidy (.clang-tidy: identifier-length, include-cleaner) on changed TUs"
: > $O/tidy.log
for f in $(cat $O/changed.txt); do nice -n 19 clang-tidy -p $B --quiet $T/$f >> $O/tidy.log 2>&1; echo "rc=$? $f"; done
grep -E "warning:|error:" $O/tidy.log | sort -u | head -20; echo "tidy diag lines: $(grep -cE 'warning:|error:' $O/tidy.log)"
echo "== merge with pull/178"
U=$HOME/d2d-lab/fork/src
git -C $U fetch -q upstream pull/178/head 2>&1 | tail -1
P=$(git -C $U rev-parse FETCH_HEAD); echo "pr178 $P"
git -C $U merge-tree --write-tree --name-only $P $(git rev-parse HEAD) > $O/merge.txt 2>&1; echo "merge-tree rc=$?"; head -3 $O/merge.txt
