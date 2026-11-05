#!/bin/bash
# covmonai (worker host): bucket the replay mismatches of cvm_run.sh seed $1 (mai2_resid.py).
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-covmonai}
python3 "$here/mai2_resid.py" "$S/run-${1:?seed}/stderr.txt" "${@:2}"
