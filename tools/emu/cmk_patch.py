"""covmonskills2: monsterai2's mai2_patch.py run with one edit dropped (nothing upstream or of monsterai2 is changed; the script text is
read, edited in memory and executed): the rebuild of the target offset `input.off_x / off_y` from the spots. That rebuild is right for
the Act 1 AIs mai2 was written for (their case line carries no offset); for the AIs whose recorded scene fixes its own offset it can
disagree with the scene, so `plain` replays them as upstream's own replay block does (offset left as its default).
usage: python3 cmk_patch.py <d2d source tree> <scratch dir> [plain]"""
import os
import sys

here = os.path.dirname(os.path.abspath(__file__))
src = open(os.path.join(here, "mai2_patch.py")).read()
if len(sys.argv) > 3 and sys.argv[3] == "plain":
    old = "input.off_x = input.target_x - input.self_x; input.off_y = input.target_y - input.self_y;"
    assert old in src, "mai2_patch.py no longer has the off_x rebuild"
    src = src.replace(old, "")
sys.argv = sys.argv[:3]
exec(compile(src, "mai2_patch.py", "exec"))
