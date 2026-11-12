cd $HOME/d2d-lab/fork/src; git fetch upstream 2>&1|tail -1; git log upstream/main --oneline -1
git grep -n "lvl.size()" upstream/main -- components | head -4
git grep -n "boss_elem_length" upstream/main -- components | head
cd $HOME/d2d-lab/wt-pp-cold4; grep -n "boss_elem_length" components/game/ai.cpp tests/monsters/test_umod_stats.cpp tests/game/test_boss_cold_length.cpp; grep -n "damage_rows" tests/monsters/test_umod_stats.cpp | head -4; grep -n "std::min(monster" tests/game/test_boss_cold_length.cpp
cd ../wt-pp-light4 2>/dev/null; cd $HOME/d2d-lab/wt-pp-light3; grep -n "import os\|os\." tools/emu/light_pierce.py | head; sed -n 63,67p tests/game/test_light_pierce.cpp
