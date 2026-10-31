# Open findings (merged from the four merge-risk reviews)

One table for review-overnight, review-day, review-evening and review-night (bodies in [archive/](archive/): [overnight](archive/review-overnight.md), [day](archive/review-day.md), [evening](archive/review-evening.md), [night](archive/review-night.md)). Built 2026-10-29 by ARCHIVE from the reviews, INDEX section 4 and `git log` (master 8689811).
Evidence tags: V = commit message or code read now says so; S = inferred from the log or code shape, not re-tested; U = no fixing commit found.
State: **fixed** = merged with tests or a code read; **open** = no fix found; **unverified-live** = fixed in code but the live behaviour was never run (the review's failure scenario is not yet disproved on a real game). No live run was done for this table. The strategy review's G1 soak (docs/review-strategy.md) is the single live check that covers most "soak" rows.

Review ids: OV = overnight, DY = day, EV = evening, NT = night.

| id | sev | finding | state | fixing commit | live check needed |
|---|---|---|---|---|---|
| OV-H1 | High | restart and ladder budgets reset by gold, potions, damage | unverified-live | 0162995 (merge ffecf16, harden) | soak G1 with injected Game.exe kill |
| OV-H2 | High | rungs 3/4 run under an 8 s guard that cancels menu re-entry | unverified-live | 0162995 | same |
| OV-H3 | High | executor counters never reset, `givenUp` permanent, misses timeouts and deaths | fixed (S) | e4ec53b | none |
| OV-M1 | Med | one failed unplanned restart fatal; rung-4 timeout shorter than recycle | unverified-live | harden (ffecf16) | soak G1 recycle drill |
| OV-M2 | Med | stale god marker restored without checking inflated life | fixed (S) | harden (ffecf16) | none |
| OV-M3 | Med | GuardSave not wired into d2bot | fixed (V: cmd/d2bot/saveguard_test.go) | 095a782 | none |
| OV-M4 | Med | planner floor counts avoided areas, wrong fallback message | fixed (S) | harden (ffecf16) | none |
| OV-M5 | Med | flee has no exit condition | fixed (S) | 61712bc | fight with no potions |
| OV-L1 | Low | data race on `launch.Recycler` counters | fixed (V: mutex in recycle.go) | 45e98f1 | `-race` on the worker host (not run) |
| OV-L2 | Low | nested ladder: errStuck inside a rung re-escalates | open (U) | none found | none |
| OV-L3 | Low | `starveArea` without TP picks Act 1 plan step | open (U) | none found | none |
| OV-L4 | Low | `expectGameExit(false)` can swallow a relaunch death, no test | open (U) | none found | kill game during Restart |
| DY-H1 | High | Nightmare/Hell campaign marked Done in a Normal game | fixed (S) | dc80f4c (merge 0a16ecf, harden2) | none |
| DY-H2 | High | difficulty switch tried once per clvl, permanent pause | unverified-live | 8f2f212, f407231, 0a16ecf | reach the switch level in a campaign |
| DY-H3 | High | chicken exits reset the death count on every new game | unverified-live (strikes now cross games; does it still bound a Diablo/Baal loop? U) | dc80f4c | boss-chicken loop at Andariel/Duriel |
| DY-H4 | High | fleet SIGKILLs d2bot after 10 s, shutdown needs about 50 s | unverified-live | 7d0a0ae (merge d05f560, 70 s grace), cd0895f | fleet stop with a running goal |
| DY-H5 | High | goals marked seeded before d2bot stored them | fixed (S) | 7d0a0ae, d05f560; reseed when store lost: 9fe2f89 | none |
| DY-M1 | Med | fleet `watch` treats one missing poll as game death | fixed (S) | 7d0a0ae (game-gap grace) | recycle under d2fleet |
| DY-M2 | Med | `CleanGame` also kills Xvfb and helper | fixed (S) | 7d0a0ae (StopGame) | fleet restart of a slot |
| DY-M3 | Med | Normal templates do not pin difficulty | fixed (S) | harden2 (0a16ecf) | none |
| DY-M4, M5 | Med | Iron Maiden retreat only drinks; weak-pack uses base level | fixed (S) | e515f74 (bossfix) | none |
| DY-M6 | Med | placeItem can leave quest item on cursor | fixed (S) | harden2 (0a16ecf) | Act 2 quest item |
| DY-M7 | Med | step overrunning `r.do` retries forever | fixed (S) | harden2 (0a16ecf) | none |
| DY-M8 | Med | only ladder pause auto-resumes | fixed (S) | harden2 (0a16ecf), 8dd398a | none |
| DY-M9 | Med | d2bot under fleet has no `D2BOT_INSTANCE` | fixed (V: fleet/bot.go env) | 7d0a0ae | none |
| DY-L1 | Low | 10 MB `cmd/d2soak/d2soak` binary committed | fixed (V: not tracked) | 7d0a0ae | none |
| DY-L2 | Low | chain: `NextIDs` unset if second add fails | unverified (code shows append after each add, S) | none found | none |
| DY-L3 | Low | d2bot log rotated only at Run start | fixed (S: rotWriter) | 095a782 (agent/lows) | none |
| DY-L4 | Low | `w.WPs[cur] \|\| InTown && w.WPs[cur]` redundant | fixed (S: line gone) | none found | none |
| DY-L5 | Low | Anya pseudo item decided after the click | open (U) | none found | Anya thaw |
| DY-L6 | Low | Hell floor 75 for every Act 1 quest | open (U) | none found | none |
| DY-L7 | Low | save guard logs a failed ledger line for real characters | fixed (V) | 61712bc, 095a782 | none |
| EV-H1 | High | container SIGKILLs d2fleet 8 s after SIGTERM, no Save and Exit | unverified-live | cd0895f (merge e42e15f, stopchain) | `docker stop` with a game |
| EV-H2 | High | d2bot worst-case SIGTERM path (about 120 s) exceeds fleet grace | unverified-live | cd0895f | same |
| EV-M3, M4 | Med | mana-burn chicken on low mana; hardTick pre-empts survival | fixed (S) | 5268b87 (hellfix) | Hell fight |
| EV-M5, M6 | Med | winereap no-op on new layout; kill safety | unverified-live | ac12432 (reapfix) | `tools/game.sh reap-all --dry-run` vs `ps` |
| EV-M7, M9 | Med | `Fleet.Close` sequential; Save and Exit races cancelled runner | unverified-live | cd0895f | fleet stop with 2 slots |
| EV-M8 | Med | farm run selection gaps | fixed partially (unlocks, clvl floor) | 5dfe9e6 | farm run |
| EV-M10 | Med | resist penalty may be applied twice (rules.go, hard.go, d2/player.go) | open (U) | none found | compare Hell resist gate with sheet resists |
| EV-L11 | Low | farm re-entry restarts count and FirstHere | fixed (S) | e2d1886 | none |
| EV-L12 | Low | antidote every 3 s, no duration model | fixed (S: antidoteDuration 30 s) | none found | none |
| EV-L13 | Low | setup token in URL; root-owned bind mount loop | open (U) | none found | none |
| EV-L14 | Low | reaper can steal d2fleet exit code | fixed (S) | 9fe2f89 | none |
| EV-L15 | Low | `soak24.sh --for 1h30m` arithmetic; dead WineMaxGrowth24h | fixed for `--for` (V: bash test); dead limit open | 421c511 | none |
| EV-L16 | Low | goals-seeded marker without store; rotWriter reopen | fixed (S) | 9fe2f89 | none |
| NT-H1 | High | stale `m.locked` never cleared on resume/remove | fixed (V: test) | c4898ab (merge 3b6f95d, goalhold) | none |
| NT-H2 | High | supervised restart re-sends `--goals campaign` | fixed (V) | 9ec3afc (merge 49ca1f2), 5e7e1cb (seeded adds) | supervised crash drill (`crashdrill.sh`, not run) |
| NT-H3 | High | `D2BOT_SUPERVISE` env makes children supervisors | fixed (V: test) | 9ec3afc, b70bf04 | none |
| NT-M1 | Med | panic with no goal never exits for the supervisor | fixed (S) | 9ec3afc | crash drill |
| NT-M2 | Med | supervisor restarts deliberate fatals with `--autostart` | fixed (S) | 9ec3afc | crash drill |
| NT-M3 | Med | goSafe gives up and leaves a zombie | fixed (S) | 9ec3afc | crash drill |
| NT-M4, M5 | Med | recover over Lock/Unlock deadlock; guard panic leaves action unguarded | fixed (S) | 9ec3afc | none |
| NT-M6 | Med | clean-start wait for Wine game | unverified-live | b5e234a | managed start with a stale game |
| NT-M7 | Med | `Goal.HeldAt` dead state | fixed: legacy-only field, read once to migrate old files, never written (V: goal.go:213, manager.go:126) | c4898ab | none |
| NT-L1, L3 | Low | Hold level stale; repeat quest events save the file | fixed (S) | a8f7ee9 | none |
| NT-L2 | Low | two producers on `r.ch`, drop-oldest could drop a Hold | open, accepted (S) | none | none |
| NT-L4 | Low | pollQuests runs under obMu | open, accepted (V) | none | none |
| NT-L5, L6 | Low | orphaned child if supervisor SIGKILLed; fleet nests supervisors | fixed (S) | b70bf04 | none |
| NT-L7 | Low | doctor nits (symlink mode, `/checks` exposes paths) | open (U) | none found | none |
| NT-L8, L9 | Low | LOOTPOTION build fallback; SEALSIM dropped error | open, accepted (V) | none | none |

## From the SME research (docs/sme/play-*.md, 2026-10-29; claims tagged V/S/U there, verify live before acting)
- Offline characters keep only their OLDEST corpse (Arreat): deathspiral retrieval logic assumes otherwise. Test with two deliberate deaths on a cheat char.
- Our entry levels are later than human norms (Nightmare 36 vs ~26-35, Andariel 22 vs ~15-18): possible XP/h win, gated by readiness checks.
- Blood Raven is optional for a human; she cost 181 aborted runs in earlier soaks.
- Elite affix handling cannot fire: the monster catalog does not read modifiers yet.
- Ladder-only runewords (Spirit, Insight, Fortitude, Infinity...) are blocked offline (builds-gear.md): test Spirit in a 4-socket sword; Grief and Voice of Reason unconfirmed. docs/items.md Q1 is wrong.
- Note: play-bot-pitfalls.md says G1 attempt 6 'sat on 3779 gold with an empty belt'; SHOPTP found 50 purchases in that log (the real defect was the gear gate, see g1-log.md).

## Added 2026-10-30 (DOCSYNC; ranked, details in [BACKLOG.md](BACKLOG.md) "OPEN, ranked")
These are not review rows. Evidence tags as above; none was re-run live by DOCSYNC.
1. Kill-injection recovery unverified since G1 attempt 8: the permission classifier denied the kill (g1-log attempt 15). OV-H1, OV-H2, OV-M1 and the stop-chain rows stay "unverified-live".
2. Nav stuck at Burial Grounds (5459,4549): root cause unproven (docs/nav.md); mitigated by NAVSTUCK only.
3. Mephisto quit-safety: d2d and the user's report disagree; needs a probe on the original game.
4. Act 2 and later never run live by the bot; Andariel never fought.
5. Free-Rogue claim is a stub until a backend/reader exists; the Monastery route needs a merc or clvl 20.
6. `potionsDrunk` counter: fixed offline in ROUTEGUARD, not confirmed on a live log.
7. Everything learned from d2d is V-d2d (reimplementation), not original-game fact.
8. Rows closed since the table was built: none re-verified here; the table is unchanged.
