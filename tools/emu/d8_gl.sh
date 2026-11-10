#!/bin/bash
# decomp8 (worker host): grep the cached whole-binary listing. usage: d8_gl.sh REGEX [MAX]
grep -E "$1" $HOME/d2d-lab/d5/all.txt | head -${2:-40}
