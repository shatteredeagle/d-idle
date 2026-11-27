# faultsoak (TASK-979, P12) design notes, from the quest drill agent

> status: runner side built (2026-11-26, card faultsoak); fork verbs are a proposal for the owner; nothing here was run live. Tags: V verified in code or a run, U unverified.

Oracle: the invariant monitor (`docs/invariants.md`) through `cmd/d2scenario` (`cmd/d2scenario/invariants.json`: hard rows turn a run RED, soft rows are logged). Every injection is a lab verb or a runner-side wrapper, goes through the ledger (ADR 0014 DIAGNOSIS), happens only in a soak row (never a measured run) and is time-stamped in the result so a hit can be tied to its fault.

| fault | hook today | missing |
|---|---|---|
| zero-life reload (the live Questy bug: reload at life 0) | V `debug stat` life = 0 on the standing hero (setup verb of `internal/scenario/setup.go`); expected invariant `life-zero-alive` or `potion-not-drunk`, fixed by qd26 | none; one row per class |
| empty belt / zero purse | V `debug stat` gold, belt items by setup (`pre.belt`, `pre.items`), `debug paygold` | a mid-run verb that removes belt items (U: `debug itemat` only reads) |
| drop life mid-fight | V `debug stat` life, `debug mon set`, `debug spawn` crowds | none |
| missing NPC | V `debug kill` on an NPC class (U: whether the unit leaves the table, not only dies) | a clean `debug unit remove <class>` (fork) |
| stale portal | V `debug portals` reads portals | a close/expire verb (fork); a bot-side stand-in is to take the portal after `debug goto` away |
| failed portal cast | V setup without scroll or tome (as the Smith row) | none |
| control socket timeout (14 in the quest log) | none | a runner-side proxy on the unix socket (`d2dctl.Options` wrapper) that delays or drops one reply per N; lives in `cmd/d2scenario`, no fork work |
| cancel storm | V `d2dctl` cancel paths | a driver that sends cancel on a timer; same proxy |
| dropped packets | none | not a lab verb in the fork (U); skip, the proxy covers what the bot sees |

Rows: one scenario per fault, `soak.fault.<name>`, class Paladin and Sorceress, report-only until a night run shows no flakes (`-repeat 3`); each hit becomes an incident scenario (`tools/provingground/incidents/`). Order: zero-life reload first (a hook and a fix exist), then belt, NPC, proxy faults.

## Built runner-side (2026-11-26, card faultsoak, TASK-979)

All of it needs no fork work. Offline tests only (fake game socket, fake clock); not run against a live d2d (U).

- `internal/faultproxy`: a line proxy for the d2d control socket. One command, one reply ended by the `ok`/`err` status line (V from `internal/d2dctl/client.go` readReply). Faults (`Spec`): `delay_reply_every_ms` + `delay_reply_ms` (a timeout for the bot's client), `drop_reply_every_ms` (command reaches the game, reply is swallowed), `delay_input_ms` for the verbs in `input_verbs`, `cancel_every_ms` + `cancel_line` (cancel storm; the line is sent between commands and its reply swallowed). Every injection is a time-stamped `Event`.
- Rates are in time since card faultstall (2026-11-27). The first sets counted commands (a held reply every 40, 20 s each); the bot sends about ten reads per snapshot, so it waited most of the wall time, one runner step outlived the 10 min d2bot watchdog and no minute line was logged (V row logs of the feature soak, docs/probes/feature-soak-2026-11-26.md 3.4: 27 holds of 20 s in 660 s, virtual clock 3 s). The proxy also kept a held reply's upstream (and its lock) open after the bot hung up, so the bot's redial met d2d's `err busy` (V offline, `TestHeldReplyReleasedWhenTheBotHangsUp`); a hold now ends when the bot hangs up. The bot side had no unbounded call: the runner waited inside `Prefetch` under the 20 s client deadline (V goroutine dump).
- `cmd/d2scenario/faults.json`: the named sets (`socket-timeouts`, `socket-drops`, `slow-input`, `cancel-storm`, `mixed`). `cancel_line` is `cmd stop`: U, the game's real cancel verb was not checked; change the data when a lab run shows another.
- Scenario data: `faults: "<set>"` on a row (field start only; `d2scenario validate` checks the name). Rows: `tools/provingground/scenarios/faultsoak.json`, ids `soak.fault.*`, tier nightly, Paladin and Sorceress, in `invariants.json` `report_only` so hard hits are listed, never red.
- Runner: in attach (field) runs `runField` starts the proxy in front of the game socket and gives the bot the proxy path; the observer after the run reads the game directly. The result carries `faults` (set, counts by kind, path of the JSON-lines event log). Env `D2SCEN_FAULTS` overrides the row's set.
- Soak driver: `d2scenario soak -id soak.fault. [-faults SET] -duration 1h [-class C] [-seed N] [-dry-run] [-strict]`. Runs the rows in turn through the slot wrapper, again with the next seed after each pass, until the duration is used (the run in progress is finished). Writes `soak.json` and prints invariant hits (hard and soft by `invariants.json`), injections by kind and survival (not BLOCKED, life above zero at the end, no `life-zero-alive`). Report-only; `-strict` exits 3 on a hard hit.
- Oracle: P1 monitor lines (`invariant <name>`) through `invariantCounts`; injections are never an assertion. Each hard hit becomes an incident scenario by hand (`tools/provingground/incidents/`).
- Writes: the proxy writes no game state. The row setup is the usual SETUP (CHEATED) writes; a faulty channel is never a gate, baseline or measured run (ADR 0014).
- Limits (U): a managed (non-attach) run starts its own d2d at the socket path, so faults cover field-start rows only. The d2d allows one control client, so the cancel storm goes through the proxy's own upstream connection and is serialised with the bot's commands, not concurrent with a command in flight.

## Fork verbs still needed: a proposal for the owner

Each is a new lab verb of the private fork (needs the owner's go, ADR 0014 DIAGNOSIS: lab only, ledgered, never in a measured run).

1. `debug unit remove <class|id>`: take an NPC out of the unit table (not a kill). Tests "the NPC the quest needs is gone" (the bot must not loop on it). Today `debug kill` on an NPC is the nearest stand-in (U: whether the unit leaves the table).
2. `debug portal expire [n]` (or `close`): expire or remove an open town portal while the hero is in the field. Stand-in today: take the portal away by `debug goto`.
3. `debug belt clear`: empty the belt in the middle of a run (today only SETUP can place belt items). Could be `debug item remove <id>` generalised.
4. Optional: `debug purse set <gold>` mid-run; `debug stat` gold already does it, so only a doc line.
Dropped packets are not a fork verb; the proxy covers what the bot sees.
