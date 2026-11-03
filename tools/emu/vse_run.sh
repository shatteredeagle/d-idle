#!/bin/bash
# verifystateev2 helper (the worker host): run vse_run.py next to the gaps read checkout of upstream's tools/emu (it holds life_apply.py).
#   tools/emu/run-on-worker.sh vse_run.sh [cases] [seed]
here=$(cd "$(dirname "$0")" && pwd)
export UP=${UP:-$HOME/d2d-lab/pr/gaps}
exec bash "$here/run-py.sh" vse_run.py "$@"
