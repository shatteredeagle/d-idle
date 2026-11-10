#!/bin/bash
# decomp8 (worker host): print a range of an upstream-main source file. usage: d8_src.sh FILE FIRST LAST
sed -n "$2,$3p" $HOME/d2d-lab/wt-decomp8/$1 | cut -c1-200
