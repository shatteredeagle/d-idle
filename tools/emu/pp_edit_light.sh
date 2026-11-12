#!/bin/bash
# prprep (worker host): narrow the rebased Lightning candidate to the rows its differential proves (LastCollide, no Pierce).
cd $HOME/d2d-lab/wt-pp-light || exit 1
python3 - <<'PY'
import re
p = "components/game/fight/missiles.cpp"; s = open(p).read()
s = s.replace("""        } else if (missile_info.last_collide) {
            // Not NextHit but LastCollide (Lightning): FUN_005a8730 passes every unit but the one the missile
            // struck last (FUN_0064a470), and FUN_005adf10 makes it the last (FUN_0064a400, 0x5adfae).""",
"""        } else if (missile_info.last_collide && !missile_info.pierce) {
            // Not NextHit but LastCollide (Lightning): FUN_005a8730 passes every unit but the one the missile
            // struck last (FUN_0064a470), and FUN_005adf10 makes it the last (FUN_0064a400, 0x5adfae).
            // ponytail: a Pierce row (arrows, javelins, Blessed Hammer) keeps the struck list; its charge rule is #225's.""")
s = s.replace("if (!missile_info.next_hit && !missile_info.last_collide) missile.struck->push_back(int(monster_index));",
              "if (!missile_info.next_hit && !(missile_info.last_collide && !missile_info.pierce)) missile.struck->push_back(int(monster_index));")
open(p, "w").write(s)
p = "components/game/fight/tick.cpp"; s = open(p).read()
s = s.replace("(missile.info->next_hit || missile.info->last_collide) && missile.last_hit_ms == now_ms",
              "(missile.info->next_hit || (missile.info->last_collide && !missile.info->pierce)) && missile.last_hit_ms == now_ms")
open(p, "w").write(s)
PY
python3 - <<'PY'
p = "docs/research/re/missiles.md"; s = open(p).read()
s = s.replace("A missile whose row has LastCollide and not NextHit (Lightning, `lightningbolt`)\nis handed",
              "A missile whose row has LastCollide and neither NextHit nor Pierce (Lightning,\n`lightningbolt`) is handed")
s = s.replace("rows as it does for NextHit rows (no `struck` list); tools/emu/light_pierce.py",
              "rows as it does for NextHit rows (no `struck` list). The Pierce rows (arrows,\njavelins, Blessed Hammer) keep their `struck` list: their oracle is not built\n(ToHit, hit and damage functions), and their charges are FUN_005ada80's. tools/emu/light_pierce.py")
open(p, "w").write(s)
PY
git diff --stat
