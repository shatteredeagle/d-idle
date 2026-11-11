#!/bin/bash
# covmissiles: which Missiles.txt columns / functions does d2d read? Counts and first matches per pattern in the skmiss checkout
# (upstream main + the SKMISS fix), components/ only.   cm_sweep.sh
d=$HOME/d2d-lab/pr/skmiss/components
for p in CollideFriend collide_friend CollideType ClientCol ClientSend KnockBack SoftHit CanSlow GetHit '"Holy"' '"Town"' NoMultiShot \
         '"Pierce"' pierce collide_kill return_fire hit_func hit_par1 next_hit '"Qty"' SrcTown CanDestroy '"Trans"' LastCollide \
         '"Collision"' '"Skill"' 'ResultFlags' 'HitFlags' 'SpecialSetup' 'MissileSkill' 'Activate'; do
  n=$(grep -rnE --include='*.[ch]pp' -- "$p" "$d" | wc -l)
  echo "== $p: $n matches"
  grep -rnE --include='*.[ch]pp' -- "$p" "$d" | head -${CM_N:-4} | cut -c1-210 | sed "s#$d/##"
done
