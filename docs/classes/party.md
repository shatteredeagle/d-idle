# Party play and class synergies (offline LAN / multi-instance bots)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). 1.14d is the target; the Arreat Summit pages are LoD-era (V), Maxroll experience is D2R (V, noted). Design context: `docs/net/multiplayer.md` (Role, LootPolicy, Rusher/Runner analogues).

Sources: Arreat Summit parties https://classic.battle.net/diablo2exp/basics/parties.shtml (V), Arreat monsters https://classic.battle.net/diablo2exp/monsters/basics.shtml (V), Maxroll experience https://maxroll.gg/d2/resources/experience (V, D2R), Maxroll rushing https://maxroll.gg/d2/resources/rushing (V), Maxroll Whirlwind Barbarian (warcries, V), Summoner Necromancer and Lightning Sentry guides (V), search snippets on level gaps (S). The Diablo Wiki Multiplayer and Player Settings pages returned 403.

## 1. Rules that shape a bot party

| Rule | Value | Tag |
|---|---|---|
| Max players per game | 8 (the `/players` command maximum) | V (Arreat monsters) |
| Party size cap | not stated by the Arreat parties page; the game cap of 8 applies | V (absence), U (that parties may fill all 8) |
| Monster life | `Life = base * (players + 1) / 2`: 1.5x at 2 players, 2x at 3, 2.5x at 4, 4.5x at 8 | V |
| Monster XP | same factor as life (base * (n+1)/2) | V (Arreat) / V (Maxroll D2R: +50% per extra player, 450% at 8) |
| Monster damage | +6.25% damage and attack rating per player setting above 1 (P2 6.25% ... P8 43.75%), applied after champion/unique bonuses. Documented for D2R by Maxroll (V https://maxroll.gg/d2/resources/player-settings); PureDiablo's Players X page lists the same table but says it "does not apply to classic Diablo 2" and notes D2R's base is v1.14 (V fetched https://www.purediablo.com/?p=6422). So for 1.14d: treat as likely present, Arreat (LoD-era) does not mention it | V (D2R), S (1.14d) |
| Monster count / density | no source found says the player count changes how many monsters spawn; all pages list only life, damage, XP, drops | U (open) |
| `/players X` | X from the current real player count up to 8; works in single player, open Battle.net and TCP/IP (V Arreat monsters page). Also `/playersX`; X cannot be below the real count; only unpartied players are simulated; monsters already spawned keep their old life, objects update at once (V PureDiablo) | V |
| Minion life | in 1.10+ minions' hit points no longer scale with the setting (S, search snippet) | S |
| Crushing Blow / Corpse Explosion | CB is divided by `0.5 + 0.5*players` (P8 = 22%); CE and Death Sentry use the monster's base life and do not scale (V Maxroll, D2R; CE note also in the Summoner guide) | V (D2R), U (1.14d) |
| XP split | killer's party shares total XP in proportion level / sum of levels; total rises ~35% per extra member in the area (Maxroll D2R: `(n-1)*Exp*87/256`, about 34%) | V |
| Level gap | gap <= 5 gets full share, 5-10 sliding, >= 10 gets 5% minimum (applies even in a party) | V |
| Auras and warcries | Paladin auras and Barbarian warcries apply to party members (V Arreat parties https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/basics/parties.shtml); radius in yards: Might and Concentration 10.6 at level 1 growing about 1.3 per level (V Arreat paladin-offense); Battle Orders 2 at level 1 up to 14.6 (V Arreat barbarian-warcries); Act 2 mercenary auras reach the whole party and minions (V Arreat hirelings) | V |
| Share conditions | same named area and about two screens from the death; Maxroll D2R says 53.33 yards | V |
| Gold | split among members in the same area, cannot be turned off | V |
| Drops (formula) | PureDiablo gives the nodrop formula `N = int(1 + AdditionalPlayers/2 + ClosePartiedPlayers/2)`; uniques and champions drop one item, super uniques two, regardless; 7 and 8 players give the same rate (V PureDiablo). Extra players raise the chance of items dropping (not quality); every two additional other players count as one extra member for the bonus, including non-party; uniques always drop magic+ and are not changed | V |
| Quests | shared with the whole party if in the same Act (rush); players who die should stay dead since quests still count | V (Maxroll rushing) |
| Players setting | forced `/players X` works in single player, open BNet and TCP/IP; the count is game-wide, so bots in a LAN game scale monsters even if idle in town | V (setting), U (idle-in-town effect) |

Implications:
- Four bots mean monsters have 2.5x life and give 2.5x XP: kills take longer but yield more per kill. Total XP per bot per kill (U arithmetic): single kill XP S, party of 4 with equal levels each gets about `S * 2.5 * (1 + 3*0.34) / 4`, roughly 1.26 S each, plus 4 times as many kills per minute. Net speed-up is about 3-5x for killers, less for the idle ones. (the formula is V, the arithmetic is mine.)
- XP needs everyone within ~2 screens and in the same area: followers must not wander (see Role `follower`).
- Level gap: keep party levels within 5 for full shares; a much lower leech gets 5% of a share multiplied by level fraction (V), so a mule/leech at low level is cheap but gets little.
- Corpse Explosion and other kill-based damage "does not scale with player count" (V Necro summoner), so CE-based Necromancers are weaker in a bigger game; Hell bosses with 4x life hurt a 1-bot killer.
- Hell/Nightmare gate: you cannot enter a difficulty without its entry quest completion in each character (U); a Rusher/Runner arrangement bypasses with TP (V Maxroll rushing: quests shared in the same Act).

## 2. Synergies by type

Everything here is U unless tagged.

| Synergy | Who | Needs orchestrator? |
|---|---|---|
| Battle Command (+1 all skills), Battle Orders (life/mana/stamina), Shout (defense), Battle Cry/Grim Ward | Barbarian buffs party members (V Maxroll WW) | No: cast on arrival; but range and "keep alive" need the leader to stay near |
| Paladin auras (Concentration, Fanaticism, Vigor, Holy Shield on self, Resist auras, Redemption, Meditation) | affect nearby party members (U); same-aura copies from multiple sources do not stack, highest wins (U) | No, but who stands near whom matters: a "stay within aura radius" follower behaviour |
| Conviction aura (merc/paladin/Infinity) lowers fire/cold/lightning resistance; works at 1/5 strength on immunes (V Maxroll immunities) | Mercs and Paladins; benefits fire/cold/lightning classes (Sorc, Druid fire, Trapsin, Amazon) | No (passive) |
| Curses: Amplify Damage (physical resist), Lower Resist (elemental+poison), Decrepify (slow, V), Life Tap | Necromancer helps everybody; Lower Resist -14% on immunes (V, 1/5) | Mild: curse timing via shared "targeted" signal, otherwise each caster picks its own |
| Mixed damage types | Sorc cold + Paladin magic hammers + Barb physical = few immunes | Planner chooses composition (immunity coverage) |
| Teleporter + killer | Sorceress Teleport, Paladin/Druid/Necro with Staff of Teleportation; V Rusher role; teleporter runs ahead and places TP, killer follows | **Yes**: ready/arrived handshake via orchestrator (design in multiplayer.md) |
| Tank/aggro | Paladin with Holy Shield, Necro skeleton army, Druid wolves/bear, Amazon Valkyrie; Barbarian taunt/howl | Partially: tank positions in front of the killer (leader/follower model) |
| Oak Sage / Heart of Wolverine / Fanaticism | Druid spirits boost party life/damage (V Fissure guide: Oak Sage life aura) | No |
| Sentry traps and Death Sentry corpse explosions | Assassin; trap damage persists while the party chokes mobs | Yes if traps must be placed where the tank holds monsters |
| Mule | any low-level or spare char parked in town; drops/pickup | Yes: LootPolicy |
| Rush / TP leech | Sorc runs, others skip areas via a TP | Yes (shared quests V) |

Notes on stacking (U): same-source buffs of different casters (e.g., two Battle Orders) do not stack; different auras do. Wise default: avoid two characters of the same support type.

## 3. Compositions

### 2 bots
| Composition | Strength | Risk |
|---|---|---|
| Sorceress + Paladin (Hammerdin) | ranged cold + magic hammers, Teleport for navigation, Paladin Vigor and Concentration help both (U), Holy Shield tank; few immunes together | Paladin must stay near; both want the same mana potions in Hell |
| Sorceress + Necromancer | Necro curses (Lower Resist) fix her cold/lightning immunes (V 1/5 on immunes), minions tank | Necro slow to set up; CE weaker with more players (V) |
| Paladin + Barbarian | strongest melee support pair (Battle Orders/Command + auras) | no Teleport unless Staff charges; both melee in tight rooms |
| Two Sorceresses | simple code reuse | same immunity gap: not recommended |

### 3 bots
- Sorceress (teleporter/killer) + Paladin (tank/aura) + Necromancer (curser/summoner): best generalist composition; roles map cleanly (U).
- Sorceress + Paladin + Barbarian (buffer): buffs improve all damage; low synergy in immunes (physical + magic + cold).
- Sorceress + Paladin + Amazon: three damage types; ranged cluster, Amazon ammo needs.

### 4 bots
- Rush/farm four: Sorceress (teleporter/killer), Paladin (tank/killer), Necromancer (curser/summoner), Barbarian (buffer). Monsters have 2.5x life and XP (V), shares about 1.26x per kill each (U math above). Risk: three non-teleporters need the orchestrator for TP handshake; bigger monsters risk Hell elites with lightning/mana burn.
- Mule bot as a 4th: Sorceress + Paladin + Necromancer + mule parked in town; the mule can be a level-1 character; leech XP is nearly 0 (V gap rule) but monsters scale for 4 players (V), which makes the game harder even though the mule does nothing. Use a mule only for the drop phase (shut down afterwards, or join after the kills).

### Bigger parties
Up to 8 is allowed (V) but monsters have 4.5x life at 8 players (V); run time limit in CPU: each instance is about a core on the worker host (AGENTS.md), so 4 is a practical limit for now.

## 4. Mapping to `party.Role`

multiplayer.md defines `Role`: leader, killer, teleporter, mule, follower, solo.

| Role | Natural classes | Behaviour change |
|---|---|---|
| leader | Sorceress, Paladin | owns the run plan, opens TPs and games |
| teleporter | Sorceress (native), others with Staff/Enigma | runs ahead; waits on "ready" |
| killer | Sorceress, Paladin, Amazon, Assassin, Druid | clears; uses `Clear` with a party-aware kill-priority |
| follower | any | stays within two screens (V XP rule) of the leader; no independent loot |
| mule | spare char | parked; picks up drops; only moves on command |
| solo | any | default today |

Missing from the model (suggested additions, U):
1. **buffer** (Barbarian warcries, Paladin auras, Druid spirits): a role (or a capability flag) meaning "stand near allies, recast buffs on a timer and on arrival in a new area".
2. **curser/support** (Necromancer): targets elites first, sets curses, summons; interacts with the killers' targeting.
3. **tank/aggro**: keeps monsters at a choke; decides front position; Paladin, Necro minions, Druid pets.
4. **Capabilities as flags** instead of a single enum: `CanTeleport`, `AuraRadius`, `Buffs[]`, `Summons`, `Ranged`, `ImmunityCoverage` so the orchestrator can match composition to the area. A character may be both killer and buffer.
5. **Party XP guard**: `xpRange` (same area, two screens) and `levelGap <= 5` as follower constraints (V).
6. **Player-count tuning**: `Tuning.PlayersX` expected (1-4) so potion thresholds and chicken rules rise with monster life scaling (V).
7. **Roster state**: party invite/accept flags per roster entry (multiplayer.md PROPOSED T6).

## 5. Which synergies need nothing special vs a coordinating orchestrator

Need nothing special (passive, same-area): Barbarian warcries, Paladin auras, Conviction, Oak Sage, curse effects, damage-type coverage, gold split (cannot be disabled, V), XP share (automatic if in range).

Need the Party orchestrator: teleporter+killer handshake and TP placement, rush quest hand-off (V), mule drop/pickup LootPolicy, who tanks where, buff timing on new areas, "all in the same area before kill" gate for XP, hostile/anti-grief handling (multiplayer.md section 4), area-ready barriers, and staging of Players X expectations.

## Live results (agent PARTYLIVE, 2026-10-28, two Sorceresses Navi L1 and Combi L18 on i1/i2, both in the Rogue Encampment)
- V: two offline clients join one TCP/IP game over 127.0.0.1; both see each other's unit and roster entry (docs/net/multiplayer.md section 9).
- V: joining does not party the players (partyid 65535). The party panel (`p`, INVITE/ACCEPT button at about (297,139) for the first other player) works through posted input: invite sets roster partyflag 4 on the inviter's list and 2 on the invitee's, accept gives both partyid 3 and partyflag 1 (in party). Rosters were rebuilt on every rejoin, so the bot must re-invite after each (re)join.
- V: a 2-player game needs nothing typed except the host IP; no game name, password or difficulty dialog.
- Not observed (U): monster life/XP/damage scaling at 2 players, XP and gold sharing, loot visibility, Players X, whether an idle player in town raises n. No monster was fought, so the three open questions below stay open.

## Open questions
- Is the 6.25% damage step present in 1.14d (S: D2R yes, PureDiablo says not 'classic')? Needs a live test on two instances (compare a monster's damage at P1 vs `/players 8`).
- Does `/players X` change monster density? No source says so (U); live check by counting monsters in a fixed area.
- Can an idle-in-town bot reduce the Players X count by leaving? (leaving the game does; idling does not, U).
