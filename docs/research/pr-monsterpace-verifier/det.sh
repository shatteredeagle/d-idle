#!/bin/bash
cd ~/d2d-lab/verifymonsterpace
export D2D_DATA=/tmp/ug-emu D2D_AGENT=verifymonsterpace
S=passive,fight,cold,static,cold2,hit_plain,hit_champion,hit_unique,hit_boss,hit_immune,hit_minion,hit_phys,merc,pet,follow
nice ~/d2d-lab/d2dslot.sh run -w 1500 -- python3 det_scenarios_run.py run bin det $S base,head > det.out 2>&1
echo rc=$? >> det.out
