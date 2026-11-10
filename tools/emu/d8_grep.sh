#!/bin/bash
# decomp8: grep upstream's re/ docs on the worker host. usage: d8_grep.sh REGEX
grep -rn -i -E "$1" $HOME/d2d-lab/wt-decomp8/docs/research/re/*.md | cut -c1-330 | head -${N:-30}
