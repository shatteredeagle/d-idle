# Risks and what would make us slip

Refreshed 2026-10-28 (MILESTONES2, master abf1195). The full register is docs/review/roadmap.md (R1-R14). This list is the subset that moves milestone dates, ranked by how likely
it is to cost a week before alpha. Likelihood and impact are judgement (U).

| # | risk | hits | likelihood | what we do | trigger to act |
|---|---|---|---|---|---|
| S1 | **Managed mode under a long soak**: proven for 90+ minutes without setcap (docs/launch.md 9, flock leak fixed), but d2cheat still needs the setcap route next to a managed bot, and an orphan game after SIGKILL is unreadable | M0, G1, F | low-med (was med) | keep one managed run per day; cheat only with `--nolock` tools; never SIGKILL the bot | a managed run fails twice for a new reason |
| S2 | **Long-run failures we have not seen yet**: longest runs are 90+ and 31 minutes (campaign), no 2 h report; the last campaign run had 13 rung-3 events from one cause | A, G1 | high | start the 2 h soak now (managed is done); triage by cause; report recovery rungs per hour (G1.rungs) | first soak report |
| S3 | **Early economy**: the 31 minute run needed a 500 gold cheat for the tome; earned-gold behaviour (repair, starvation) never seen | A | high | A-2 package; no cheats inside soaks | abandoned trips in the first soak |
| S4 | **the worker host is shared and loaded** (load 30-40, about one core per instance, the user's desktop): timing knobs flake, soaks are invalidated by reboots or heavy jobs, few instances free | all | high | cap live agents at 3-4; one instance reserved for soaks; record load in every report; a second x86 host for soaks once F-1/F-2 work (the container on the NAS; ask first, U) | ladder rate rises with load |
| S5 | **Task-layer sprawl**: internal/task exists and is sound offline; the risk moved to the adapters (boss.Runner, questmem, dialogs) and to Acts 2-5 handlers | B, C | med | adapters as data; reviewer look at the boss adapter; no C1 code before B-8 | an adapter > 200 lines or a handler with its own state machine |
| S6 | **Quest/dialog blindness**: Den and Akara rows and the in-memory source are live; real completion words and Kashya/Warriv/Charsi/Cain rows are not | B, C | med | proxies + save at menus; recorded row indices + confirm by effect | V-01 inconclusive |
| S7 | **No CI runner**: still none; 300 commits on master, whole-repo tests are not run on the head box (4 GB) | all | high (until the token) | user registers a forge runner on the worker host; until then the main agent runs tools/ci/check.sh + race tests on the worker host before each merge | any red master |
| S8 | **Soak wall time**: 24 h and 72 h runs block an instance and cannot be sped up | G1, G2, H | certain | plan the soak instance a week ahead; never run two soaks on one instance; partial reports are still useful | soak queue > 2 days |
| S9 | **Boss difficulty at plan levels** (Andariel at clvl 15-20 with Act 1 gear; Duriel; cold immunes in Act 3+) | B, C1 | med | planner level floors; potions; merc early; measure deaths per boss | > 2 deaths per boss attempt |
| S10 | **Save loss** (saves still not wired into start/stop): a kill during a write corrupts a save, or a kill loses an hour of progress | G1 | med | M0.saves + G-2 save cycle before the 24 h soak | first corrupt save |
| S11 | **Merge churn and coordinator context** (many branches, one main agent) | all | med | consumer-side interfaces, disjoint directories, small merges; this file and milestones.json as the index | > 2 conflicts per day |
| S12 | **Scope creep into the game layer and the showcase** before the gate (UI, prestige, parties, stream tooling: 7 stream/public branches merged in one day) | beta, H | high | L1-L3 and E stay behind H; S gates nothing and must not take a worker host instance from a soak | a stream/UI branch while A is below 100 % |
| S13 | **User-side items** (runner token, the worker host firewall, NAS deploy, a friend to test F) stall packages | M0, F | med | list them on the hub "Things waiting on you"; agents do not wait on them | item older than 3 days |
| S14 | **Boss scripts are guesses**: every number in docs/boss.md is U, wiring is unbuilt, and Blood Raven already beat clvl 6 | B | high | wire, then run docs/boss.md section 5 on cheated levels before any unattended clear; raise level floors | first boss attempt with > 1 death |
| S15 | **Second class debt**: combat is class-agnostic (live), but leveler, town, merc, cheat allow-list and infra were Sorceress-shaped (docs/classes/smoke-paladin.md) | H | med | do the Paladin after the first Normal Act 1 clear, as H.class_* | a class change breaks Sorceress soaks |

## Retired or reduced since the first version
- Party/loopback (old E risk): retired, the join works (docs/net/multiplayer.md 9). fd growth: not a leak (docs/leakhunt.md). Helper deaths: fixed, 90 minutes clean.
  Managed mode: proven. Nav seed dependence and dungeon exits: fixed (docs/nav.md).

## What would make alpha slip (in order of likelihood)
1. The first 2 h soak shows a new class of failure every run (S2) and A takes 3-4 iterations instead of 2; or the earned-gold economy (S3) does not close.
2. The worker host capacity: fewer than 3 instances free for live work for several days (S4).
3. Boss scripts need more than one rewrite after the first live checks (S14), or Andariel needs clvl 22+ (S9).
4. A 24 h managed run exposes a leak the 45 minute soak could not (S1).
5. The 24 h soak has to be restarted more than twice (S8): each restart costs a calendar day.
