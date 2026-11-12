#!/bin/bash
# prprep (worker host): clang-tidy findings of the Cold Enchanted tests, fixed in the commits that own them.
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
cd $HOME/d2d-lab/wt-pp-cold || exit 1
sed -i '/^#include <cstdint>$/d; s/\[&\](const auto& el) { return el.type == type; }/[\&](const auto\& slot) { return slot.type == type; }/' tests/game/test_boss_cold_length.cpp
$G add tests/game/test_boss_cold_length.cpp; $G commit -q --fixup=HEAD
sed -i 's/const int lo = damage_pct \* boss.elem_min_pct \/ 100, hi = std::max(damage_pct \* boss.elem_max_pct \/ 100, lo);/const int low = damage_pct * boss.elem_min_pct \/ 100, high = std::max(damage_pct * boss.elem_max_pct \/ 100, low);/; s/element_rows\[std::size_t(slot)\] = lo; element_rows\[std::size_t(slot) + 1\] = hi;/element_rows[std::size_t(slot)] = low; element_rows[std::size_t(slot) + 1] = high;/' tests/monsters/test_umod_stats.cpp
sed -i 's/^#include "uniques.hpp"$/#include "montypes.hpp"\n#include "uniques.hpp"/' tests/monsters/test_umod_stats.cpp
grep -n "low\|high\|montypes" tests/monsters/test_umod_stats.cpp | head
$G add tests/monsters/test_umod_stats.cpp; $G commit -q --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -3 | cut -c1-100
git status --short | head -3
