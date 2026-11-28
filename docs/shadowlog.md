# Shadow log digest (`d2shadow`)

> status: current (card road4, TASK-1121, 2026-11-28). Offline tool; nothing here ran against a live game or a sandbox log. Tags: V verified (how), S secondary, U unverified.

Four dark-launch rows name shadow log lines as their enable evidence ("a night of `admission shadow` lines, differ count reviewed", "shadow lines replayed and reviewed", "mode 1 shadow on a sandbox night shows no disagreement", "stage 1 replay vs pickit: no worse"). Until now nothing read those lines out of a recorded bot log; the counts were copied by hand from `ops status --why` or an API page. `d2shadow` reads one or more bot logs (JSON lines, the format `cmd/d2bot` writes) and prints one digest and one verdict per family.

```
go run ./cmd/d2shadow [-json] [-strict] [-rows N] LOGFILE...
```

- `-strict`: exit 1 when a family's verdict is `review` or `worse` (exit 2: usage or an unreadable file).
- `-json`: the report as `shadowlog/1` (`internal/shadowlog.Report`).
- `-rows N`: disagreement rows per family in the text output (default 10, 0 = all).

To use it on a sandbox night: copy the log out read-only (the way `docs/probes/evidence-logs-2026-11-28.md` does), run `d2shadow -strict LOG`, and paste the family block into the ledger row's note. The log copy is never committed.

## Families

| family | knob whose shadow writes it | line (source, V read in code) | what the digest groups | verdicts |
|---|---|---|---|---|
| `admission` | `bot.admission` | `admission shadow` with `differ`, `area`, `purpose`, `old`, `new`, `row`, `need` (`internal/bot/admission.go` `admitNote`) | totals per cell (`new-only`, `old-only`, `words`) and per purpose; one row per case (cell, purpose, row, old words, new words), areas folded | `no-lines`, `review` |
| `planner` | `agency.planner` | `planner shadow: clvl= class= plan= profile= agree= best= aura= auraProfile= arch= unplayed= warnings= via=` (`internal/agency/planner.go` `ShadowLine`) | lines and agreeing lines per class, warning codes; one row per differing (class, attack, profile attack, aura, profile aura), the attack's level dropped | `no-lines`, `clean`, `review` |
| `upkeep` | `combat.deliveryUpkeep` (mode 1) | `combat upkeep shadow` with `skill`, `what`, `old`, `driver` (`internal/combat/upkeep.go` `upkeepShadow`) | totals per kind and per skill; one row per case; the why line says how many cases were seen once and how many again | `no-lines`, `review` |
| `loot` | `loot.rulesEngine` (shadow `bot.lootShadow`) | `loot shadow: <scope> <item> today= rules= rule= reason=` (`internal/loot/lootshadow.go`) | totals per scope; one row per (scope, today, rules, rule) with an example item | `no-lines`, `review`, `worse` |

Rules a reader must know:

- `no-lines` is not `clean`. The admission, upkeep and loot shadows log only when they disagree, so an empty family proves agreement only if the shadow ran: check the knob on the sandbox, and for admission the agree count in `ops status --why`. Only the planner writes a line when it agrees, so only it can be `clean`.
- `worse` (loot): at least one `ground` line where today's code picks the item up and the rules do not. That is the ledger's "no worse on recorded drops" read literally for stage 1. The other direction (rules pick up more) and the tidy and sell scopes are `review`.
- The loot lines are deduplicated per scope, area, item code and quality and capped at 20 a minute by the bot, and the admission lines are logged once per case: the counts are a floor on how often a case happened, exact on which cases happened. The full loot counters stay at `GET /api/lootrules/shadow`.
- Upkeep "seen once" against "seen again": the bot logs a lasting disagreement once, so the same case a second time is a new episode. A case seen once per skill fits the driver learning a state id (the ledger's "state learning lines"); which cases those are is the reviewer's call (U: the ledger does not define the term, the tool does not guess it).
- Folded repeats (`last line repeated`, `internal/logx.Dedup`) are counted with their `times`.
- A line of a family that does not parse is counted as `unparsed` and forces `review`: the reader is behind the bot's line format. The planner test builds its lines with `agency.ShadowLine`, and the loot test compares the scope and action words with `internal/loot` and `internal/lootrules`, so a format change fails in CI first (V `go test -p 1 ./internal/shadowlog`). The admission and upkeep attribute names are copied from the two call sites (S: no shared constant).
- The digest has no class or build rule of its own: classes, skills and item names are whatever the lines carry. The tests use two classes (Paladin and Necromancer planner lines, a Sorceress and a Barbarian upkeep skill).

## Not covered

- `arbiter shadow` lines: `d2arbiter` and `d2scenario arbiter-replay` own those (docs/arbiter.md).
- Whether a disagreement is right. The tool sorts and counts; the review is a person's or the SME agent's.
- No sandbox log has been run through it yet (U). First use: one recorded night per sandbox, cited in the rows `bot.admission`, `agency.planner`, `combat.deliveryUpkeep` and `loot.rulesEngine` of the dark-launch ledger.
