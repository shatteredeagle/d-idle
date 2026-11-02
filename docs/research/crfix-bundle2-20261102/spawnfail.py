import importlib.util, sys
sys.path.insert(0, sys.argv[1] and "/home/jesse/d2d-lab/src/tools/emu")
import missiles
orig = missiles.Game.unit; n = [0]
def unit(self, cls, *a):
    if cls == 19:
        n[0] += 1
        if n[0] == 2: return None   # the second monster fails to spawn
    return orig(self, cls, *a)
missiles.Game.unit = unit
spec = importlib.util.spec_from_file_location("sgr", sys.argv[1]); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
m.game("fireball", int(sys.argv[2]))
