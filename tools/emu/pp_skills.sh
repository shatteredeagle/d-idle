#!/bin/bash
cd $(dirname "$0")/../..
bash tools/emu/mpq-table.sh missiles > /tmp/pp_missiles.txt; bash tools/emu/mpq-table.sh skills > /tmp/pp_skills.txt
python3 tools/emu/pp_skills.py /tmp/pp_missiles.txt /tmp/pp_skills.txt
