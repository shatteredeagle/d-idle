"""decomp6: the claim of fid/missile-last-unit on the scratch tree: a LastCollide missile (Lightning) hands ONE unit a frame
to the hit and skips the unit it struck last (FUN_005ae1f0, FUN_005a8730 / FUN_0064a470 / FUN_0064a400), instead of
striking every touching monster once (the `struck` list).   python3 sk6_patch_lastcollide.py TREE"""
import sys
t = sys.argv[1]
p = t + "/components/game/fight/missiles.cpp"
s = open(p).read()
old = """            missile.last_hit = int(monster_index);
            missile.last_hit_ms = now_ms;
        }
"""
new = """            missile.last_hit = int(monster_index);
            missile.last_hit_ms = now_ms;
        } else if (missile_info.last_collide) {
            // Not NextHit but LastCollide (Lightning): FUN_005a8730 passes every unit but the one the missile
            // struck last (FUN_0064a470), and FUN_005adf10 makes it the last (FUN_0064a400, 0x5adfae).
            if (missile.last_hit == int(monster_index)) return false;
            missile.last_hit = int(monster_index);
            missile.last_hit_ms = now_ms;
        }
"""
assert s.count(old) == 1
s = s.replace(old, new)
old = "if (!missile_info.next_hit) missile.struck->push_back(int(monster_index));"
assert s.count(old) == 1
s = s.replace(old, "if (!missile_info.next_hit && !missile_info.last_collide) missile.struck->push_back(int(monster_index));")
open(p, "w").write(s)
p = t + "/components/game/fight/tick.cpp"
s = open(p).read()
old = "if (missile.info && missile.info->next_hit && missile.last_hit_ms == now_ms) return false;"
assert s.count(old) == 1
s = s.replace(old, "if (missile.info && (missile.info->next_hit || missile.info->last_collide) && missile.last_hit_ms == now_ms) return false;")
open(p, "w").write(s)
