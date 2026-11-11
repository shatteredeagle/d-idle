#!/bin/sh
# SORCREST2: summarise a rim run (sr2_run.sh output): who is in, who is out, by direction and by size.
F=${D2_SCRATCH:-/tmp}/rim_$1_$2.txt
echo "cases" $(grep -c '^R' $F)
echo "d2d_in_game_out" $(awk '$1=="R" && $7==1 && $8==0' $F | wc -l)
echo "d2d_out_game_in" $(awk '$1=="R" && $7==0 && $8==1' $F | wc -l)
echo "equal_at_centre (f=g=2 is nearest the missile centre)"
awk '$1=="R" && $5==2 && $6==2 && $7!=$8' $F | wc -l
echo "by size:"
awk '$1=="R" && $7!=$8 { n[$2]++ } END { for (k in n) print "size", k, n[k] }' $F | sort
echo "max d2d reach among hits, in subtile (sqrt of max dx2+dy2 with the fraction), game rim = 4:"
awk '$1=="R" && $7==1 { x = $3 + $5/5 - 0.5; y = $4 + $6/5 - 0.5; d = sqrt(x*x + y*y); if (d > m) m = d } END { print m }' $F
echo "min distance among d2d misses that the game takes in:"
awk '$1=="R" && $7==0 && $8==1 { x = $3 + $5/5 - 0.5; y = $4 + $6/5 - 0.5; d = sqrt(x*x + y*y); if (m == "" || d > m) m = d } END { print "farthest d2d miss that game takes in:", m }' $F
