cd $HOME/d2d-lab/fork/src
git log upstream/main --oneline -i --grep="lightning" | head -8
echo --- seq; git log upstream/main --oneline -i --grep="sequence" | head -5
echo --- ; git show upstream/main:components/game/fight/cast.cpp 2>/dev/null | grep -n -i "seq\|SQ\b" | head -8
git grep -n -i "seqtrans\|seq_trans\|seqnum" upstream/main -- components | head -8
echo --- merc hire; git grep -n "hire_offers" upstream/main -- components apps | head -8
echo --- gold; git grep -n "kGoldPerLevel\|purse_cap\|gold_limit" upstream/main -- components | head -5
echo --- resurrect; git grep -n "resurrect_mode\|ResurrectMode" upstream/main -- components | head -5
echo --- attract; git grep -n -i "attract" upstream/main -- components/game | head -5
