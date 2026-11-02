#!/bin/bash
# verifynmlevel (the worker host): the 14 det scenarios (tools/emu/det_scenarios_lab.py of this worktree) on my own lab
# builds of base and head ($F/vnl-bin/d2d.base, d2d.head), each twice (base2 / head2 rerun the same binary: a hash
# that differs between base and base2 flaps by itself), in ONE hidden d2d slot. Then the compare.
here=$(cd "$(dirname "$0")" && pwd)
emu=$here/../../../tools/emu
S=/tmp/d2agent-verifynmlevel/det
SC=passive,fight,cold,static,cold2,hit_plain,hit_champion,hit_unique,hit_boss,hit_immune,hit_minion,merc,pet,follow
mkdir -p $S
case ${1:-run} in
run)
  D2D_AGENT=verifynmlevel D2D_DATA=$HOME/d2d-lab/data nice -n 10 bash "$here/../../../tools/d2dslot.sh" run -w 1500 -- \
    python3 "$emu/det_scenarios_run.py" run $HOME/d2d-lab/fork/vnl-bin $S $SC base,head,base2,head2 > $S/run.log 2>&1
  echo "run exit $?"; tail -3 $S/run.log ;;
compare)
  python3 "$emu/det_scenarios_run.py" compare $S $SC base head,base2,head2 ;;
esac
