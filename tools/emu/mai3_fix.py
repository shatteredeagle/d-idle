"""covmonai2: patch the generated replay program (mai2_patch.py's output) once more: the leader's rally effect is added only when d2d's port
did not push one itself (the Scarab's port sets the flag and pushes the effect: counted twice by the second pass's patch, a harness
artifact: 32 of 1,500 Scarab cases in this card's first run). usage: python3 mai3_fix.py <scratch dir>"""
import sys

path = sys.argv[1] + "/mai2_replay.cpp"
t = open(path).read()
old = "if (rally) effects.push_back(d2d::rules::ThinkEffect{ d2d::rules::ThinkEffect::rally, 0, 0 });"
new = ("if (rally && std::none_of(effects.begin(), effects.end(), [](const d2d::rules::ThinkEffect& f) { return f.kind == d2d::rules::ThinkEffect::rally; }))"
       " effects.push_back(d2d::rules::ThinkEffect{ d2d::rules::ThinkEffect::rally, 0, 0 });")
assert old in t
t = t.replace(old, new, 1)
if "#include <algorithm>" not in t:
    t = "#include <algorithm>\n" + t
open(path, "w").write(t)
