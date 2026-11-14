#!/bin/bash
# stateev_diff helper (the worker host): run stateev_diff_run.py next to the gaps read checkout of upstream's tools/emu (life_apply.py).
#   tools/emu/run-on-worker.sh --ttl 2400 stateev_diff.sh [N_per_seed] [seed ...]
here=$(cd "$(dirname "$0")" && pwd)
export UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
exec bash "$here/run-py.sh" stateev_diff_run.py "$@"
