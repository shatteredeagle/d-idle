"""decomp6: the claim of fid/cold-enchanted-length: a Cold Enchanted unique's chill is stat 12 x 5 + 100 frames long
(FUN_005a1c70), not the 1 frame d2d's missing length leaves.   python3 sk6_patch_coldlen.py TREE"""
import sys
t = sys.argv[1]
p = t + "/components/rules/uniques.hpp"
s = open(p).read()
old = "    int elem_len = 0;                           // ghostly: its cold's length (frames)\n"
new = ("    int elem_len = 0;                           // the cold's length (frames): ghostly 150, Cold Enchanted 100 + 5 a level\n"
       "    int elem_len_per_level = 0;                 // Cold Enchanted: FUN_005a1c70 adds stat 12 x 5 + 100\n")
assert s.count(old) == 1
s = s.replace(old, new)
old = "// FUN_005a1370's \"at most two immunities\""
new = ("// The cold length a boss's blow carries (frames): FUN_005a1c70 adds the unit's level x 5 + 100 (a level above 110 counts as 110:\n"
       "// the 113 units of tools/emu/umod_stats.py get 650), ghostly's FUN_005a1080 150.\n"
       "inline int boss_elem_length(const BossStats& stats, int level) { return stats.elem_len + stats.elem_len_per_level * std::min(level, 110); }\n"
       "// FUN_005a1370's \"at most two immunities\"")
assert s.count(old) == 1
s = s.replace(old, new)
old = "            stats.elem_max_pct = umods.constants[std::size_t(base + 3 + difficulty_index)];\n            resist(id);"
new = ("            stats.elem_max_pct = umods.constants[std::size_t(base + 3 + difficulty_index)];\n"
       "            stats.elem_len = id == umod::cold ? 100 : 0;                  // FUN_005a1c70: the length 100 + level x 5\n"
       "            stats.elem_len_per_level = id == umod::cold ? 5 : 0;\n"
       "            resist(id);")
assert s.count(old) == 1
s = s.replace(old, new)
open(p, "w").write(s)
p = t + "/components/game/ai.cpp"
s = open(p).read()
old = "std::max(damage_pct * boss.elem_max_pct / 100, damage_pct * boss.elem_min_pct / 100), boss.elem_len, put++"
assert s.count(old) == 1
s = s.replace(old, "std::max(damage_pct * boss.elem_max_pct / 100, damage_pct * boss.elem_min_pct / 100), d2d::rules::boss_elem_length(boss, monster.stats.level), put++")
open(p, "w").write(s)
p = t + "/tests/monsters/test_umod_stats.cpp"
s = open(p).read()
old = "if (boss.elem == 2) cold_len = boss.elem_len;"
assert s.count(old) == 1
s = s.replace(old, "if (boss.elem == 2) cold_len = boss_elem_length(boss, level);")
open(p, "w").write(s)
p = t + "/docs/research/re/monsters.md"
s = open(p).read()
old = "    element min / max = MonLvl damage for its level × (elemental rows) %,\n    then FUN_005a1370;\n"
assert s.count(old) == 1
s = s.replace(old, "    element min / max = MonLvl damage for its level × (elemental rows) %,\n    cold enchanted (FUN_005a1c70) also the cold's length: the unit's level\n    × 5 + 100 frames (stat 0x38; ghostly's FUN_005a1080 adds 150),\n    then FUN_005a1370;\n")
open(p, "w").write(s)
