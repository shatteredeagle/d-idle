#!/bin/bash
# decomp8: build the named targets in build-decomp8 (nice 19). usage: d8_build.sh target...
export PATH=$HOME/.local/bin:$PATH
nice -n 19 cmake --build $HOME/d2d-lab/build-decomp8 -j 4 --target "$@" 2>&1 | tail -5
