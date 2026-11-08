# Draft PR body: vendor prices follow-on (fix/vendor-prices-followon-v2)

> status: draft (agent DECOMPPRICE2, 2026-10-31; branch renamed `fix/vendor-prices-followon-v2` by card prfixes, which only changed one tooling line: `zip(..., strict=True)` for ruff B905). Not posted. Proof: [pr-price2-proof.md](pr-price2-proof.md). Written for the case "open after #152 merges, rebased to the two claims plus tooling" (the recommendation in the evidence pack); if it goes into #152 instead, the two sections below become sections of #152's body.
>
> status: SUPERSEDED 2026-11-07 (card TASK-76, docs/research/fid-vendor-price-trace.md): D9 and D10 are already on upstream main (npc mult divide-first, a9548d38; repair gate, bacec633); this draft must not be opened.
>
> status: #152 MERGED 2026-11-01: rebase onto upstream main is clean (merge-tree, docs/upstreamsync.md). The follow-on itself is not posted.

---

**Title:** Vendor prices: npc mults divide first above 0xffff; no repair without a Repair type or durability

## What

Two more differences between `rules::item_price` / `rules::repair_cost` and game.exe 1.14d's `FUN_0062efb0`, one commit each, on top of #152:

1. **The npc.txt multipliers divide by 1024 first above 0xffff.** At `0x62faac` (buy), `0x62fae5` (sell), `0x62fb1d` (repair) and in the quest loop (`0x62fb70`..`0x62fc53`), a price over 0xffff becomes `(price / 1024) * mult` (signed, toward 0); at or under 0xffff it stays `price * mult / 1024`. d2d always multiplied first. New `rules::npc_mult`, used for the npc and quest mults of all three prices.
2. **No repair for an item whose ItemTypes row has no Repair, or that neither restocks nor has durability.** `FUN_0062e660` (called at `0x62efee`) passes only identified, non-ethereal items with missing charges, or with ItemTypes Repair (+8) and either a restocking stack (`FUN_0062e5d0`) or durability (`FUN_00629930`: nodurability 0, a durability column, max != 0, not indestructible). d2d priced any item with a saved max durability, so worn bows (Repair 0), crossbows (nodurability 1) and throwing potions had a repair price; game.exe's is 0. New `ItemType::repair` from ItemTypes "Repair".

## Evidence (re-run on your copy)

game.exe's own `FUN_0062efb0` in the emulator (`price_oracle.py`, from #152) against `test_game --price` on the real build:

```sh
cd tools/emu
uv run python price_edge.py 120000 11 > cases.tsv
uv run python price_oracle.py game cases.tsv > g.txt
../../build/tests/test_game --price cases.tsv | grep -E '^[0-9]+ ' > d.txt
uv run python price_claims.py cases.tsv g.txt d.txt
```

`price_edge.py` aims at the two terms (armor bases around 0xffff / 0x10000 and far above; bows, crossbows, potions, rings with durability 0 / 1 / max - 1 / max / 255; every vendor and difficulty). `price_claims.py` counts mismatches on the columns each term moves (`price_model.py`, equal to game.exe on all 120,000):

| Build | divide-first cases (buy / sell / repair) | repair-gate cases (repair) | clean (buy / sell / repair) |
|---|---|---|---|
| #152 head | 46,730 / 545 / 14,356 | 15,519 | 46,730 / 545 / 28,473 |
| + npc mults | 0 / 0 / 0 | 15,519 | 0 / 0 / 14,117 |
| + repair gate | 0 / 0 / 0 | 0 | 0 / 0 / 0 |

On #152's 100,000-case set (`price_oracle.py gen 100000 7`): 9,232 / 154 / 1,608 and 5,163 -> 0, clean 0 / 0 / 0. No case that matched before differs after.

`test_rules`, `test_combat`, `test_game`, `test_protocol` green; `pr_guard.py` 0 FAIL / 0 WARN; `presubmit.py --quick` 0 FAIL (build WARN: built and tested by hand); clang-tidy (repo config) clean on our lines. `tools/emu/price_edge.py` and `price_claims.py` are new tools (CODEOWNERS review).

## Limits

- The oracle's items carry no stat list: charges, staff mods, magic stat costs and sockets are not compared. With charges missing, game.exe passes the gate for any type and adds the full base x rep mult when the item has no durability; d2d charges the charges only (`// ponytail:`).
- Quest mults: same rule read at the quest loop; the oracle's quest flags are 0, so `test_rules` covers that path.
- game.exe multiplies in 32 bits; bases are kept below 2^20 in the cases, where neither side wraps (no 1.14d price comes near).
- Whole-item repair (game.exe 1) and stack quantities stay as in #152.

Agent: Claude Code
