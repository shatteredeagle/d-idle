"""covmonai2: append this card's monster.ai rows to docs/coverage/ledger.tsv (idempotent: skips when the evidence doc already has rows).
usage: python3 -I tools/emu/mai3_ledger.py"""
T = "docs/research/covmonai2-trace.md"
A = "Duriel,Summoner,Mephisto,Izual,Hydra,TrappedSoul,UberMephisto,UberDiablo,UberIzual"
B = ("Mummy,GreaterMummy,Vulture,Baboon,SandRaider,ClawViper,SandLeaper,Scarab,Swarm,PantherWoman,PantherJavelin,PinHead,SandMaggot,SandMaggotQueen,"
     "MaggotEgg,MaggotLarva,Mosquito,ThornHulk,WillOWisp,ZakarumZealot,ZakarumPriest,FrogDemon,Tentacle,TentacleHead,FetishShaman,HighPriest,FetishBlowgun,BatDemon")
C = ("ReanimatedHorde,SiegeBeast,Minion,SuicideMinion,Succubus,SuccubusWitch,Overseer,MinionSpawner,Imp,Catapult,FrozenHorror,BloodLord,CatapultSpotter,NpcBarb,"
     "GenericSpawner,DeathMauler,PutridDefiler,BaalTentacle,BaalMinion,Ancient,Nihlathak,ClawViperEx,BaalCrabClone,UberBaal,SiegeTower,HellMeteor,Sarcophagus,Spirit,"
     "ArcaneTower,Trap-RightArrow,Trap-LeftArrow,DesertTurret,VileMother,VileDog,FingerMage,Regurgitator,DoomKnight,AbyssKnight,OblivionKnight,Megademon")
rows = [
    ("monster.ai", A, "TESTED-0-DIFF", T, "27000", "seed 20261105", "think act, pace, effects, AI control words and seed: game.exe think vs d2d rules::mon_think, 3,000 cases each, Act 2 to 4 bosses (Duriel, Summoner, Mephisto, Izual, Hydra, Trapped Soul, Uber Izual; UberMephisto and UberDiablo are a bare ret 4); harness hooks listed in the doc"),
    ("monster.ai", B, "TESTED-0-DIFF", T, "42000", "seed 20261105", "same aspect, 1,500 cases each, Act 2 to 4 monsters (mummies, vultures, baboons, sand raiders, scarabs, panthers, maggots, Act 3 and Act 4 thinks); WillOWisp on its reachable dance states"),
    ("monster.ai", C, "TESTED-0-DIFF", T, "60000", "seed 20261105", "same aspect, 1,500 cases each, Act 4 casters and Act 5 thinks, Baal's minions, towers, traps; Ancient with the target placed at the AI distance, CatapultSpotter and PutridDefiler with raw skill ids relabelled"),
    ("monster.ai", "Diablo", "TESTED-DEVIATION", T, "3000", "seed 20261105", "61 of 3,000 differ: D4 (Skill1 pending with no target casts at none), D5 (act 15 cast carries unit id and type); 18 act or n differences not isolated (U)"),
    ("monster.ai", "Arach", "TESTED-DEVIATION", T, "1500", "seed 20261105", "378 of 1,500 differ: D2 charge is a run in game.exe (337), D3 Spider Lay casts at no unit (41); nothing else differs once a flags-0 back-off that cannot set off is modelled as the idle it is"),
    ("monster.ai", "Fetish", "TESTED-DEVIATION", T, "1500", "seed 20261105", "64 of 1,500 differ, all pace 50 vs 0: D1 a back-off that cannot set off keeps pace 50 in game.exe (57), 7 more not isolated"),
]
text = open("docs/coverage/ledger.tsv").read()
if T in text:
    print("rows already there")
else:
    if not text.endswith("\n"):
        text += "\n"
    text += "".join("\t".join(r) + "\n" for r in rows)
    open("docs/coverage/ledger.tsv", "w").write(text)
    print("added", len(rows))
