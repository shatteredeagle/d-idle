#!/bin/bash
# decomp8: sum the result lines of the Act 1 range runs. usage: d8_sum.sh DIR
cd $HOME/d2d-lab/d8/$1 || exit 1
grep -h "match game.exe" l*.txt | wc -l # | awk "{s+=\$1} END{print s}"
echo "non-matching result lines:"; grep -h "game.exe" l*.txt | grep -v -E "^[a-z]+ level [0-9]+: ([0-9]+)/\1 .*match game.exe" | head
echo "populated rooms:"; grep -h "populated rooms" l*.txt | sed -E 's/.*: ([0-9]+)\/([0-9]+) populated.*/\1/' | awk "{s+=\$1} END{print s}"
