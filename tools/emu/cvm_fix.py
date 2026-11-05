"""covmonai: one edit to the replay program mai2_patch.py wrote (<dir>/mai2_replay.cpp). The patch adds a rally effect whenever d2d set
`*input.rally` (the Fallen and the Fallen Shaman signal a rally that way and push no effect). d2d's Scarab pushes the effect itself
(`effect(ThinkEffect::rally)` next to `*input.rally = true`, monsters.hpp), so the patch would count it twice: skip it for the AIs named
here (default Scarab). usage: python3 cvm_fix.py <dir> [AI ...]"""
import sys

path = sys.argv[1] + "/mai2_replay.cpp"
ais = sys.argv[2:] or ["Scarab"]
t = open(path).read()
old = "if (rally) effects.push_back("
assert t.count(old) == 1, "the rally line was not found exactly once"
cond = " && ".join('ai_name != "%s"' % a for a in ais)
t = t.replace(old, "if (rally && %s) effects.push_back(" % cond)
open(path, "w").write(t)
print("cvm_fix: rally effect not added for", ", ".join(ais))
