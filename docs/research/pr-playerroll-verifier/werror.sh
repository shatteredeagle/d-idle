#!/bin/bash
# werror.sh base|head COMPILER: the changed TUs (and one TU that includes combat.hpp) with
# CI's flags plus -Wshadow -Wrange-loop-construct -Wconversion, -O2, nice, one at a time.
w=$1; cxx=$2; R=~/d2d-lab/verifyplayerroll; B=$R/build-$w
out=$R/werror-$w-$(basename $cxx).log; : > $out
for tu in tests/test_combat.cpp tests/game/test_missile_hit.cpp components/game/fight/missiles.cpp; do
  [ -f $R/$w/$tu ] || { echo "== $tu: absent" >> $out; continue; }
  cmd=$(python3 -c "import json,sys;d=json.load(open('$B/compile_commands.json'));m=[e for e in d if e['file'].endswith('/$tu')];print(m[0]['command'] if m else '')")
  [ -n "$cmd" ] || { echo "== $tu: no compile command" >> $out; continue; }
  args=$(python3 -c "import shlex,sys;a=shlex.split(sys.argv[1]);i=a.index('-o');del a[i:i+2];c=a.index('-c');del a[c:c+2];print(shlex.join(a[1:]))" "$cmd")
  echo "== $tu" >> $out
  (cd $B && eval nice $cxx $args -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion -Wno-error=conversion -Wno-error=shadow -O2 -c $R/$w/$tu -o /dev/null) >> $out 2>&1
  echo "rc=$?" >> $out
done
grep -c "warning:" $out; grep -E "^== |^rc=" $out
