#!/bin/bash
# verifycovmissiles (the worker host): who in d2d (upstream main checkout) mentions the pierce stats.
cd "$HOME/d2d-lab/src/components" || exit 1
for pat in skill_pierce item_pierce 'psum\[.*166' kPierceIndex pierce_idx; do
  echo "== $pat"
  grep -rn -e "$pat" . --include=*.cpp --include=*.hpp | cut -c1-230 | head -8
done
