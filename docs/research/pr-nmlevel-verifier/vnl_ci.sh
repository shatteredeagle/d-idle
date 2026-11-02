#!/bin/bash
# verifynmlevel (the worker host): the CI parity checks, one process at a time, niced.
#   vnl_ci.sh warn   g++ and clang++ -fsyntax-only on the PR's .cpp files, base and head, with the CI flags plus
#                    -Wshadow -Wrange-loop-construct -Wconversion; $S/warn_<tree>_<cc>.txt
#   vnl_ci.sh tidy   the CI's "Names and includes" selection (tools/tidy-pr-select.py on the head build) and
#                    clang-tidy on every selected TU, one at a time; $S/tidy_head.txt; each failing TU again on base
#   vnl_ci.sh tests  the suites named in the brief, ctest, pr_guard and presubmit --quick on base and head
set -o pipefail
F=$HOME/d2d-lab/fork
S=/tmp/d2agent-verifynmlevel
BASE=6935cf64 HEAD=9672fc6c
FILES="components/game/ai.cpp components/game/fight/tick.cpp components/game/world/objects.cpp components/game/world/quests/act3.cpp components/game/world/quests/act4.cpp components/game/world/quests/act5.cpp tests/game/test_boss_level.cpp tests/monsters/test_monsters.cpp"
EXTRA="-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion -Wno-error"
cmd_of() {  # build dir, file -> the compile command without -o / -c
  python3 - "$1" "$2" <<'PY'
import json, shlex, sys, os
build, f = sys.argv[1], sys.argv[2]
for e in json.load(open(os.path.join(build, "compile_commands.json"))):
    if e["file"].endswith("/" + f):
        args = shlex.split(e["command"]) if "command" in e else e["arguments"]
        out, skip = [], False
        for a in args[1:]:
            if skip: skip = False; continue
            if a in ("-o", "-c"): skip = a == "-o"; continue
            if a.startswith("-W") or a == "-Werror": continue
            out.append(a)
        print(e["directory"]); print(shlex.join(out)); break
PY
}
case $1 in
warn)
  for t in base head; do
    for cc in g++ clang++; do
      : > $S/warn_${t}_$cc.txt
      for f in $FILES; do
        mapfile -t c < <(cmd_of $F/build-vnl-$t $f)
        [ -n "${c[0]}" ] || { echo "no command for $f" >> $S/warn_${t}_$cc.txt; continue; }
        ( cd "${c[0]}" && eval nice -n 15 $cc ${c[1]} -fsyntax-only $EXTRA ) >> $S/warn_${t}_$cc.txt 2>&1
      done
      echo "$t $cc: $(grep -c 'warning:' $S/warn_${t}_$cc.txt) warnings, $(grep -c 'error:' $S/warn_${t}_$cc.txt) errors"
    done
  done ;;
tidy)
  cd $F/wt-vnl-head || exit 2
  git diff --name-only $BASE $HEAD | grep -E '\.(cpp|hpp)$' > changed.txt
  BUILD_DIR=$F/build-vnl-head python3 tools/tidy-pr-select.py > $S/tidy_files.txt; rm -f changed.txt
  echo "selected $(wc -l < $S/tidy_files.txt) TUs"
  : > $S/tidy_head.txt
  while read -r tu; do
    if nice -n 15 clang-tidy -p $F/build-vnl-head --quiet --extra-arg=-Wno-unknown-warning-option "$tu" > $S/tidy_one.txt 2>&1; then echo "OK $tu" >> $S/tidy_head.txt
    else echo "FAIL $tu" >> $S/tidy_head.txt; grep -E "warning:|error:" $S/tidy_one.txt | sed 's/^/    /' >> $S/tidy_head.txt; fi
  done < $S/tidy_files.txt
  echo "head: $(grep -c ^OK $S/tidy_head.txt) ok, $(grep -c ^FAIL $S/tidy_head.txt) fail"
  : > $S/tidy_base.txt
  grep ^FAIL $S/tidy_head.txt | while read -r _ tu; do
    btu=${tu/wt-vnl-head/wt-vnl-base}
    if nice -n 15 clang-tidy -p $F/build-vnl-base --quiet --extra-arg=-Wno-unknown-warning-option "$btu" > $S/tidy_one.txt 2>&1; then echo "OK $btu" >> $S/tidy_base.txt
    else echo "FAIL $btu" >> $S/tidy_base.txt; grep -E "warning:|error:" $S/tidy_one.txt | sed 's/^/    /' >> $S/tidy_base.txt; fi
  done
  echo "base (head's failing TUs): $(grep -c ^OK $S/tidy_base.txt) ok, $(grep -c ^FAIL $S/tidy_base.txt) fail" ;;
tests)
  export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
  export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
  for t in base head; do
    B=$F/build-vnl-$t
    for x in test_game test_combat test_rules test_game_rules test_monsters test_monster_ai test_monster_thinks test_boss_level; do
      p=$(find $B/tests -maxdepth 2 -name $x -type f | head -1)
      [ -n "$p" ] || { echo "$t $x: not built"; continue; }
      ( cd $B/tests && timeout 1200 nice -n 15 "$p" > $S/test_${t}_$x.txt 2>&1 ); echo "$t $x: exit $? ($(tail -1 $S/test_${t}_$x.txt | cut -c1-120))"
    done
    ( cd $B && timeout 2400 nice -n 15 ctest -j1 --timeout 600 > $S/ctest_$t.txt 2>&1 ); echo "$t ctest: exit $? $(grep -E 'tests passed|tests failed' $S/ctest_$t.txt)"
    grep -E '^\s+[0-9]+ - ' $S/ctest_$t.txt | sed "s/^/  $t failed: /"
  done
  cd $F/wt-vnl-head || exit 2
  ( BUILD_DIR=$F/build-vnl-head timeout 900 nice -n 15 python3 tools/pr_guard.py $BASE $HEAD > $S/pr_guard.txt 2>&1 ); echo "pr_guard: exit $? $(tail -2 $S/pr_guard.txt | tr '\n' ' ')"
  ( BUILD_DIR=$F/build-vnl-head timeout 1800 nice -n 15 python3 tools/presubmit.py --quick --base $BASE > $S/presubmit.txt 2>&1 ); echo "presubmit --quick: exit $? $(tail -2 $S/presubmit.txt | tr '\n' ' ')" ;;
esac
