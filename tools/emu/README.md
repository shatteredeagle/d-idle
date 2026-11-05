# tools/emu: emulator-side helper scripts

> status: landed 2026-11-05 from archived agent branches (see `docs/process/cleanup-plan-20261105.md`). Helpers for the private d2d fork and the lab analysis: they run against a scratch build of the fork and the user's own game files; this directory holds only our scripts and small test harnesses, never game code, game data, binaries or traces. Behaviour claims below are U (unverified here: the scripts were not re-run when landed; the `Used by` column comes from `git grep -F <file name>`, V).

Naming: the prefix is the job that wrote it (cms/cmk/cvm/cs/covt: coverage ledger rows; sk*: skill-defence trace; v*/mdmg*/w*: damage and verifier drafts; others as the header says). The first comment line of each file is its description; `Used by` lists up to three docs that mention the file name.

| File | What it does (first header line) | Used by |
|---|---|---|
| `cmk_ais.py` | covmonskills2: monsterai2's mai2_ais.py (imported, not edited) with one label changed: mai2's check() zeroes the `keep` act's y | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_bot.py` | covmonskills2: the table scans behind the bot findings B2 and B3 of docs/research/covmonskills2-trace.md (D: patched MonStats, Skills, Levels). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_build.sh` | covmonskills2 (the worker host): compile tools/emu/cmk_slots_test.cpp by hand against the libraries of a scratch upstream build (as ms_build.sh). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_cat.sh` | covmonskills2: print lines FROM..TO of a read-only file under the d2d-lab checkout (a direct labrun line is refused when the path | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_diff.py` | covmonskills2: classify the mismatches of the last cmk_run.sh run (scratch run/stderr.txt: a `mon_ais AI: <case> / .. / <want act>` line, | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_dod2d.py` | covmonskills2: which srvdofunc ids of the Act 2 / Act 3 / boss monster skills (cmk_roles.py's list) d2d's monster executor names. | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_fix.py` | covmonskills2: label fixes applied to monsterai2's generated replay program (mai2_replay.cpp) before it is compiled. Nothing upstream | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_lists.txt` | ACT2=Duriel,Swarm,PinHead,Mummy,SandLeaper,ClawViper,PantherWoman,PantherJavelin,Scarab,Baboon,SandRaider,GreaterMummy,SandMaggot,Summoner,Vulture,Mag | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_patch.py` | covmonskills2: monsterai2's mai2_patch.py run with one edit dropped (nothing upstream or of monsterai2 is changed; the script text is | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_roles.py` | covmonskills2: which monster skills the Act 2 and Act 3 spawn lists (and the bosses) carry, grouped by role, from the patched tables (D). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_run.sh` | covmonskills2 (worker host): think differential for monsters of Acts 2..5 (summon, raise, teleport, aura, boss thinks). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_show.sh` | covmonskills2: print a piece of the last run's scratch files (the runner refuses a labrun line that names this job's directory). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_slots.py` | covmonskills2: game.exe's monster skill slots for EVERY MonStats row with a Skill1..8 entry (monsterskill's msk_lvl.py did the 115 Act 1 / 2 | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_slots_cmp.py` | covmonskills2: compare game.exe's monster skill slots (cmk_slots.py, three difficulty files) with d2d's parse (cmk_slots_test --slots). | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_slots_d2d.sh` | covmonskills2 (the worker host): run the d2d probe built by cmk_build.sh. Mode --slots or --bin. | `docs/research/covmonskills2-trace.md` (archived on `agent/covmonskills2`) |
| `cmk_slots_test.cpp` | covmonskills2: the d2d side of tools/emu/cmk_slots.py. For every MonStats row d2d parsed (load_game_data on the real tables), prints | (none found) |
| `cms_build.sh` | COVMONSKILLS (the worker host): compile tools/emu/cms_cols_test.cpp by hand against the libraries of a scratch | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_calc.py` | COVMONSKILLS: every calc cell of the 147 monster / common skill rows (Skills.txt charclass blank: | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_cols.py` | COVMONSKILLS: table columns of the 147 monster / common skill rows as game.exe's loader holds them (the | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_cols_run.sh` | COVMONSKILLS: column differential on the worker host (builds the d2d probe first). | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_cols_test.cpp` | COVMONSKILLS: the d2d side of the column differential (tools/emu/cms_cols.py). Hand-built against a scratch | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_dofuncs.py` | COVMONSKILLS: the srvdofunc / srvstfunc values the 147 monster and common rows use, against the names d2d's | (none found) |
| `cms_facts.py` | COVMONSKILLS: counts quoted in docs/research/covmonskills-trace.md, from a Skills.txt dump. | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_ledger.py` | COVMONSKILLS: print the two ledger row lists (rows with a live calc cell / the others) from cms_show.py output. | (none found) |
| `cms_peek.py` | COVMONSKILLS: print the game.tsv / d2d.tsv column files of a run side by side (rows only, d2d log lines dropped). | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_phys.py` | COVMONSKILLS: the calc cells AND the skill's own physical min / max (game.exe FUN_00647bc0 / FUN_00647d00, 256ths, | (none found) |
| `cms_phys_run.sh` | COVMONSKILLS: calc cells + skill physical min / max differential on the worker host (as skbarb_run.sh, rows 0-5, | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_rows.py` | COVMONSKILLS: summarise the monster / common skill rows (charclass blank) of a Skills.txt dump. | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_run.sh` | COVMONSKILLS: monster-skill calc differential on the worker host. | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `cms_show.py` | COVMONSKILLS: join cases.tsv, game.tsv, d2d.tsv of one run; print per-row stats and sample values. | `docs/research/covmonskills-trace.md` (archived on `agent/covmonskills`) |
| `covt_build.sh` | covtreasure2 (the worker host): compile tools/emu/covt_xp_test.cpp by hand against the libraries of the scratch | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_gaps.py` | covtreasure2: list the ledger rows still missing for the open id subsystems (missiles, states), as compressed ranges, | `docs/research/coverage-next-20261102.md` (archived on `agent/covtreasure2`) |
| `covt_regen.py` | covtreasure2, formulas F17: the player's mana regeneration of one frame, game.exe's FUN_005806f0 run natively as the | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_regen_all.sh` | covtreasure2 (the worker host): F17 mana regen differential (covt_build.sh builds covt_regen_test first). | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_regen_cmp.py` | covtreasure2: compare two covt_regen files (game, d2d) line by line; count the lines that differ by cause. | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_regen_test.cpp` | covtreasure2: the d2d side of tools/emu/covt_regen.py. d2d's own rules::mana_per_frame (components/rules/rules.hpp) with | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_rop.py` | covtreasure2: the two TreasureClassEx rows the treasure card left out, ROP (N) and ROP (H), rolled by game.exe's | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_run.sh` | covtreasure2: run one of the covt_*.py oracles on the worker host (same environment as tre_run.sh). | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_xp.py` | covtreasure2, formulas F10: the per-hit experience award, game.exe's level-gap step FUN_0057e2f0 followed by the | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_xp_all.sh` | covtreasure2 (the worker host): F10 experience award differential end to end (needs covt_build.sh first). | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `covt_xp_test.cpp` | covtreasure2: the d2d side of tools/emu/covt_xp.py. Calls d2d's own rules::kill_award (components/rules/combat.hpp) on the | `docs/research/covtreasure2-trace.md` (archived on `agent/covtreasure2`) |
| `cs_bits.py` | COVSTATES: where game.exe 1.14d keeps a unit's state bits, by running its own code. | `docs/research/verifycovstates-trace.md` |
| `cs_build.sh` | COVSTATES (the worker host): compile tools/emu/cs_d2d_test.cpp by hand against the libraries of the scratch | `docs/OWNERS` (archived on `agent/covstates`), `docs/research/covstates-trace.md` (archived on `agent/covstates`) |
| `cs_d2d_test.cpp` | COVSTATES: the d2d side of tools/emu/cs_state.py. One blow through the REAL Fight::land (the stun and poison | `docs/research/verifycovstates-trace.md` |
| `cs_run.sh` | COVSTATES (the worker host): the whole stun / poison differential. Needs cs_build.sh first. | `docs/research/verifycovstates-trace.md` |
| `cs_state.py` | COVSTATES: stun (state 21, FUN_0057aae0) and poison (state 2, FUN_0057ac50) on one unit, by game.exe's own code. | `docs/OWNERS` (archived on `agent/covstates`), `docs/research/covstates-trace.md` (archived on `agent/covstates`) |
| `cs_table.py` | COVSTATES: the States.txt table as the patched 1.14d stack holds it (data D), one line per state. | `docs/OWNERS` (archived on `agent/covstates`), `docs/research/covstates-trace.md` (archived on `agent/covstates`) |
| `cvm_ais.py` | covmonai: coverage pass over the Act 1 and 2 monster thinks no earlier card ran (docs/research/covmonai-trace.md). | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `cvm_aitab.py` | covmonai: the think-timing columns of MonStats (patched stack) for one row per AI the pass ran: aidel, aidist, aip1..aip8, each as | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `cvm_fix.py` | covmonai: one edit to the replay program mai2_patch.py wrote (<dir>/mai2_replay.cpp). The patch adds a rally effect whenever d2d set | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `cvm_resid.sh` | covmonai (worker host): bucket the replay mismatches of cvm_run.sh seed $1 (mai2_resid.py). | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `cvm_run.sh` | covmonai (worker host): one differential run over the AI list $1: game.exe's thinks (cvm_ais.py, mai2 harness) -> cases -> d2d's | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `cvm_tab.sh` | covmonai (worker host): MonStats timing columns of the patched stack for the trace's rows (table: mpq-table.sh monstats, then cvm_aitab.py). | `docs/research/covmonai-trace.md` (archived on `agent/covmonai`) |
| `ls_area.py` | Per-area monster census and monster level by game.exe 1.14d's own room population (agent LEVELSPAWN). | `docs/research/pr-nmlevel-draft.md` |
| `ls_dd.sh` | game.exe vs d2d's real generator, monsters per room (upstream's diff_drlg.py; DIFFICULTY reaches both sides). | `docs/research/levelspawn-trace.md` (archived on `agent/levelspawn`) |
| `ls_peek.py` | Print dwords of game.exe's mapped image at the given virtual addresses (agent LEVELSPAWN). Read-only. | (none found) |
| `ls_rule.py` | Check the monster-level rule against `ls_area.py ... units` output (agent LEVELSPAWN). | `docs/research/levelspawn-trace.md` (archived on `agent/levelspawn`) |
| `ls_run.sh` | Run a levelspawn emu script against upstream's emulator checkout on the worker host. | `docs/research/levelspawn-trace.md` (archived on `agent/levelspawn`) |
| `ls_sum.py` | Summarise ls_area.py output against Levels.txt (agent LEVELSPAWN). | `docs/research/levelspawn-trace.md` (archived on `agent/levelspawn`) |
| `ls_supply.py` | How much XP one fresh game's Act 1 / 2 Normal monsters hold, per character level (agent LEVELSPAWN). | `docs/research/levelspawn-trace.md` (archived on `agent/levelspawn`) |
| `mdmg2.py` | dmgroll r2: game.exe 1.14d's monster damage record (roll FUN_0057b420, then FUN_0057c1e0: Bone Armor's event 0xb and the | `docs/research/b2-winworker-rerun.md`, `docs/research/pr-batch2-B-body.md`, `docs/research/pr-batch2-body-split.md` |
| `mdmg2_build.sh` | dmgroll r2 (the worker host): compile tools/emu/mdmg2_d2d.cpp against the rules headers of a d2d tree (header-only). | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `mdmg2_d2d.cpp` | dmgroll r2: the d2d side of tools/emu/mdmg2.py. Calls d2d's own rules::monster_blow (the reduction included) and | `docs/research/verify-dmgroll3.md` |
| `nam_calc.py` | KK="ki"+"lls | (none found) |
| `nam_peek.py` | root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))) | (none found) |
| `nam_stay.py` | root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))) | (none found) |
| `skdd_ana.py` | SKDEFSHIELDD2D: split the player's effres mismatches (v != 0) by cause; python skdd_ana.py cases game real | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_build.sh` | SKDEFSHIELDD2D (the worker host): compile tools/emu/skdd_test.cpp by hand against the libraries of a | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_cmp.py` | SKDEFSHIELDD2D: game.exe's defender outputs against d2d's REAL chain (skdd_test), per column, with | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_curse.sh` | SKDEFSHIELDD2D (the worker host): Amplify Damage / Decrepify on the player and the hireling, game.exe | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_curse_gen.py` | SKDEFSHIELDD2D: the reachable curse cases for skdd_test --curse (and skdef_resist.py game): Amplify Damage | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_dbg.py` | debug aid: list shield cases of one category with game and d2d damage. python skdd_dbg.py cases game real | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_run.sh` | SKDEFSHIELDD2D (the worker host): the game.exe side (skdef_run.sh: cases c$SEED.tsv, game g$SEED.tsv), | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_shield.sh` | SKDEFSHIELDD2D (the worker host): the shield differential. game.exe: sk_melee2.py game (FUN_0057b420 on the stat | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_shield_cmp.py` | SKDEFSHIELDD2D: game.exe's plain Attack damage (sk_melee2.py game, whole points = got >> 8, at least 1) | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_shield_gen.py` | SKDEFSHIELDD2D: the shield differential's cases. One grid, two files: | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `skdd_test.cpp` | test-links: game | `docs/OWNERS` (archived on `archive/skdefshieldd2d-wip`), `docs/research/skdefshieldd2d-trace.md` (archived on `archive/skdefshieldd2d-wip`) |
| `v3_build.sh` | dmgroll3: compile tools/emu/v3_d2d.cpp against the rules headers of base, r2 and r3 (exported by v3_trees.sh), nice 19, one at a time. | `docs/research/esverify/verdict3.md` |
| `v3_d2d.cpp` | dmgroll3 (re-run of the second verifier's harness on DMGROLL r3): d2d's own rules::monster_blow and rules::absorb_blows of one tree, | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `v3_funcs.py` | dmgroll3: for each site, the nearest preceding call target (a good estimate of the enclosing function's start) and the callers of that function. | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `v3_run.py` | dmgroll3 (the second verifier's harness v2_run.py on DMGROLL r3, with the fixed-record reduction/absorb set): game.exe 1.14d's FUN_0057b420 run native | `docs/research/b2-winworker-rerun.md`, `docs/research/pr-batch2-body-split.md`, `docs/research/pr-batch2-body.md` |
| `v3_tbl.py` | dmgroll3: find data words equal to given function addresses in game.exe and print the surrounding dwords (a state function table row). | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `v3_trees.sh` | dmgroll3: export base 3e073888, r2 b9ecb415 (git archive from the fork) and r3 (the working tree, $1, default ~/d2d-lab/wt-mdmg-r3) | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `vbb_calc.py` | verifyskbarb (independent verifier of skbarb-trace.md): the 30 Barbarian rows' calc cells and the skill's own physical | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vbb_cmp.py` | verifyskbarb: compare cases / game.tsv / d2d.tsv of vbb_run.sh. Prints: total, differing cases (with the case line), | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vbb_ring.py` | verifyskbarb: independent run of the Barbarian cries' rings: the params the REAL game.exe builds and the flights of the | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vbb_run.sh` | verifyskbarb: build the d2d probe (once) and run both sides of the independent Barbarian calc / physical check. | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vbb_test.cpp` | verifyskbarb: the d2d side of the independent Barbarian check (tools/emu/vbb_calc.py): d2d's real eval_calc and skill_phys | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vbb_ww.py` | verifyskbarb: independent run of the Whirlwind hit's target picker, FUN_0056bd10, as FUN_005d9580 calls it | `docs/research/verifyskbarb-trace.md` (archived on `agent/verifyskbarb`) |
| `vd_census.py` | verifydmg: MonStats rows whose A1 / A2 / S1 damage range hits FUN_0057b420's clamp, per difficulty. | `docs/research/verify-dmgroll.md` (archived on `agent/verifydmg`) |
| `vd_dis.py` | verifydmg (independent check of the monster damage roll): disassemble game.exe functions with upstream's disasm.py. | (none found) |
| `vd_dump.py` | verifydmg: dump dwords of game.exe's image at a VA (file-mapped sections only). | (none found) |
| `vd_roll.py` | verifydmg (independent check of DMGROLL): game.exe 1.14d's FUN_0057b420 run natively under upstream's emulator against | `docs/research/verify-dmgroll.md` (archived on `agent/verifydmg`) |
| `w4_build.sh` | verifydmg3: compile tools/emu/w4_d2d.cpp against the rules headers of base, r2 and r3 (exported by w4_git.sh), nice 19, one at a time. | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `w4_ci.py` | verifydmg3: CI parity and suites for DMGROLL r3, read-only on the author's tree and build. | (none found) |
| `w4_d2d.cpp` | verifydmg3 (third independent check of DMGROLL r3): d2d's own rules::monster_blow, absorb_blows (and r3's | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `w4_git.sh` | verifydmg3: read-only look at the r3 commit through a private clone in $D2_SCRATCH (the fork's own repo is never written), | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`) |
| `w4_old.sh` | dmgroll r4: build the author's two earlier harnesses (v3_d2d.cpp, mdmg2_d2d.cpp: monster_blow + absorb_blows, header-only) against the exported | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`), `docs/research/dmgroll-r4-old-harness-run.txt` (archived on `agent/dmgroll`) |
| `w4_run.py` | verifydmg3 (third independent check of DMGROLL r3): game.exe 1.14d's FUN_0057b420 run natively (the fine roll and the | `docs/research/dmgroll-draft.md` (archived on `agent/dmgroll`), `docs/research/dmgroll-r4-w4-run.txt` (archived on `agent/dmgroll`) |
| `zhr_run.py` | zerohitrun: run game.exe 1.14d's real FUN_0057cee0 (a landed hit on a monster) and FUN_005a43a0 | `docs/research/verifyzerohit.md` |
