#!/bin/bash
# stateevrun helper (the worker host): run ser_events.py next to the gaps read checkout of upstream's tools/emu (it holds life_apply.py).
#   tools/emu/run-on-worker.sh ser_run.sh [random_cases] [seed]
here=$(cd "$(dirname "$0")" && pwd)
export UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
exec bash "$here/run-py.sh" ser_events.py "$@"
