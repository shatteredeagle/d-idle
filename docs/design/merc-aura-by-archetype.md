# Merc aura by class archetype (design note, no code)

> status: proposal (card classkeys). Nothing here is built. Tags: V verified in the repo by reading code, S secondary, U unverified.

## Today (V, read in this worktree)
- `town.HirePolicy` (internal/town/merc.go) picks the aura from the difficulty alone: Normal and Hell `Defiance`, Nightmare `Might`. It takes no class (V).
- `internal/task/act2.go` `mercNodes` and `quest.Act2Merc` hard-code `Defiance` (V).
- `internal/plandata/act2plan.json` `merc.aura` already has a class split, `{"melee":"Defiance","caster":"Defiance"}`, and `Act2Plan.Merc.Aura` loads it, but nothing reads it (V: grep finds no consumer outside the loader). Both values are equal, so no class gets a different aura.
- The class facts live in `internal/d2/classtraits.json` (`ClassTraits`: `caster`, `manaNeed`, `mercEarlyRogue`), read through `d2.TraitsOf`, which returns an error for an unknown class (V).
- `d2.Merc.Aura` is never readable from memory (V comment in internal/d2/merc.go); the hire window text is the only source, so the wanted aura is data the hire step carries.

## What a Necromancer wants (S, U for the numbers)
A summoner's damage comes from minions and its own life is thin, so the useful Act 2 rows are the ones that protect or heal the hero (Prayer) or raise the minions' hit rate (Blessed Aim does not apply to minions, so it is not it). Defiance helps every build and stays the default. The choice between Prayer (Normal and Hell rows exist, V hireling.txt via docs/progression/merc.md) and Defiance for the Necromancer is a balance decision for the user or the SME; this note only makes it expressible as data. Nightmare has Might only among the damage auras (V HirePolicy comment): a summoner gains nothing from Might on its own hits, so a Nightmare override per archetype is also a data row.

## Field
Put the aura in the class traits, keyed by difficulty, with today's value as the default so no other class changes:

```json
"Necromancer": { "caster": true, "manaNeed": 2, "mercEarlyRogue": true,
                 "mercAura": { "normal": "Prayer" } }
```

- `mercAura` is `map[string]string`, difficulty name (`normal`, `nightmare`, `hell`) to a skills.txt aura name. An absent key falls back to the difficulty default that `HirePolicy` has today. An empty map means "no class preference".
- The value is checked where the hire step reads the hireling rows (`town.mercAura` names each row's aura from `hireling.txt`): a configured aura that no row of the act and difficulty offers is an error naming the class, difficulty and aura, never a silent fallback to Defiance. A JSON-only check at load is limited to a non-empty string and a known difficulty key.
- `act2plan.json` `merc.aura` (`melee`/`caster`) is then redundant. Drop it in the same change, or keep it as the table-level override of the class default (a build table may want another aura than its class); prefer dropping it, because nothing reads it.

## Go change (small)
1. `internal/d2/classtraits.go`: add `MercAura map[string]string `json:"mercAura"`` to `ClassTraits`; a `MercAuraFor(c Class, diff string) (string, bool)` helper next to `TraitsOf`.
2. `internal/town/merc.go`: `HirePolicy(diff, merc, gold)` gains the class (`HirePolicy(class d2.Class, diff, merc, gold)`); after the difficulty default it overrides `w.Aura` when `MercAuraFor` has a row. The gold floor stays by difficulty. `cmd/d2bot/merc_adapter.go` passes `s.Player.Class`.
3. `internal/task/act2.go` `mercNodes` and `quest.Act2Merc` stop hard-coding `Defiance`: the hire step takes the aura from the same helper (the `MercHirer.HireMerc` signature already carries the aura string).
4. Tests, two classes: a Necromancer row changes the Normal aura and leaves Nightmare at `Might`; a Sorceress (no `mercAura`) keeps today's `Defiance`/`Might`/`Defiance`; an aura no row offers returns an error that names class, difficulty and aura.
5. `mercRow` selection (`town.mercAura`, the row to click) already compares by aura name, so no change is needed there (V: `strings.EqualFold(a, w.Aura)`).

## Not covered
Whether the hire window exposes the aura of each row in the same text on every difficulty is U (docs/town.md says only the type text shows); a live check at the hire screen decides whether `Prayer` is clickable by name or needs the `Row` index. The act 1 Rogue (`mercEarlyRogue`) has no aura choice.
