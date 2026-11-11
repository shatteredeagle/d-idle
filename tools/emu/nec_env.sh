#!/bin/sh
# necskill oracle env on the worker host (source it). The upstream emu modules
# (emu.py, skill_tips.py, skill_damage.py) come from the palskill read checkout.
export D2=${D2:-$HOME/d2d-lab/pr/palskill}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export D2_BUILD=${D2_BUILD:-$HOME/d2d-lab/src/build}
HERE=${HERE:-$PWD}
export PYTHONPATH=$HERE:$D2/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
