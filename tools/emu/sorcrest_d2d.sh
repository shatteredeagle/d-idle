#!/bin/sh
# SORCREST: run the d2d side (test_sk_rest) on the worker host. usage: sorcrest_d2d.sh --burst-set|--burst-roll ROW SKILL LEVEL|TRIALS
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
exec nice $HOME/d2d-lab/pr/sorcrest-out/build/tests/test_sk_rest "$@"
