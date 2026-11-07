# Currencies: registry, ledger and the bank operation

> status: implemented offline, 2026-11-06, TASK-62. NOT verified live: nothing here has met a real game. The bot-side hook that observes the pack (job B, TASK-64) and the overlay rows (TASK-63) are separate. Design and reasons: [research/metacurrency-design.md](research/metacurrency-design.md); ADR 0016 (Amendment 2026-11-06); terms in GLOSSARY (Mark, Tally, Bank, Registry).
> Tags: **V** verified by a test in this repo (named); **U** unverified.

Code: `internal/currency` (registry, ledger, bank, wallet) and `internal/watch/currency.go` (the screen). Off in vanilla mode: no ledger file is ever created and `/api/currency` answers 404.

## Registry (data)

`internal/currency/registry.json`, embedded; `currency.Load(path)` reads another file. Schema `currency.registry/1`; unknown JSON fields are an error. Four lists:

| list | row fields |
|---|---|
| `currencies` | `id`, `scope` (`per_character` or `account`), `name`, `blurb`, `storage` (`item` or `ledger`), `item` (item storage only: `code`, `art_donor`; the code stays empty until the overlay row exists), `stackable`, `cap` (stack cap, item currencies), `sources`, `sinks`, `converts_to` (`currency`, `ratio`, `via` = the bank recipe id), `display` (`icon`, `label` after the amount, `order`, `where`: `header`, `spend`) |
| `sources` | `id`, `currency`, `name`, `daily_cap` (soft cap per UTC day, 0 = none) |
| `upgrades` | `id`, `sink`, `currency`, `tier`, `name`, `effect`, `costs` (cost of level 1, 2, ...), `wired` (false: listed, cannot be bought yet) |

Validation (V `registry_test.go`): unknown scope or storage, duplicate id, a `converts_to` cycle or a bad ratio, a stackable currency without a cap, a cap on a non-stackable one, an item currency that is not per character (and a ledger one that is not account scoped), a source of the wrong currency, an upgrade whose sink the currency does not have. Every message names the row. No silent defaults.

Add a currency: add a row (and its sources, and upgrades if it is spent); a ledger currency needs no code. An item currency also needs its overlay rows (TASK-63).

## Ledger

Directory per profile (the bot uses `<instance state>/currency`, env `D2BOT_CURRENCY_DIR` overrides): `ledger.jsonl`, `ledger.head`, `ledger.lock`.

- One JSON object per line. Field order is the canonical order (`seq`, `ts`, `prev`, `kind`, ...). `prev` is the hex sha256 of the previous line's bytes (`genesis` for the first). A line that does not re-encode to itself is corrupt (V).
- `ledger.head` holds `{seq, hash}` of the last entry, written (atomic rename) after the line is fsynced. It makes a cut tail detectable: the file shorter than the head, or a rewritten entry, fails. A head one entry behind is a crash between the two writes and is fine (V `TestVerifyDetectsTamperAndTruncation`, `TestHeadBehindByOneIsFine`). Limit: an adversary who rewrites the file AND the head is not detected; the threat is bot bugs and silent corruption.
- Writes run under `flock` on `ledger.lock` (unix; a lock file elsewhere), each op re-reads and verifies the whole file, appends one line, fsyncs. V with 8 goroutines and 4 processes (`TestConcurrentAppendsStayChained`, `TestConcurrentProcesses`).
- A torn last line (crash mid-write) makes `Verify` fail; the next write drops it and `Repaired()` says so (V).
- Entries carry `mode` (the `overlay.Mode` stamp), `instance`, `character`, `run`.

Kinds: `pending` (write-ahead), `commit` (observed credit), `void` (pack unchanged), `refused` (cheated), `credit`, `grant` (one-offs), `spend`. Only commit, credit, grant and spend change a balance.

Rules of `Append` (V `ledger_test.go`): vanilla mode refuses everything (`ErrVanilla`); a credit names a ledger currency and a source of that currency; a key that is already booked returns the earlier entry with `ErrDuplicate`, never a second credit; the part of a credit above the source's `daily_cap` is booked as `sealed` (kept, not spendable); a spend that exceeds the spendable balance is refused, not clamped; `Spend` takes the next level's cost from the upgrade row. `Balance(currency)` replays: spendable = credited - sealed - spent.

## Bank operation

`mark x n` in the pack becomes `n x ratio` tally. Credit is the OBSERVED pack loss, never the intended number.

1. `BeginBank(ctx, req, obs)`: refuses in vanilla mode, for an unknown recipe, a cheated run, an empty pack; reads the pack count (`before`); appends `pending{key, before}`. A key already closed answers from the ledger; a key still pending resumes.
2. The caller transmutes the cube recipe (job B).
3. `CommitBank(ctx, ticket, obs)`: reads the pack again (`after`); `after == before` appends `void`; a run that became cheated appends `refused` and returns `ErrCheated`; else appends `commit{amount = (before-after)*ratio, ref = pending seq}`.
4. `Bank(ctx, req, obs, act)` does all three; the commit runs even if `act` fails (the pack decides).
5. `Recover(ctx, obs)` at start settles pendings left by a crash: credit = loss since the pending entry, never more than it recorded (V `TestCrashBetweenPendingAndCommitRecovers`).

`BankRequest.Key` is the idempotency key of the observation (for example `<character>/<n>`). Replaying it, even with a different pack, credits nothing (V `TestBankReplayNeverDoubleCredits`).

### Observer (for the bot hook)

```go
type Observer interface {
    PackCount(ctx context.Context, character, currency string) (int, error)
    Cheated(ctx context.Context, character, run string) (bool, error)
}
```

`PackCount` is a read of the hero's pack (not the stash). `Cheated` is true when the run has gameplay writes or the stack came from a setup write: back it with `internal/cheat` `Ledger.CheatedKey` since the run start. `currency.Fake` is the test double. The bot also calls `Wallet.SetCarried(character, "mark", n)` after each inventory read so the screen can show marks in the pack.

## Watch screen

`GET /api/currency` (schema `watch.currency/1`, read-only, behind the same viewer token rules): `enabled`, `mode`, `text`, `strip`, `error`, `currencies[]` (`id, scope, name, blurb, icon, label, order, where, amount, sealed, cap, storage`; `amount` is null until known) and `spend[]` (`id, name, effect, tier, currency, level, maxLevel, cost, affordable, enabled, why`). Vanilla: `enabled` false, text "Currencies are off in vanilla mode." A ledger that fails `Verify` shows an `error` line and no balances. Page: a strip in the header and a Spend tab with one row per upgrade; all text through `textContent`, no inline script or style, no URL. `Buy` is disabled unless the upgrade is `wired` and affordable; nothing is wired in this slice and there is no write route. Golden files: `internal/watch/testdata/currency-*.json`.

Wiring: `d2bot --backend d2d --overlay-dir DIR --ladder --watch` opens the wallet; without `--overlay-dir` no currency code runs and the config does not announce it. `ladderRig.Wallet` is where job B hooks in.

## Not done / unverified

- U: the real cube recipe, the overlay Misc code and stack roll (TASK-63); the pack read in the game (TASK-64).
- U: rates and prices in `registry.json` are the design's guesses.
- Not wired: any real upgrade; the spend POST route.
