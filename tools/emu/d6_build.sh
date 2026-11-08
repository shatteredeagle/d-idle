#!/bin/bash
# decomp6 (worker host): build the upstream-main test binaries the four oracles need
B=$HOME/d2d-lab/build-decomp6
nice -n 19 ninja -C $B -j3 test_missile_hit test_next_hit test_walk_velocity test_boss_level test_freeze_rules test_freeze_resume test_cold_length test_static_field test_combat test_game test_monsters mpq-cat > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED|unknown target" $B.build.log | head; tail -1 $B.build.log
