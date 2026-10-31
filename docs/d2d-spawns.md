# d2d monster spawns per area, 20 seeds (agent PROBES-D2D, 2026-10-30)

Tags: V-d2d = measured on the private lockstep fork (`d2d.lab5`, seeds 1 to 20, fresh Sorceress, `--no-save`, Normal); U = not checked on Game.exe. The original game is the acceptance target: these are the d2d columns to compare against.
Probe: `cmd/d2dspawn spawns` through `docs/probes/run-d2d.sh` (one d2d per seed, areas in order 2,3,4,5,8 in the same game), table by `docs/probes/spawns_table.py`. Raw data: `docs/probes/d2d-spawns.csv`.

## Method

`debug goto <area>` (the Den: Blood Moor, then `debug enter` of the warp to level 8, because `goto 8` renders black), 60 ticks, then a warp to a 30-cell grid over the whole level (cells from `debug level` W x H), 12 ticks at each point, `monsters` read after each; units are de-duplicated by unit id. Levels keep their rooms loaded (V-d2d: the list only grows, ids are stable, `debug level` stayed on the same area at every grid point), so the count is the whole level, not what is near the player. Dead monsters are not counted.

## Columns (the original-game side fills the same ones)

`area,seed,normal,minion,champion,unique,superunique,total,names,boss_names`: area = Levels.txt id; `normal` = monsters with no boss flag; `minion` = pack followers of a champion/unique; `champion`; `unique` = d2d's "unique" boss column, i.e. what players call rare monsters; `superunique` (Bishibosh, Rakanishu, Treehead WoodFist, Corpsefire); `names` = "Name xN" for normals and minions; `boss_names` = "kind:Name" for champion/unique/superunique. On Game.exe fill the same columns from the monster list (U: how to read champion versus minion there is the original-side card's job).

## Result (V-d2d, 20 seeds each)

area,seed,normal,minion,champion,unique,superunique,total,names,boss_names
2,1,54,0,0,0,0,54,"Fallen x26;Quill Rat x9;Zombie x19",""
3,1,129,5,0,1,1,136,"Dark Hunter x3;Dark Spearwoman x13;Fallen Shaman x22;Fallen x94;Gargantuan Beast x2","superunique:Bishibosh;unique:Snot Wing the Sharp"
4,1,66,10,0,2,0,78,"Carver x15;Dark Ranger x9;Fallen Shaman x3;Fallen x10;Hungry Dead x35;Moon Clan x4","unique:Star Kill the Unholy;unique:Steel Dancer"
5,1,111,9,0,2,0,122,"Carver Shaman x2;Carver x83;Spike Fiend x28;Vile Hunter x3;Vile Lancer x4","unique:Death Fist;unique:Spine Eye"
8,1,24,5,0,0,1,30,"Fallen Shaman x2;Fallen x9;Gargantuan Beast x5;Zombie x13","superunique:Corpsefire"
2,2,81,0,0,0,0,81,"Fallen x53;Quill Rat x12;Zombie x16",""
3,2,73,3,0,1,0,77,"Dark Hunter x42;Dark Spearwoman x22;Gargantuan Beast x12","unique:Haze Grumble the Hungry"
4,2,128,16,0,2,1,147,"Carver x33;Dark Ranger x12;Fallen Shaman x6;Fallen x21;Foul Crow Nest x6;Foul Crow x49;Hungry Dead x5;Moon Clan x12","superunique:Rakanishu;unique:Dread Wight;unique:Gore Growler"
5,2,99,6,0,1,1,107,"Brute x2;Carver Shaman x3;Carver x61;Vile Hunter x29;Vile Lancer x10","superunique:Treehead WoodFist;unique:Steel Brow"
8,2,57,5,0,0,1,63,"Fallen Shaman x8;Fallen x32;Gargantuan Beast x6;Zombie x16","superunique:Corpsefire"
2,3,48,0,0,0,0,48,"Fallen x25;Quill Rat x13;Zombie x10",""
3,3,160,2,3,0,1,166,"Dark Hunter x27;Fallen Shaman x21;Fallen x105;Gargantuan Beast x9","champion:Champion Fallen Shaman;champion:Champion Fallen Shaman;champion:Possessed Fallen Shaman;superunique:Bishibosh"
4,3,96,11,0,1,1,109,"Carver x32;Fallen Shaman x1;Fallen x6;Foul Crow Nest x3;Foul Crow x35;Hungry Dead x3;Moon Clan x8;Skeleton x19","superunique:Rakanishu;unique:Warp Claw the Hungry"
5,3,119,6,0,1,1,127,"Brute x2;Carver Shaman x4;Carver x92;Skeleton Archer x18;Vile Lancer x9","superunique:Treehead WoodFist;unique:Blood Drinker"
8,3,57,5,0,0,1,63,"Fallen Shaman x9;Fallen x33;Gargantuan Beast x3;Zombie x17","superunique:Corpsefire"
2,4,82,0,0,0,0,82,"Fallen x42;Quill Rat x24;Zombie x16",""
3,4,164,2,4,0,1,171,"Dark Hunter x31;Dark Spearwoman x13;Fallen Shaman x23;Fallen x99","champion:Champion Dark Hunter;champion:Champion Dark Hunter;champion:Champion Dark Hunter;champion:Possessed Dark Hunter;superunique:Bishibosh"
4,4,72,8,2,0,1,83,"Carver x19;Fallen Shaman x3;Fallen x16;Hungry Dead x18;Moon Clan x4;Skeleton x20","champion:Champion Foul Crow;champion:Possessed Foul Crow;superunique:Rakanishu"
5,4,81,10,0,2,0,93,"Carver Shaman x3;Carver x41;Skeleton Archer x18;Spike Fiend x17;Vile Hunter x12","unique:Bile Rend the Flayer;unique:Dread Eater the Jagged"
8,4,68,5,0,0,1,74,"Fallen Shaman x11;Fallen x46;Gargantuan Beast x1;Zombie x15","superunique:Corpsefire"
2,5,63,0,0,0,0,63," x1;Fallen x38;Quill Rat x15;Zombie x9",""
3,5,136,0,4,0,0,140,"Dark Hunter x35;Dark Spearwoman x17;Fallen Shaman x14;Fallen x70","champion:Champion Fallen Shaman;champion:Champion Fallen Shaman;champion:Champion Fallen Shaman;champion:Ghostly Fallen Shaman"
4,5,98,18,0,2,1,119,"Carver x27;Fallen Shaman x3;Fallen x10;Hungry Dead x17;Moon Clan x22;Skeleton x37","superunique:Rakanishu;unique:Gut Flame;unique:Mind Dancer"
5,5,113,10,0,2,0,125,"Carver Shaman x3;Carver x96;Skeleton Archer x14;Vile Lancer x10","unique:Pulse Venom;unique:Rust Spawn"
8,5,41,5,0,0,1,47,"Fallen Shaman x5;Fallen x23;Gargantuan Beast x2;Zombie x16","superunique:Corpsefire"
2,6,61,0,0,0,0,61,"Fallen x21;Quill Rat x16;Zombie x24",""
3,6,54,3,0,1,0,58,"Dark Hunter x36;Dark Spearwoman x11;Gargantuan Beast x10","unique:Gut Maggot the Axe"
4,6,53,0,2,0,0,55,"Carver x8;Foul Crow Nest x1;Foul Crow x24;Moon Clan x6;Skeleton x14","champion:Champion Foul Crow;champion:Ghostly Foul Crow"
5,6,89,2,4,0,1,96,"Brute x2;Carver Shaman x3;Carver x63;Skeleton Archer x6;Vile Lancer x17","champion:Champion Vile Hunter;champion:Champion Vile Hunter;champion:Champion Vile Hunter;champion:Fanatic Vile Hunter;superunique:Treehead WoodFist"
8,6,48,5,0,0,1,54,"Fallen Shaman x5;Fallen x22;Gargantuan Beast x8;Zombie x18","superunique:Corpsefire"
2,7,70,0,0,0,0,70," x1;Fallen x43;Quill Rat x21;Zombie x5",""
3,7,119,7,0,1,1,128,"Dark Hunter x5;Dark Spearwoman x11;Fallen Shaman x19;Fallen x81;Gargantuan Beast x10","superunique:Bishibosh;unique:Soul Widow the Mad"
4,7,98,8,0,2,0,108,"Carver x13;Dark Ranger x5;Fallen Shaman x3;Fallen x12;Foul Crow Nest x1;Foul Crow x37;Moon Clan x6;Skeleton x29","unique:Flame Kill the Howler;unique:Star Fester"
5,7,73,10,0,2,0,85,"Carver Shaman x1;Carver x53;Skeleton Archer x6;Spike Fiend x17;Vile Lancer x6","unique:Bone Feast;unique:Foul Maw the Slasher"
8,7,43,5,0,0,1,49,"Fallen Shaman x6;Fallen x20;Gargantuan Beast x3;Zombie x19","superunique:Corpsefire"
2,8,70,0,0,0,0,70,"Fallen x35;Quill Rat x20;Zombie x15",""
3,8,169,6,0,1,0,176,"Dark Hunter x43;Dark Spearwoman x6;Fallen Shaman x25;Fallen x96;Gargantuan Beast x5","unique:Moon Shifter the Hammer"
4,8,48,12,0,1,1,62," x1;Carver x11;Dark Ranger x10;Hungry Dead x30;Moon Clan x8","superunique:Rakanishu;unique:Pit Grumble"
5,8,125,9,0,2,0,136,"Carver Shaman x5;Carver x50;Spike Fiend x23;Vile Hunter x45;Vile Lancer x11","unique:Blight Thorn the Hammer;unique:Steel Pox the Wraith"
8,8,70,5,0,0,1,76,"Fallen Shaman x13;Fallen x46;Gargantuan Beast x4;Zombie x12","superunique:Corpsefire"
2,9,58,0,0,0,0,58,"Fallen x32;Quill Rat x16;Zombie x10",""
3,9,102,5,0,1,1,109,"Dark Spearwoman x14;Fallen Shaman x16;Fallen x73;Gargantuan Beast x4","superunique:Bishibosh;unique:Plague Sludge the Dark"
4,9,105,14,0,1,1,121,"Carver x24;Dark Ranger x18;Fallen Shaman x3;Fallen x15;Foul Crow Nest x2;Foul Crow x42;Hungry Dead x15","superunique:Rakanishu;unique:Foul Horn"
5,9,134,0,4,0,0,138,"Carver Shaman x3;Carver x101;Spike Fiend x26;Vile Lancer x4","champion:Berserker Vile Hunter;champion:Champion Vile Hunter;champion:Champion Vile Hunter;champion:Champion Vile Hunter"
8,9,50,5,0,0,1,56,"Fallen Shaman x8;Fallen x29;Gargantuan Beast x4;Zombie x14","superunique:Corpsefire"
2,10,83,0,0,0,0,83,"Fallen x48;Quill Rat x20;Zombie x15",""
3,10,117,6,0,1,0,124,"Dark Hunter x27;Dark Spearwoman x6;Fallen Shaman x18;Fallen x66;Gargantuan Beast x6","unique:Haze Razor the Jagged"
4,10,116,8,3,0,1,128,"Carver x25;Dark Ranger x11;Fallen Shaman x3;Fallen x14;Foul Crow Nest x2;Foul Crow x34;Hungry Dead x35","champion:Champion Moon Clan;champion:Champion Moon Clan;champion:Ghostly Moon Clan;superunique:Rakanishu"
5,10,152,9,0,2,0,163,"Carver Shaman x4;Carver x102;Vile Hunter x45;Vile Lancer x10","unique:Chaos Brow;unique:Wrath Eye the Unclean"
8,10,36,5,0,0,1,42,"Fallen Shaman x6;Fallen x21;Gargantuan Beast x2;Zombie x12","superunique:Corpsefire"
2,11,81,0,0,0,0,81," x1;Fallen x42;Quill Rat x19;Zombie x19",""
3,11,62,6,0,1,1,70,"Dark Hunter x35;Dark Spearwoman x9;Fallen Shaman x2;Fallen x16;Gargantuan Beast x6","superunique:Bishibosh;unique:Night Blister"
4,11,45,6,0,2,0,53,"Carver x10;Dark Ranger x3;Foul Crow Nest x2;Foul Crow x21;Hungry Dead x12;Skeleton x3","unique:Black Fester;unique:Rust Crow the Witch"
5,11,159,5,0,1,1,166,"Brute x2;Carver Shaman x2;Carver x124;Vile Hunter x25;Vile Lancer x11","superunique:Treehead WoodFist;unique:Shadow Growler"
8,11,40,5,0,0,1,46,"Fallen Shaman x8;Fallen x23;Zombie x14","superunique:Corpsefire"
2,12,85,0,0,0,0,85,"Fallen x42;Quill Rat x22;Zombie x21",""
3,12,86,3,0,1,0,90,"Dark Spearwoman x19;Fallen Shaman x11;Fallen x48;Gargantuan Beast x11","unique:Haze Widow the Sharp"
4,12,53,5,0,1,0,59," x1;Dark Ranger x12;Hungry Dead x27;Skeleton x18","unique:Gloom Pox the Shade"
5,12,171,0,4,0,0,175,"Carver Shaman x6;Carver x89;Spike Fiend x31;Vile Hunter x38;Vile Lancer x7","champion:Champion Vile Lancer;champion:Champion Vile Lancer;champion:Champion Vile Lancer;champion:Possessed Vile Lancer"
8,12,63,5,0,0,1,69,"Fallen Shaman x10;Fallen x43;Gargantuan Beast x1;Zombie x14","superunique:Corpsefire"
2,13,56,0,0,0,0,56,"Fallen x42;Quill Rat x8;Zombie x6",""
3,13,130,2,4,0,1,137,"Dark Spearwoman x16;Fallen Shaman x17;Fallen x89;Gargantuan Beast x10","champion:Berserker Fallen Shaman;champion:Champion Fallen Shaman;champion:Champion Fallen Shaman;champion:Champion Fallen Shaman;superunique:Bishibosh"
4,13,78,11,0,1,1,91,"Carver x35;Dark Ranger x8;Fallen Shaman x3;Fallen x8;Foul Crow Nest x4;Foul Crow x20;Hungry Dead x11","superunique:Rakanishu;unique:Gore Claw the Grim"
5,13,122,2,3,0,1,128,"Brute x2;Carver Shaman x3;Carver x86;Spike Fiend x8;Vile Hunter x25","champion:Champion Vile Lancer;champion:Champion Vile Lancer;champion:Fanatic Vile Lancer;superunique:Treehead WoodFist"
8,13,49,5,0,0,1,55,"Fallen Shaman x7;Fallen x22;Gargantuan Beast x5;Zombie x20","superunique:Corpsefire"
2,14,69,0,0,0,0,69,"Fallen x34;Quill Rat x20;Zombie x15",""
3,14,152,6,0,1,1,160,"Dark Hunter x44;Dark Spearwoman x12;Fallen Shaman x16;Fallen x82;Gargantuan Beast x4","superunique:Bishibosh;unique:Chaos Brow"
4,14,63,18,0,2,1,84,"Carver x12;Hungry Dead x23;Moon Clan x7;Skeleton x39","superunique:Rakanishu;unique:Rust Grin the Jagged;unique:Sin Widow the Flayer"
5,14,87,2,0,0,1,90,"Brute x2;Carver Shaman x4;Carver x55;Skeleton Archer x10;Spike Fiend x13;Vile Lancer x5","superunique:Treehead WoodFist"
8,14,43,5,0,0,1,49,"Fallen Shaman x8;Fallen x29;Gargantuan Beast x2;Zombie x9","superunique:Corpsefire"
2,15,67,0,0,0,0,67,"Fallen x34;Quill Rat x22;Zombie x11",""
3,15,111,7,0,1,1,120,"Dark Hunter x59;Dark Spearwoman x6;Fallen Shaman x3;Fallen x40;Gargantuan Beast x10","superunique:Bishibosh;unique:Puke Wolf the Impaler"
4,15,110,0,3,0,0,113,"Carver x27;Fallen Shaman x6;Fallen x24;Foul Crow Nest x1;Foul Crow x21;Hungry Dead x15;Skeleton x16","champion:Champion Moon Clan;champion:Champion Moon Clan;champion:Champion Moon Clan"
5,15,138,11,0,2,1,152,"Brute x2;Carver Shaman x4;Carver x95;Vile Hunter x42;Vile Lancer x6","superunique:Treehead WoodFist;unique:Haze Growler the Howler;unique:Puke Feast the Witch"
8,15,20,5,0,0,1,26,"Fallen Shaman x3;Fallen x7;Gargantuan Beast x2;Zombie x13","superunique:Corpsefire"
2,16,84,0,0,0,0,84,"Fallen x50;Quill Rat x20;Zombie x14",""
3,16,205,8,0,1,1,215,"Dark Spearwoman x21;Fallen Shaman x30;Fallen x142;Gargantuan Beast x20","superunique:Bishibosh;unique:Dread Razor the Hammer"
4,16,83,0,2,0,0,85,"Carver x11;Fallen Shaman x3;Fallen x10;Foul Crow x36;Moon Clan x11;Skeleton x12","champion:Champion Foul Crow;champion:Ghostly Foul Crow"
5,16,114,2,3,0,1,120,"Brute x2;Carver Shaman x4;Carver x45;Skeleton Archer x12;Spike Fiend x12;Vile Hunter x41","champion:Champion Vile Lancer;champion:Champion Vile Lancer;champion:Champion Vile Lancer;superunique:Treehead WoodFist"
8,16,48,5,0,0,1,54,"Fallen Shaman x5;Fallen x18;Gargantuan Beast x5;Zombie x25","superunique:Corpsefire"
2,17,64,0,0,0,0,64,"Fallen x31;Quill Rat x17;Zombie x16",""
3,17,64,6,0,1,0,71,"Dark Hunter x46;Dark Spearwoman x14;Gargantuan Beast x10","unique:Steel Eye"
4,17,51,8,4,0,1,64," x1;Carver x13;Hungry Dead x15;Moon Clan x12;Skeleton x18","champion:Champion Skeleton;champion:Champion Skeleton;champion:Champion Skeleton;champion:Ghostly Skeleton;superunique:Rakanishu"
5,17,141,2,0,0,1,144,"Brute x2;Carver Shaman x6;Carver x105;Skeleton Archer x17;Vile Lancer x13","superunique:Treehead WoodFist"
8,17,44,5,0,0,1,50,"Fallen Shaman x5;Fallen x23;Gargantuan Beast x3;Zombie x18","superunique:Corpsefire"
2,18,70,0,0,0,0,70,"Fallen x42;Quill Rat x17;Zombie x11",""
3,18,126,3,0,1,0,130,"Dark Hunter x44;Fallen Shaman x19;Fallen x62;Gargantuan Beast x4","unique:Spirit Hack"
4,18,65,10,0,2,0,77," x1;Carver x17;Dark Ranger x9;Fallen Shaman x3;Fallen x13;Foul Crow x23;Hungry Dead x5;Moon Clan x4","unique:Gut Maim the Shade;unique:Pit Feast"
5,18,88,11,0,2,0,101,"Carver Shaman x4;Carver x65;Spike Fiend x16;Vile Lancer x14","unique:Rot Spell;unique:Snot Hawk the Hunter"
8,18,69,5,0,0,1,75,"Fallen Shaman x9;Fallen x40;Gargantuan Beast x3;Zombie x22","superunique:Corpsefire"
2,19,75,0,0,0,0,75,"Fallen x47;Quill Rat x11;Zombie x17",""
3,19,118,2,2,0,1,123,"Dark Hunter x39;Fallen Shaman x13;Fallen x62;Gargantuan Beast x6","champion:Berserker Fallen Shaman;champion:Champion Fallen Shaman;superunique:Bishibosh"
4,19,65,8,0,0,1,74,"Carver x21;Dark Ranger x10;Fallen Shaman x3;Fallen x11;Hungry Dead x12;Moon Clan x16","superunique:Rakanishu"
5,19,147,8,0,1,1,157,"Brute x2;Carver Shaman x6;Carver x57;Skeleton Archer x15;Spike Fiend x18;Vile Hunter x51;Vile Lancer x6","superunique:Treehead WoodFist;unique:Gore Skin the Mauler"
8,19,47,5,0,0,1,53,"Fallen Shaman x5;Fallen x17;Gargantuan Beast x5;Zombie x25","superunique:Corpsefire"
2,20,60,0,0,0,0,60,"Fallen x42;Quill Rat x6;Zombie x12",""
3,20,138,0,2,0,0,140,"Dark Hunter x31;Fallen Shaman x18;Fallen x83;Gargantuan Beast x6","champion:Champion Fallen Shaman;champion:Possessed Fallen Shaman"
4,20,89,8,3,0,1,101," x1;Carver x36;Dark Ranger x6;Fallen Shaman x4;Fallen x18;Foul Crow Nest x4;Foul Crow x15;Hungry Dead x13","champion:Champion Foul Crow;champion:Champion Foul Crow;champion:Ghostly Foul Crow;superunique:Rakanishu"
5,20,84,5,0,1,0,90,"Carver Shaman x3;Carver x36;Skeleton Archer x16;Spike Fiend x23;Vile Lancer x11","unique:Stone Thirst"
8,20,50,5,0,0,1,56,"Fallen Shaman x7;Fallen x27;Gargantuan Beast x7;Zombie x14","superunique:Corpsefire"

(Mean per seed; min-max of total in brackets.)

## Reading

- Counts per area are large and vary a lot by seed (Cold Plains 58 to 215), so any single-game comparison against the original needs many games, not one.
- Champions and rares are rare in d2d: Blood Moor had **no** champion, rare or minion pack in any of 20 seeds, Cold Plains 0.9 champions and 0.7 rares per game on average. Whether the original spawns a rare in the Blood Moor more often is the open question for the original side (U; this is the number that feeds XP and drop expectations in the early game).
- Three Blood Moor games hold one monster whose name column is empty (seeds 5, 7, 11): left as "" in the CSV (U: not identified; the raw class id is not kept by the probe).
- Same seed gives different totals when the visit order or step sequence differs (a Cold Plains-only run of seed 1 gave 138 normals, the 5-area run 129): the d2d RNG advances with every draw (docs/d2d-fork.md section 2). Counts are comparable as distributions, not per seed.
- Not measured: Normal only, no Nightmare/Hell; monsters spawned later (nightly respawn or waves) are not part of this count; no hostile-versus-friendly split (all listed non-dead monsters counted).
