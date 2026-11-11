#!/bin/sh
# Environment for the skab_* oracles on the worker host (source it).
# D2 = an upstream d2decomped checkout used for emu.py / life_apply.py / disasm.py.
export D2=${D2:-$HOME/d2d-lab/pr/skresistab-d2d}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_BUILD=${D2_BUILD:-$HOME/d2d-lab/src/build}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$D2/tools/emu:${PYTHONPATH:-}
PY=/tmp/casttrace-venv/bin/python
