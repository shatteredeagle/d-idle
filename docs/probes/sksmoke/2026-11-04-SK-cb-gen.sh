#!/bin/bash
# prints the probe string: LEVEL CASTS [FRAMES [PRESTEPS]] (PRESTEPS: idle frames before the first cast, so a repeat run is a different sample on the deterministic lockstep) (one frame per step, monsters + debug missiles each frame)
L=$1; N=$2
P="debug goto 2;debug stat 1 800;debug stat 6 256000;debug stat 7 256000;debug stat 9 400000;debug points 38 $L;debug mana full;debug spawn zombie1 1 near;debug mon set 17 life 1000000 maxlife 1000000 dmg 0 0 res fire 0 res light 0 res cold 0 res phys 0 res magic 0;monsters${4:+;step $4}"
for c in $(seq 1 $N); do
  P="$P;debug cast 38 unit 17"
  for i in $(seq 1 ${3:-75}); do P="$P;step 1;monsters;debug missiles"; done
done
echo "$P"
