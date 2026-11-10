# palprofile: Paladin attack order as data (2026-11-10)

Source: the SME consult in `docs/research/project-review-2026-11-10.md` section 4.4 (items 3 and 6), user approved. Tags: V verified here by test or file, S secondary (the SME's reading of the wiki), U unverified.

## Change (`internal/combat/classdata/profiles.json`, Paladin only; V by `TestPaladinBands`)

| band | before | after |
|---|---|---|
| clvl 18 and Blessed Hammer | Blessed Hammer, Zeal, Smite, Attack | Blessed Hammer, Zeal, Attack |
| clvl 12 and Zeal | Zeal, Smite, Attack | Zeal, Attack |
| below (default) | Smite, Attack | Sacrifice, Attack |

- Smite is no longer ranked at all: a 2-mana shield attack kept only as the Charge prerequisite (S). With Attack always last and free, ranking Smite over it was the "mana short" defect; dropping it is the data form of "Smite not over Attack".
- Sacrifice leads only before Zeal (zero mana, big damage bonus, costs life per hit, S). Once Zeal is learned it is out of the order again, as BUILDV2 wanted, because the life cost matters most when the hero is strong enough to be hit a lot. The existing `melee.retreatLifePct` (40) guards the life cost (U: not measured).
- The Sorceress profile is untouched; `TestPaladinBands` checks it still leads with Blizzard and ranks neither Sacrifice nor Smite (V).

## Prayer while fighting: no data change (V by reading `internal/combat/aura.go`)
Prayer already is a resting aura (`restLifePct` 70): `wantAura` skips it in a fight unless no other ladder aura is reachable (Might, Holy Fire, Thorns are listed after it). The format cannot say "never in a fight"; making the fallback unreachable would be a Go rule change, left open. U: whether Prayer ever fires in a fight on the live bot.

## Mana potions before Zeal
The SME's "no mana potions before Zeal" is an economy rule, not a profile field; not changed (open, see the `actplan` or economy cards).

## Cause of the sandbox back-offs: UNSURE
The SME doubts but cannot prove that Prayer draining mana, or Smite being ranked first, caused the "out of mana: backing off" lines. `docs/killloop.md` already names a fix in `resource.AttackChoice` (melee profiles swing out of mana). This change lowers mana use at low level; it is not proven to remove the back-offs. Live check after deploy: `grep -c 'out of mana: backing off'` per hour and the `killtime` mean, one fresh Paladin (not run here, U).

## Doc fixes
`docs/sme/class-paladin.md` (Prayer costs mana) and `docs/research/class-early-game.md` (Zeal and Charge at clvl 12, not 6).
