#!/bin/bash
# werror.sh COMPILER: (inside the namespace) each changed TU with its own compile command + CI flags + -Wshadow -Wrange-loop-construct -Wconversion
cxx=$1; P=/home/jesse/d2d-lab/verifyplayerroll; B=$P/build-base; out=/home/jesse/d2d-lab/verifymonsterpace/werror-$(basename $cxx).log; : > $out
for tu in components/game/ai.cpp tests/monsters/test_monster_ai.cpp tests/monsters/test_monster_thinks.cpp; do
  cmd=$(python3 -c "import json;d=json.load(open(\"$B/compile_commands.json\"));m=[e for e in d if e[\"file\"].endswith(\"/$tu\")];print(m[0][\"command\"] if m else \"\")")
  args=$(python3 -c "import shlex,sys;a=shlex.split(sys.argv[1]);i=a.index(\"-o\");del a[i:i+2];c=a.index(\"-c\");del a[c:c+2];print(shlex.join(a[1:]))" "$cmd")
  echo "== $tu" >> $out
  (cd $B && eval nice $cxx $args -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion -Wno-error=conversion -Wno-error=shadow -Wno-error=sign-conversion -O2 -c $P/base/$tu -o /dev/null) >> $out 2>&1
  echo "rc=$?" >> $out
done
