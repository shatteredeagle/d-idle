# palprofile: Paladin attack order as data (2026-11-10)

> status: the attack bands below were superseded on 2026-11-12 by the section "Round 2" at the end (Sacrifice band, Holy Fire crowd rule).

Source: the SME consult in `docs/research/project-review-2026-11-10.md` section 4.4 (items 3 and 6), user approved. Tags: V verified here by test or file, S secondary (the SME's reading of the wiki), U unverified.

## Change (`internal/combat/classdata/profiles.json`, Paladin only; V by `TestPaladinBands`)

| band | before | after |
|---|---|---|
| clvl 18 and Blessed Hammer | Blessed Hammer, Zeal, Smite, Attack | Blessed Hammer, Zeal, Attack |
| clvl 12 and Zeal | Zeal, Smite, Attack | Zeal, Attack |
| below (default) | Smite, Attack | Attack |

- Smite is no longer ranked at all: a 2-mana shield attack kept only as the Charge prerequisite (S). With Attack always last and free, ranking Smite over it was the "mana short" defect; dropping it is the data form of "Smite not over Attack".
- Reading of the brief: "Sacrifice before Zeal" is the skill-point order (Sacrifice is the Zeal prerequisite; the build tables already take it at clvl 3, V `TestBuildV2PaladinSequence`), not an attack rung. Sacrifice costs life per hit (S), so it stays out of the order, as BUILDV2 and `TestPalProbeLadderAlongBothTables` already require. Before Zeal the order is plain Attack under Might (zero mana). Open choice for the user: if the SME meant Sacrifice as an attack rung, add it to the default band; the tests then change in `internal/leveler`.
- The Sorceress profile is untouched; `TestPaladinBands` checks it still leads with Blizzard and ranks neither Sacrifice nor Smite (V).

## Prayer while fighting: no data change (V by reading `internal/combat/aura.go`)
Prayer already is a resting aura (`restLifePct` 70): `wantAura` skips it in a fight unless no other ladder aura is reachable (Might, Holy Fire, Thorns are listed after it). The format cannot say "never in a fight"; making the fallback unreachable would be a Go rule change, left open. U: whether Prayer ever fires in a fight on the live bot.

## Mana potions before Zeal
The SME's "no mana potions before Zeal" is an economy rule, not a profile field; not changed (open, see the `actplan` or economy cards).

## Cause of the sandbox back-offs: UNSURE
The SME doubts but cannot prove that Prayer draining mana, or Smite being ranked first, caused the "out of mana: backing off" lines. `docs/killloop.md` already names a fix in `resource.AttackChoice` (melee profiles swing out of mana). This change lowers mana use at low level; it is not proven to remove the back-offs. Live check after deploy: `grep -c 'out of mana: backing off'` per hour and the `killtime` mean, one fresh Paladin (not run here, U).

## Doc fixes
`docs/sme/class-paladin.md` (Prayer costs mana) and `docs/research/class-early-game.md` (Zeal and Charge at clvl 12, not 6).

## Round 2 (2026-11-12, card palprofile2): Sacrifice and the aura ladder

Input: decomp13's PALSKILL matrix (`docs/research/pal-skill-matrix.md`, branch `agent/decomp13`) and the SME order. Tags: V verified by test here, I the matrix's arithmetic model, U unverified live.

### What changed (data in `internal/combat/classdata/profiles.json`, code only for the gates)

| Paladin band | before | after |
|---|---|---|
| default | Attack | clvl 2+ with Sacrifice learned: Sacrifice, Attack; else Attack |
| clvl 12 and Zeal | Zeal, Attack | Sacrifice, Zeal, Attack (Sacrifice 1 28.7 dps against Zeal 1 23.8, I) |
| clvl 15 and Zeal | (same band) | Zeal, Sacrifice, Attack (Zeal 4 31.8 passes Sacrifice 1, I) |
| clvl 18 and Blessed Hammer | Blessed Hammer, Zeal, Attack | Blessed Hammer, Zeal, Sacrifice, Attack |

The aura ladder keeps Prayer (rest), Holy Fire, Might, Thorns, but Holy Fire carries `needsCrowd`: it is wanted only with `combat.holyFireMinMonsters` (3) hostiles within 8 subtiles of the player (the fight loop counts them before `KeepAura`); with fewer, or outside a fight, Might comes next. Once Holy Fire is on it stays until the count drops below the knob minus one (no flip per tick). Reason (I, matrix 3.3/3.4): Might 4 adds +4.5 dps on 8.6 at clvl 6; Holy Fire 4 gives 1.75 dps per monster, so it needs about 3 in reach.

Skill points (no change, `build-paladin-normal30-v2.json`): Might 1 at clvl 2, Sacrifice 1 at clvl 3 (the Zeal prerequisite, now also used as the attack), Prayer 1 at clvl 4, Might to 8 by clvl 11, Zeal 4 at 12 to 15, Holy Fire only from clvl 16. That already matches the SME order (no Holy Fire at 6, Vengeance late); the legacy `normal30` table (existing characters) buys Holy Fire at 6 and is left alone.

### B1: why Sacrifice is on by default (life accounting)

Sacrifice takes 8 percent of the damage it deals (V read in game.exe by decomp13; the cap and the before/after-blow rule are U). A swing costs a few points of 80 to 200 life. The bot cannot drain itself into a chicken loop because `Profile.LifePrice` skills (Sacrifice) are offered only while all hold (`internal/combat/lifeprice.go`, V by `TestSacrificeNeverUnderTheLifeFloor`, `TestSacrificeNeedsPotionsAndLiveMerc`):
- `combat.sacrifice` is 1;
- life is at least `combat.sacrificeMinLifePct` (70); below it, plain Attack until life is 10 points above the floor again (hysteresis, 80 by default). The chicken line (25), the drink line (55) and the retreat line (40) all lie below the floor, so the potion and chicken logic never see a life drop caused by Sacrifice alone: the skill stops at 70 and the next swing is a plain Attack;
- the belt holds `combat.sacrificeMinPotions` (1) healing or rejuvenation potions;
- a hired merc is not dead (when the merc wiring reports one).
State is noted once per fight tick (`noteLifePrice`); before the first note the skill is not offered. Every band lists Attack last, so the fallback always exists. Other classes have no `lifePrice` entry (V by `TestLifePriceOnlyPaladin`).

### B3: Zeal kill times from d2d runs are pessimistic

d2d's Zeal gap is 15 ticks against 7 in game.exe (V run, decomp13 `pal-zeal-trace.md`): a 5-strike cast is 75 ticks against 43, so kill-time model inputs measured on d2d are about 1.74 times too slow for Zeal until the fork fix (`fid/zeal-gap-r1`, `agent/przeal`) lands. The bot is not changed for this; do not tune Zeal against Sacrifice from d2d kill times (the clvl 15 split above is the model, not a measurement).

### Live-check list (sandbox, after the measured first-hour run ends; nothing here ran live)
1. Fresh Paladin, Sacrifice 1 at clvl 3: confirm the left skill switches to Sacrifice (`cast` log shows Sacrifice) and a kill still takes about the same swings (damage x1.9, U).
2. Log life before and after a Fallen kill: the cost should be about 8 percent of the damage dealt (decomp drill C1: 4 for a capped 50-life dummy, not 40). Confirm the real cost and whether it is capped at the target's life.
3. Run to the floor on purpose (no potions drunk): at 70 percent the cast log must show Attack, and Sacrifice must return at 80.
4. Empty the belt of potions: no Sacrifice.
5. Fight with two Fallen then four Fallen with Holy Fire learned (clvl 16+ table, or give the point on a throwaway character): the aura should be Might with 2, Holy Fire with 3 or more; count aura switches per minute (should not flip).
6. Compare potions drunk per area against a run with `combat.sacrifice` 0. If potion use rises more than the kill-speed gain justifies, set the knob to 0 (default for the profile can change in one line).
