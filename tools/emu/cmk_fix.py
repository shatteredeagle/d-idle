"""covmonskills2: label fixes applied to monsterai2's generated replay program (mai2_replay.cpp) before it is compiled. Nothing upstream
is edited; the scratch file is rewritten in place. Each fix is a representation difference, stated in docs/research/covmonskills2-trace.md.

  rally   mai2_patch.py appends a `rally` effect whenever the think set the rally flag (the Fallen's port sets the flag and emits no effect).
          The Scarab / Mummy style ports emit the effect themselves, so the replay would see it twice: append it only when absent.

usage: python3 cmk_fix.py <scratch dir>"""
import sys

path = sys.argv[1] + "/mai2_replay.cpp"
t = open(path).read()
old = "if (rally) effects.push_back("
new = ("if (rally && std::none_of(effects.begin(), effects.end(), [](const d2d::rules::ThinkEffect& f) { return f.kind == d2d::rules::ThinkEffect::rally; })) effects.push_back(")
assert old in t, "mai2_replay.cpp has no rally push (mai2_patch.py changed?)"
t = t.replace(old, new, 1)
if "#include <algorithm>" not in t:
    t = "#include <algorithm>\n" + t
open(path, "w").write(t)
print("cmk_fix: rally effect de-duplicated")
