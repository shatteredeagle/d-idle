#!/bin/bash
# esverify: run a tools/emu python script with the gaps read checkout (it holds life_apply.py) on the path.
here=$(cd "$(dirname "$0")" && pwd)
export UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
exec bash "$here/run-py.sh" "$@"
