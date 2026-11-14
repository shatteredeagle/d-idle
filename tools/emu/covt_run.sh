#!/bin/bash
# covtreasure2: run one of the covt_*.py oracles on the worker host (same environment as tre_run.sh).
#   tools/emu/run-on-worker.sh --ttl 1500 covt_run.sh covt_rop.py 10000
HERE=$(cd "$(dirname "$0")" && pwd)
D2=${D2:-$HOME/d2d-lab/pr/skmana}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export D2_BUILD=${D2_BUILD:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build}
export TRE_DROPDUMP=${TRE_DROPDUMP:-$HOME/d2d-lab/fork/build-pristine/tools/drop-dump/drop-dump}
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu" || exit 2
script=$1
shift
exec nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" "$HERE/$script" "$@"
