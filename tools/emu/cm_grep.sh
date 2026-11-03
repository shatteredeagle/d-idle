#!/bin/bash
# covmissiles: grep a read-only d2d checkout for reading. Usage: cm_grep.sh PATTERN DIR_UNDER_D2D_LAB [GLOB]
d=$HOME/d2d-lab/$2
grep -rnE --include="${3:-*.[ch]pp}" "$1" "$d" | head -${CM_MAX:-80} | cut -c1-260
