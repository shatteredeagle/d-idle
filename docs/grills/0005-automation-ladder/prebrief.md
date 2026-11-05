# Prebrief for grill 0005: the automation ladder (ADR 0016)

status: ready (not yet grilled). Written 2026-11-05 for the interviewee (the user) and the grilling model. How the session runs: [../../process/grill-workflow.md](../../process/grill-workflow.md), mechanism [../../process/grill-agent.md](../../process/grill-agent.md). Start with `tools/grill.sh 0005 automation-ladder`. Tags: V = read in a repo file (path given), S = veteran answer of the d2-sme agent from memory, U = unverified or judgement, D = the user's decision of 2026-11-05. Classic 1.14d; no D2R facts used.

## 1. The decision in one sentence

Turn the user's product-direction decisions of 2026-11-05 (d.idle as an incremental Diablo II: automation tiers T0 to T5 unlocked by learning from the player's answers and upgraded by sacrifice) into ADR 0016 and at most three cards, by stress-testing them until the open points have an answer or a named trigger.

This is ONE decision (the ladder as the product shape). Anything else found on the way goes under "Parked" in the ADR draft.

## 2. Sources to read first

- `docs/research/automation-ladder.md` (V, DRAFT, on main): tiers, mechanics sketch, risks, backlog re-ranking, proposed record text.
- `docs/research/automation-ladder-record-proposal.md` and `docs/research/decision-point-store-spec.md` (V on branch `agent/ladderrecord`; if they are not on main yet, read them there with `git show agent/ladderrecord:<path>`): the user's answers recorded in section 7 of the draft, proposed index line, ADR 0016, glossary terms, and the store spec. Treat their text as a proposal, not as accepted.
- [ADR 0015](../../adr/0015-goals-acceptance-on-game-exe.md) (goals G-A to G-F, "done means", the parked list) and [ADR 0009](../../adr/0009-d2d-primary-backend.md) (d2d is the primary backend, so custom objects are possible).
- [docs/decisions.md](../../decisions.md) block "Current truth" (wins over older text).
- `PLAN.md` lines 85 to 87: softcore first, hardcore later as the rebirth (prestige) mechanic, the meta-game layer PARKED, `Tuning` metadata and the data-only `goals.Upgrade` contract.
- [docs/roadmap.md](../../roadmap.md) M6 (idle layer: "defined by a grill after M5", parked), [docs/capabilities.md](../../capabilities.md) (what is proven live), [docs/cards.md](../../cards.md) (cap of 12 open cards, each mapped to one goal).

## 3. Already DECIDED by the user (do not re-ask; challenge only if the user reopens)

1. **Shape.** Game shapes ranked 4 (automation as progression) first, 1 (watch the hero) second, 2 (job runner) third. The abstract simulator (shape 3) is REJECTED: the real engine keeps running the game (D).
2. **No direct control.** No click-to-move, no casting. The player gives task orders and answers questions (D).
3. **Currency.** Anything sacrificed is currency, under ONE worth rule (a single value table for gold, gems, runes). The runes-versus-runewords tension is accepted, not solved by a second currency (D). Details of the worth rule are NOT decided (Q1, Q2).
4. **Learning.** A rule is learned when the same answer is given in the same context N times in a row, with N by risk: 2, 3 or 5 (low, medium, high). Learned rules are revocable (D). The context key is NOT decided (Q3).
5. **Tiers versus acts.** The full ladder T0 to T5 exists in Act 1 Normal first; later acts and difficulties come after (D).
6. **Start.** Every run starts from a fresh clvl 1 hero, no premade character (D).
7. **Failure surface.** Three channels: needs-input (question card, hero waits safe in town), incident (plain-language journal report, rare at high tiers), uptime (visible stat) (D).
8. **Rebirth.** A hardcore death resets the character; learned automation survives (D, from the draft). Whether altar quality survives is open (Q8).

## 4. The tiers in one table (V draft, section 2)

| Tier | Name | Code behind it today (V `automation-ladder.md`, `docs/capabilities.md`) |
|---|---|---|
| T0 | Order-taker | mostly built; soaked on Act 1 Sorceress |
| T1 | Tidy | built, proven in drills; economy tuning open |
| T2 | Runner | partial; `Runs` goal and boss farm scripts exist, offline only |
| T3 | Planner | partial; leveler and readiness rule built, one proven build |
| T4 | Questor | partial; Act 1 quests and merc exist, Andariel never cleared |
| T5 | Farmer | NOT STARTED as a goal-driven loop (drop tables for Normal Acts 1 and 2 only, MF 0) |

Missing for every tier: the ask mechanism, the learned-rule store, the altar object, the question-card UI (V draft, sections 3 and 4).

## 5. What the grill must PRODUCE

1. ADR 0016 text, status `proposed`, MADR format (Context, Considered Options, Decision Outcome, Consequences, Sources), with gates, triggers and a "Parked" list. The user flips it to accepted.
2. The one `docs/decisions.md` line (with the ADR link, existing style).
3. Glossary terms, as proposed text in `outcome.md` (the grill session does not edit `GLOSSARY.md`): at least Tier, Decision point, Learn by doing, Altar, Sacrifice, Needs-input, Incident, Uptime, Rebirth, plus any new ones (Context key, Worth rule).
4. A short list of cards, MAX 3, each with its goal G-A to G-F mapping (see Q13), also in `outcome.md`. No change cards are created by the session.

## 6. Grill questions (push hard; each with why and what to probe)

1. **The worth rule.** Why: "one worth rule" decides whether the economy can be gamed. Probe: is worth tied to drop rarity, to vendor price, to rune rank? Can a player sacrifice gold forever and skip farming? Does Hel cost 20 times Ist or does it grow exponentially? What stops a gold exploit from the T1 vendor loop? Risk: a flat rule makes junk the best fuel; an exponential one makes T5 unreachable.
2. **Sacrifice cost versus farm rate.** Why: it sets the whole pace. Probe: target hours of play per tier (U: none exist); is a tier purchasable in minutes at clvl 5 yet needs a Hel at T5? Which tier must a first-time player reach in the first session? Risk: too cheap and the ladder is a menu, too dear and it is a grind that the bot must already automate to afford.
3. **Context-key buckets.** Why: the rule "same answer, same context" is only as good as the key. Probe: too fine (exact life percent, exact monster) learns nothing; too coarse (area only) learns wrong rules, such as "always retreat in Cave" from three Fire Enchanted packs. Name the key for the T0 retreat point and for one T1 loot point; who defines the buckets (data table per decision point)? What happens at a bucket boundary?
4. **N by risk 2/3/5.** Why: it is the learning dial. Probe: who assigns risk to a decision point (static in the catalogue, or from consequence)? Does a contradicting answer reset the streak or only pause it? Is "same answer" strict equality for numeric answers (drink at 40 versus 45)? What is the escape when the player is just click-through-tired (a rule learned from boredom)?
5. **Soaks versus the answer log.** Why: learned rules make a run depend on the answer history (V draft section 8, Measurement). Probe: do Gate 2 and Gate 3 need a scripted answer file and a recorded rule set per run, and does the run record carry it next to `writes after start: N`? Is a soak with a rule learned from a human answer valid evidence? Does an answer count as a gameplay-affecting input under the crutch rule (AGENTS.md)?
6. **Unattended timeout.** Why: a hero waiting forever wrecks uptime, a hero guessing wrecks trust. Probe: default wait before the hero acts on its safe default (minutes? never?); that default action must not count toward any streak; does uptime exclude time waiting for a human? What is the safe place when the hero is mid-fight (U: town may be unreachable)?
7. **ADR 0015 "done means".** Why: decisions.md currently calls a fresh uncheated 24 h Act 1 clear plus the A:B verdict "done" (V ADR 0015). The draft proposes Gate 2 and Gate 3 as the T4 acceptance, and the product bar as T5. Probe: does the ladder amend item 1 of ADR 0015 or add a second bar? What is the T5 farm gate (hours, a goal like "3 runes of rank r05 or better", drops per hour against a model)? Who waits for T5 before "done"? Does G-A to G-F still hold?
8. **Rebirth and the altar.** Why: it decides whether hardcore is a reset or a prestige. Probe: do learned rules, altar quality, both or neither survive? Stored per account or per character? If quality survives, a rebirth hero at clvl 1 has T5 quality and no gear: is that fun or broken? What does a rebirth hero actually lose (levels, items, quests)?
9. **Un-parking the idle layer.** Why: PLAN.md, roadmap M6 and ADR 0015 list the idle layer as deferred or parked, to be defined "by a grill after M5/G-A" (V). Probe: is this grill that grill, so M6 moves up, or does it only write the ADR and keep building stays frozen until G-A? Which of the "deferred or parked" lines in ADR 0015 item 10 does ADR 0016 supersede, and which stay (container, hardcore build)?
10. **Runewords versus the farm goal.** Why: the accepted tension (decided 3) still needs a rule. Probe: does a farm goal reserve runes for a named runeword (a keep-list like `cubejobs` has, V `internal/cubejobs`)? Can the altar refuse reserved runes? Who decides, the player per goal or the tier? What does the hero do on finding the exact goal rune: sacrifice it or stash it?
11. **The early game must be short and fun.** Why: a clvl 1 hero with orders only could be dull before T0 unlocks (V draft section 8). Probe: how many minutes to T0, how many questions in them, what is on screen? Is the first town trip scripted? Does a second rebirth skip the tutorial because rules survive? Define "fun" as a measurable check (questions per 10 minutes, time to first unlock).
12. **Watch the hero view (rank 2) and recovering from a wrong rule.** Why: rank 2 is the second shape, and learning with no undo is a trap. Probe: what does the watch view need from T0 (embedded viewport, V `docs/d2d-view.md`; the journal; the uptime bar)? Where do learned rules live and how does the player see them (a rules list, a "why did you do that" link)? Is revoke one rule, one tier, or reset all? What if a wrongly learned rule killed the hero?
13. **Scope of v1 and the card budget.** Why: the cap is 12 open cards, each mapped to one goal G-A to G-F (V `docs/cards.md`, ADR 0015 G-F). Probe: which tiers ship first (T0 to T1 only, or to T4 where the code exists)? Expect up to three cards: T5 farm-goal spec (drop tables beyond Normal Acts 1 and 2, run planner, journal), the decision-point store (spec exists, draft, no code), the question-card UI. Which G does each map to (G-F? a new goal?) and which existing cards drop priority (U, from the draft: m2-roam, m2-tripwp, new town economy tuning, task-13 container soak, task-15 stream-grid versus rank 2)? Is any current card a blocker for the ladder (the T4 cards task-1 to task-3 stay)?
14. **Direction fit with the freeze.** Why: the tooling freeze allows design docs only (V draft header). Probe: what is the first thing that may be coded and under which goal, so ADR 0016 does not become a seventh stream (V `docs/roadmap.md` section 1)?

## 7. Risks (read before you start)

- All SME numbers (N values, decision points, farm targets) are S or U; the draft says so. Treat them as candidate questions, not premises.
- Testing explosion if unlocks stop being linear (2^k configurations); rebirth adds the state "rules present, character fresh" (V draft section 8).
- Scope: the engineering is at T3/T4, the product value sits in T0 to T2 feel and T5. A T5 push before T4 passes repeats the "seven streams" problem.
- EULA and offline: the player supplies the data files; nothing touches Battle.net (V AGENTS.md); custom d2d objects stay in the private fork (ADR 0003).

## 8. Recommendation of the prebrief author

Withheld until the user has answered section 6, per the grill agent rule.
