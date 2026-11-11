#!/bin/sh
# SORCREST: summarise sk_rest_burst.py output: hits per size and the cases that differ from dist^2 <= R^2.
# usage: sorcrest_burst_sum.sh FILE R
awk -v R="$2" '$6!="nomissile"{n++; w=($4*$4+$5*$5<=R*R)?1:0; if (w!=$6) bad++; h[$2]+=$6; c[$2]++}
END{printf "cases %d, differ from dist2 <= %d^2: %d;", n, R, bad+0; for (s in h) printf " size %s: %d of %d hit", s, h[s], c[s]; print ""}' "$1"
