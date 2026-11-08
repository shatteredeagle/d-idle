import re,os
p=os.path.expanduser("~/d2d-lab/pr/skilldmg-r1/tools/emu/skill_damage.py")
s=open(p).read()
s=s.replace("e.hook(0x6442a0, lambda e: self.levels.get((e.arg(1) - FAKE) // 0x40, 0), 3)",
"e.hook(0x6442a0, lambda e: self.lv((e.arg(1) - FAKE) // 0x40, e.arg(2)), 3)")
s=s.replace("    def row(self, skill):","""    def lv(self, i, flag):
        base = self.levels.get(i, 0)
        if os.environ.get("SD_ITEMS") and flag and base > 0:
            return base + 1 + i % 3
        return base

    def row(self, skill):""",1)
s=s.replace("import random\n","import os\nimport random\n",1)
s=s.replace("        lo = self.e.call(0x644d50, self.unit, skill, lvl, 1)","        if os.environ.get('SD_ITEMS') and lvl > 0: lvl += 1 + skill % 3\n        lo = self.e.call(0x644d50, self.unit, skill, lvl, 1)")
open(p,"w").write(s)
p=os.path.expanduser("~/d2d-lab/pr/skilldmg-r1/tests/game/test_game.cpp")
s=open(p).read()
old="            env.base_level = env.level = [&](int other) { const auto found = levels.find(other); return found == levels.end() ? 0 : found->second; };"
assert old in s
new="""            const bool items = std::getenv("SD_ITEMS") != nullptr;
            env.base_level = [&](int other) { const auto found = levels.find(other); return found == levels.end() ? 0 : found->second; };
            env.level = [&](int other) { const int base = env.base_level(other); return items && base > 0 ? base + 1 + other % 3 : base; };
            if (items && lvl > 0) lvl += 1 + skill_id % 3;"""
s=s.replace(old,new)
open(p,"w").write(s)
