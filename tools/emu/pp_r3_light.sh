#!/bin/bash
# prprep2 (worker host): Lightning candidate r3: rebase on current upstream main, limit to srv_do == 1 rows, oracle knob removed, counts agree.
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
F=$HOME/d2d-lab/fork/src; T=$HOME/d2d-lab/wt-pp-light3
cd $F; git fetch upstream 2>&1 | tail -1
[ -d $T ] || git worktree add -b fid/missile-last-unit-r3 $T upstream/main 2>&1 | tail -1
cd $T
git log --oneline -1
$G cherry-pick df6d59f2 89059e2d 2>&1 | tail -1
python3 - <<'PY'
p="components/game/gamedata/gamedata.hpp"; s=open(p).read()
a="bool last_collide = false;                   // LastCollide (+4 bit 0): it passes the unit it last struck by\n"
assert a in s
s=s.replace(a,a+"                         // The one-unit-a-frame, skip-the-last-unit rule (FUN_005ae1f0, FUN_005a8730) for a srvdofunc 1 row with LastCollide and neither NextHit\n                         // nor Pierce. srvdofunc 3 rows (standing clouds) and Pierce rows keep the struck list.\n                         bool last_unit_rule() const { return last_collide && !next_hit && !pierce && srv_do == 1; }\n")
open(p,"w").write(s)
p="components/game/fight/missiles.cpp"; s=open(p).read()
s=s.replace("} else if (missile_info.last_collide && !missile_info.pierce) {","} else if (missile_info.last_unit_rule()) {")
s=s.replace("!(missile_info.last_collide && !missile_info.pierce)","!missile_info.last_unit_rule()")
s=s.replace("// ponytail: a Pierce row (arrows, javelins, Blessed Hammer) keeps the struck list; its charge rule is #225's.","// ponytail: a Pierce row (arrows, javelins, Blessed Hammer) and a srvdofunc 3 row (standing clouds) keep the struck list.")
open(p,"w").write(s)
p="components/game/fight/tick.cpp"; s=open(p).read()
s=s.replace("(missile.info->next_hit || (missile.info->last_collide && !missile.info->pierce))","(missile.info->next_hit || missile.info->last_unit_rule())")
open(p,"w").write(s)
p="docs/research/re/missiles.md"; s=open(p).read()
i=s.index("## One unit a frame, and the last unit skipped"); j=s.index("## Pierce — FUN_005ada80")
new="""## One unit a frame, and the last unit skipped — FUN_005ae1f0, FUN_005a8730

A missile of a srvdofunc 1 row (its handler at 0x73c768 entry 1 is FUN_005ae1f0)
with LastCollide and neither NextHit nor Pierce is handed ONE unit a frame:
FUN_005ae1f0 stops at the first unit FUN_00641cb0 finds in the subtiles the
frame entered that FUN_005a8730 passes (every unit but the one the missile
struck last, FUN_0064a470), and FUN_005adf10 makes that unit the last one
(FUN_0064a400, 0x5adfae). So two monsters in one frame are struck on two frames
at most, and a monster the missile left for another may be struck again. The
player skills with such a row are Lightning, Lightning Fury, Inferno, Bone Spear
and Arctic Blast. `Fight::skill_missile_hits` keeps `last_hit` for these rows as
it does for NextHit rows (no `struck` list). The Pierce rows (arrows, javelins,
Blessed Hammer) and the srvdofunc 3 rows (Blizzard shards, Poison Explosion and
the javelin clouds, which stand at velocity 0) keep their `struck` list: the
first need ToHit and the hit and damage functions in the oracle, the second take
FUN_005ae480's standing path (FUN_0064cb90), not run. tools/emu/light_pierce.py
and test_light_pierce check 1,192 casts a seed. Which unit of a frame goes first
is d2d's monster order here; game.exe walks the subtiles in path order (not read).

"""
s=s[:i]+new+s[j:]
open(p,"w").write(s)
PY
$G add -A components docs; $G commit -q --fixup=HEAD
sed -i 's/for index in (touched\[::-1\] if os.environ.get("SK6_REVERSE") else touched):   # decomp6: the unit order is d2d.s monster order, or its reverse/for index in touched:   # the unit order is d2d'"'"'s monster order/' tools/emu/light_pierce.py
grep -n "for index in" tools/emu/light_pierce.py
$G add tools/emu/light_pierce.py; $G commit -q --fixup=HEAD~2
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --msg-filter 'grep -v "^(cherry picked from" | sed "s/through FUN_005a8850 and FUN_005adf10/through the filter at 0x5a8850 and FUN_005adf10/; s/A LastCollide row with neither NextHit nor Pierce struck/A srvdofunc 1 row with LastCollide and neither NextHit nor Pierce (Lightning, Lightning Fury, Inferno, Bone Spear, Arctic Blast) struck/"' --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -3 | cut -c1-110; git log -2 --format=%B | head -12
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
git status --short | head
