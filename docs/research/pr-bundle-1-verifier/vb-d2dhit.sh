#!/bin/bash
# vb-d2dhit.sh NAME : d2d side of the hit footprint on build-NAME, six rows, eight classes
B=~/d2d-lab/verifybundle/build-$1; O=~/d2d-lab/verifybundle/out-$1; mkdir -p $O
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
P="nice -n 19 timeout 600 $B/tests/test_vb_probe"
C="190 191 2 5 38 58 104 156"
$P --hit blizzard1 Blizzard 9 $C > $O/hit_159.d2d 2>&1; echo 159 $?
$P --hit firewall "Fire Wall" 6 $C > $O/hit_69.d2d 2>&1; echo 69 $?
$P --hit meteorfire Meteor 6 $C > $O/hit_240.d2d 2>&1; echo 240 $?
$P --hit blaze Blaze 6 $C > $O/hit_67.d2d 2>&1; echo 67 $?
$P --hit "erruption crack 1" Eruption 6 $C > $O/hit_462.d2d 2>&1; echo 462 $?
$P --hit poisonexplosioncloud "Poison Explosion" 6 $C > $O/hit_217.d2d 2>&1; echo 217 $?
$P --burst fireball "Fire Ball" 400 > $O/burst_fb.d2d 2>&1; echo fb $?
$P --burst glacialspike "Glacial Spike" 400 > $O/burst_gs.d2d 2>&1; echo gs $?
