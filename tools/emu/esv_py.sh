#!/bin/bash
# esverify: run a tools/emu python script with the gaps read checkout (it holds life_apply.py) on the path.
here=$(cd "$(dirname "$0")" && pwd)
export UP=${UP:-$HOME/d2d-lab/pr/gaps}
exec bash "$here/run-py.sh" "$@"
