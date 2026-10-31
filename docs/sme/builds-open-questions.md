# SME builds: resolved open questions (classic LoD 1.14d, offline SP)

Owner: SMEKB-B2. Resolves open questions left in `builds-gear.md` section 9, `builds-endgame-melee.md` section 6 and `builds-leveling.md` section 8.
Tags: **V** fetched or read this session (source named), **S** secondary or snippet, **U** memory or inference. D2R flagged where it differs.

Sources used:
- **T**: the install's own game tables, `runes.tsv`, `skills.tsv`, etc. in the worker host's extraction cache `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/` (extracted by `internal/data` from the 1.14d MPQs; same tables the game reads). Strongest source available.
- **AS**: Arreat Summit archive, still served at `https://classic.battle.net/diablo2exp/...` (Blizzard primary, 1.10-1.11 era).
- **AB**: Amazon Basin wiki `https://theamazonbasin.com/wiki/index.php?title=<Name>` (fetched via WebFetch summaries, so numbers were cross-checked against T).
- **AG**: Almar's Guides `https://almarsguides.com/Computer/Games/Diablo2/Misc/RuneWords/LadderOnlyInSinglePlayer/`.

## Q1. Which Act 5 quest gives Ral, Ort and Tal?

**Answer: "Rescue on Mount Arreat" (Act 5 Quest 2), turned in to Qual-Kehk after freeing the 15 captive barbarians (3 groups) in the Frigid Highlands. Reward: the three runes Ral + Ort + Tal and the ability to hire Barbarian mercenaries.**
- V AS runes page `https://classic.battle.net/diablo2exp/items/runes.shtml`: "You receive Ral, Ort and Tal as a quest reward in Act V."
- V Almar's `https://almarsguides.com/Computer/Games/Diablo2/Quests/Act5/2RescueonMountArreat/`: "Reward: Can Hire Barbarian Mercenaries; Ral Ort and Tal High Runes ... Qual Kehk gives you the Ral, Ort and Tal".
- V Maxroll `https://maxroll.gg/d2/resources/important-quests` agrees (D2R page, same quest in 1.14d).
- The Arreat Summit "Quest Rewards" page (`.../quests/rewards.shtml`) lists Act V quests 1, 3, 4, 5 (sockets, scroll of resistance + rare, personalize, exp) but omits quest 2; quest 2 is the rune quest per the sources above. V.
- Other rune rewards in vanilla 1.14d (V AS rewards page): Act 4 Hellforge gives 1 random rune per difficulty: Normal El-Amn, Nightmare Sol-Um, Hell Hel-Gul (plus 4 gems). Act 5 Larzuk socket, Anya scroll (+10 all res) and a rare class item.
- Per-difficulty repeat: S/U. Each difficulty has its own quest state, so the reward should repeat in Nightmare and Hell (U, community practice); not fetched. A bot can check the in-game quest bit instead (see `docs/quest.md`). Ral+Ort+Tal is exactly Ancient's Pledge (shield, V T: Runeword1 `Ral+Ort+Tal`, `shld`, not ladder).
- Easy reachability note for a bot: all 15 barbarians must be freed in one game session (S almarsguides snippet), they stand in three cages groups in the Frigid Highlands, the quest is gated behind Shenk (U).

## Q2. 1.14d effects of Chains of Honor, Stone, Memory, Delirium

All four are non-ladder (the `server` column of `runes.tsv` is empty, V T) so they can be made offline. All use runes in the original order. The stats below are the runeword's own properties (T); the gem/rune socket bonuses of the inserted runes are added on top.

**Chains of Honor** (display name; internal row "Bound by Duty", Runeword14, V T `strings.tsv` key `Runeword14` = "Chains of Honor").
- Body armor, 4 sockets, Dol+Um+Ber+Ist. clvl req 63 (AB, S).
- T: +2 all skills, +200% damage to demons, +100% damage to undead, 8% life steal, +70% enhanced defense, +20 Str, all resist +50. AB adds Replenish Life +7, Damage Reduced by 8%, 25% MF (not visible in our tsv extract, which keeps the first seven properties per runeword; those three come from the Ber/Dol/Ist rune bonuses in armor: Dol Replenish +7, Ber DR 8%, Ist MF 25%; V AB, U on which rune).
- All Resistances +65 on the Arreat page (V AS runewords-110) is the sum: +50 from the runeword and +15 from the Um rune socketed in body armor. So the 65 resists figure is real on the finished item; the "50" in our table is only the runeword part. V (arithmetic of AS vs T vs AB).
- Offline use: needs a 4-socket body armor (Larzuk or a 4os drop) and four runes that are all high: Dol (rune 14), Um (22), Ber (30), Ist (24). The rune upgrade recipes from Hel (15) up are ladder-flagged in cubemain (V `docs/items.md` 3.1), so these must be found as drops (Ber is a Hell-only drop in practice, U).

**Stone** (Runeword137, V T).
- Body armor, 4 sockets, Shael+Um+Pul+Lum, clvl 47.
- T: +40% faster hit recovery, +220-260% enhanced defense, 300 defense vs missile, +16 Str, +16 Vit, charges: Level 16 Clay Golem (16), Level 16 Molten Boulder (80), balance2 40 (the +40% FHR). AS shows +60% FHR and +10 Energy and all res +15: that is +40 (runeword) +20 (Shael in armor) and +10 Energy (Lum) and +15 res (Um), consistent with T. V.
- Non-ladder, so the best FHR armor; useful for casters in Act 1-2 before Enigma (S). D2R: same.

**Memory** (Runeword83, V T).
- Staff, 4 sockets, Lum+Io+Sol+Eth.
- T: +33% faster cast rate, +20% maximum mana, +50% enhanced defense, magic damage reduced by 7, +3 sorceress skill levels, +3 Energy Shield, +2 Static Field. AB confirms (sorceress-only skills; clvl 37, AB S). Gem bonuses on top: Lum +10 Energy, Io +10 Vit, Sol +9 min damage, Eth -25% target defense (V AB table).
- Staff only. Non-ladder. For a Sorceress it is the best early staff (3+3+... skills); on other classes only the FCR/mana/defense matter.

**Delirium** (Runeword22, V T).
- Helm, 3 sockets, Lem+Ist+Io, clvl 51 (AB S).
- T: +2 all skills, +261 defense (ac), 11% chance Level 18 Confuse on striking, 14% chance Level 13 Terror when hit, 6% chance Level 14 Mind Blast when hit, 1% chance Level 50 "Delerium Change" when hit (the transform: undead rat man, uses monster `bonefetish1`, AB V), charges Level 17 Attract (60). Gem bonuses: Lem +50% gold, Ist +25% MF, Io +10 Vit.
- DANGER for a bot: when the 1% Delirium proc fires the character turns into a rat man for several seconds, with a changed skill bar, no blocking animation (AB V) and no normal skills. The duration is not given in AB (the `Delerium Change` skills.txt row can be read in T; not yet inspected; U). A bot reading its skill panel could misread this state, so avoid Delirium on an unattended bot unless the transform is detected and waited out. S/U.

D2R: all four exist with the same names in 1.14d and D2R; Chains of Honor, Stone, Memory and Delirium are in both ladder and non-ladder D2R (D2R does not separate ladder). U.

## Q3. Are Grief and Voice of Reason ladder-only offline?

**Answer: Yes, both carry the ladder ("server") flag in this install's `runes.tsv`, so they are NOT craftable in vanilla 1.14d offline play.** V.
- T: row Runeword47 (display "Grief") and Runeword99 (display "Voice of Reason") have `server = 1`. The internal names in the Rune Name column are "Widowmaker" (Eth+Tir+Lo+Mal+Ral, `axe`/`swor`) and "Winter" (Lem+Ko+El+Eld, `mace`/`swor`), which is why the first pass's Project list (23 rows incl. "Widowmaker" and "Winter") and Arreat/others (Grief, Voice of Reason) looked different. Display names from `strings.tsv` keys Runeword47/99. The two lists are the same 23 words: Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath.
- V AS `https://classic.battle.net/diablo2exp/items/runewords-110.shtml`: "The following Rune Words will only work on the Battle.net Realms for Ladder Characters only. They will not work for single or open characters or non-ladder characters", then the list (Brand, Death, ..., Grief ... Voice of Reason). The 1.11 page (`runewords-111.shtml`) states the 1.11 words (Bone, Enlightenment, Myth, Peace, Principle, Rain, Treachery) "will work in single-player, for Open Characters, and for Realm Characters".
- V AG: in 1.14d single player the ladder-only runewords are only available by changing `runes.txt` through `-direct -txt` and a custom `data\global\excel` (modded game, not vanilla). Without that the items do not form. "Previously created" ladder words stop working when `-direct -txt` is dropped. Our project must not assume them.
- Earlier first-pass belief (items.md 3.1, "offline makes everything") was wrong: the Arreat page and almarsguides both disagree. The first-pass statement `builds-gear.md` section 0 ("ladder-only not craftable offline") is correct. Resolved.
- Grief (ladder) stats for reference (T): 35% Venom lvl 15, +30-40% IAS, +340-400 damage, ignore target defense, -25% target defense, +1.875%/clvl damage to demons, 5-30 fire damage, -20-25% enemy poison resist, 20% deadly strike(?), prevent monster heal(?), +2 mana/+10-15 life per kill. Voice of Reason: 15% Frozen Orb lvl13, 18% Ice Blast lvl 20, +220-350% demon damage, +280-300% undead damage, +100-220 cold damage, -24% enemy cold resist, cannot be frozen, +10 Dex, 75% extra gold.
- D2R: no ladder split, so both exist everywhere in D2R. Do not copy Maxroll's D2R builds around Grief for the 1.14d bot.

## Q4. Also checked: the same ladder list applies to the Hel-and-above cube rune recipes (V `docs/items.md` 3.1 cubemain; AS cube page snippet). Unchanged from first pass.

## Q5. Correction to the first pass: Act 2 mercenary auras in Hell

`builds-leveling.md` section 1 says the Act 2 merc runs "Thorns/Holy Freeze/Might in Nightmare and Hell". V (Arreat `https://classic.battle.net/diablo2exp/basics/hirelings.shtml` and `hireling.tsv` of this install): Thorns, Holy Freeze and Might are the **Nightmare** auras; **Hell** reverts to Prayer (Combat), Defiance (Defense) and Blessed Aim (Offense), like Normal (Normal: Prayer / Defiance / Blessed Aim). So a Might merc exists only in Nightmare. Hell mercenary cost 15,000 (T). Use this table for readiness checks.

## Q6. Correction: Holy Bolt is not a general attack in 1.14d

V (Amazon Basin `title=Holy_Bolt`): in LoD the bolt damages only hostile undead and heals allies; the +50% per Blessed Hammer level is "magic damage to hostile undead". First-pass notes that treat Holy Bolt as a starter attack (`docs/classes/paladin.md` Holy Fire/Holy Bolt lines) should not.

## Q7. Cross-check of counts (for the class files)

Immunity counts in `class-*.md` come from `levels.tsv` spawn lists joined to `monstats.tsv`; counting every enabled killable monster entry in Hell gives larger totals (fire 83, lightning 74, cold 108, poison 93, physical 15, magic 6 distinct names). The per-act spawn-list counts (fire 54 across Acts 1-5) are the ones used for "what can I meet in this act".
