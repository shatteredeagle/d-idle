# Advisor R14: runeword and cube advice (TASK-991)

> status: built dark (knob `items.advisor`, default 0). Advice lines only.

Rule (owner): inventory management beyond starter gear and potions is player-driven. The advisor therefore only SUGGESTS; it never moves, sockets or cubes an item. Everything it knows is data, so a class or recipe is a row, not code.

## Parts
- `internal/advisor/advisor.json`: runeword rows (rune names in socket order, base type codes, classes, rank, `ladder`, `merc`). Rune lists V (docs/items.md section 5, from runes.txt). Which class prefers which, and the ranks, are U (SME review).
- `advisor.Advise(Stock, Options)`: the matcher. A recipe is advised when its class list is empty or names the class, and (if `ladder`) only with `Options.Ladder` (= `items.ladderRunewords`, ADR 0024). An empty class is an error, never a default class.
- Kinds, best first: `make` (runes and a white, empty base with exactly the rune count of sockets are in hand), `base` (runes in hand, no base), `cube` (the runes can be had from spare runes; steps from `cubejobs.RuneNeed`, never consuming runes the target needs, rune cap 13 vanilla, 33 with the ladder switch since Dol to Hel and above are ladder-flagged, docs/items.md Q1 U), `near` (one rune short). Within a kind, recipes that name the class come before general ones, then by rank.
- `Stock.AddVault`, `Stock.Add`: gather pack, stash and vault into one stock; each base remembers where it is so the line can say "in the stash".
- `watch.AdviceSection` (`advice`, "Worth making"): a Return screen section with lines and no buttons; `FunConfig.Advice == nil` leaves it off.

## Not done (follow-ups)
- Bot side: build the Stock from the hero snapshot, stash and vault store, read `items.advisor`, set `FunConfig.Advice` (cmd/d2bot/funwire.go). Needs the live inventory feed; tests use fixtures only.
- Shrines and monster aura or curse status reflexes (the other halves of R14) are not in this slice; they are behaviour, not advice, and need an owner go.
- SME review of the lines per class (the enable criterion), then the ledger row `items.advisor` can move to trial.
