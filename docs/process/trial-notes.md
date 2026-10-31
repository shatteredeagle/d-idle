# Trial notes: change cards, specs and the checker (SPECBUILD, 2026-10-30)

Tags V (measured or run, how stated), S (doc), U (judgement). Everything here is from one build session by one agent; no multi-agent evidence yet.

## What was built (V: files in the branch, `go test` green)
`openspec/` (README, 7 specs with 53 requirements, 5 active cards, 11 archive cards, `.adopted`), `docs/OWNERS`, `docs/adr/` (8 ADRs), `cmd/changecheck` (check, `status [-write]`, `new`, `-diff`, `-cli`), `tools/openspec.sh` (pinned CLI wrapper), `.claude/agents/spec-sme.md`, doc fixes and 13 stale banners.

## Overhead, honestly
Wall times below are my estimates from the session, not instrumented (the commit clock in the container is too coarse to use; U).
| item | minutes | note |
|---|---|---|
| one seeded spec (6 to 10 requirements, with sources) | about 4 each, 25 total | most time is finding the source number, not the format; the format itself costs about 20 percent extra over prose because of scenarios |
| one ACTIVE card by hand (proposal, header, tasks, gates) | about 3 | the Why (50+ chars with evidence) and the gates are the real work |
| one ARCHIVE card from a merge | about 1 | one generator call; the facts come from `git log` and the doc already written by the agent |
| a card with `changecheck new` | under 1 to scaffold, 2 to fill | not used for the seeded cards (written by script), so this is U |
| writing the checker and tests | about 15 | 9 tests, about 600 lines |
So a card is about 3 minutes if the coordinator already knows the job, the same time as writing the brief it replaces. It is NOT cheaper than a brief; its value is that it persists, is checkable and is the same text the agent reads.

## Double bookkeeping I felt
1. Card header `owns:` vs `docs/OWNERS` lines: the same globs twice. changecheck catches drift, but better to GENERATE OWNERS from the cards (the standing defaults stay hand-written). Cut candidate 1.
2. Card `gates:` vs the status board items vs the report: three places say "done means X". Keep gates on the card, let the board items be generated from `tasks.md` checkboxes (not done).
3. Archive step: `git mv`, set `status/merge/result` in the header, drop OWNERS lines, regenerate STATUS, apply spec deltas. The CLI `archive` does only the delta merge and the move; our header stays `in-progress` after it (V, see below). A `changecheck archive <id> <hash> "<result>"` command would remove four manual steps. Not built; recommended.
4. BACKLOG.md is now a third list of "what to do" next to cards; the open items of BACKLOG should become cards or be dropped.

## What changecheck caught (V: ran on the repo while building)
- Real: the first run flagged two merges (`specresearch-a/b`) naming no card (fixed by `openspec/.adopted` = the adoption commit, so only later merges count); a STATUS block that was missing; OWNERS missing (twice, in the build order). Nothing caught on d2dwire/d2dlab/d2dcast/d2dprice themselves: they were written disjoint, so the overlap check never fired on real data (U whether it will under 8 agents; it is covered by unit tests with synthetic overlaps).
- My own mistake the process would have prevented: I first typed the d2dbackend merge hash wrong (`e2b966c` is the branch tip, the merge is `e2eb3d5`); `git log --merges` cross-check is what exposed it. Hand-typed hashes in cards are a risk: have the archive command read the hash from git.
- The ownership check works on globs only: `-diff agent/X` compares a branch with its card at merge time (unit-untested against a real repo, U; it shells out to git).

## CLI: worth it? (user asked; V = run with tools/openspec.sh on throwaway copies under /tmp, then deleted)
- Setup: `tools/openspec.sh` = `npx -y @fission-ai/openspec@1.15.0 "$@"` with telemetry and update ping off; blocks `init` and `update`. No install, nothing written outside `openspec/` (run in /tmp copies; the repo itself was only read). Startup about 1 s once the npx cache is warm (V `time`), needs the network the first time.
- Node: The head box has node v24.22.1; the worker host has node v26.12.1 and npx 12.2.1 (V, `node --version` read-only over ssh); both exceed the 20.19 minimum. Not installed by me.
- `validate --all --strict` on all 7 specs and 5 changes: PASS with our header block in tasks.md, with `.openspec.yaml` `skip_specs: true` (INFO line only), and our extra seeded delta (V). So the header and the cards are CLI-compatible.
- `archive tune-g1 --yes` on a copy with one MODIFIED and one ADDED requirement against `soak-gates` (V): merged atomically (+1 added, ~1 modified), the folder moved to `archive/2026-10-30-tune-g1`, `validate --all --strict` still passed, our header was left untouched (`status: in-progress`, empty `merge:` stay) so it must be updated by hand or by our own command. It also warned about unticked tasks and continued because of `--yes`.
- Name mismatch: a MODIFIED block with the header `G1 belts` (spec has `G1 belt`): `archive` aborted with `MODIFIED failed ... not found. Aborted. No files were changed.` and a recursive diff of `openspec/` before and after showed no change (V). BUT `validate --all --strict` PASSED that same broken change (14 passed, 0 failed): validate does not catch a delta that targets a missing requirement; only `archive` does. This contradicts the reference doc's note that validate reports "Archive would refuse this delta" (V for 1.15.0 in this setup; the note may apply to another invocation form, U). Practical rule: run `archive` (or check the delta names by hand) BEFORE trusting a green validate.
- What validate caught that changecheck did not: nothing on the real repo (everything passed in both), because changecheck's `spec` check copies the same rules. The one thing validate has and we do not: the scenario-loss check on MODIFIED and delta-vs-main name resolution at archive time (needs the real merge logic). What changecheck has and validate does not: ownership overlap, exclusive resources, OWNERS and STATUS sync, merge-names-a-card.
- Minutes saved vs hand-editing: a delta merge by hand is about 5 minutes per change and error-prone (partial MODIFIED blocks lose scenarios); the CLI does it in a second and atomically. With one or two spec-changing changes per day this is 5 to 10 minutes a day. Most changes here have NO spec delta (`skip_specs`), so the saving is small today (U for volume).
- Recommendation: KEEP the wrapper, use the CLI for exactly two things: `archive` when a change carries a spec delta, and `validate --all --strict` in CI or the merge step where node exists. Do not adopt `init`, skills or `/opsx:*`. `changecheck -cli` runs validate as an opt-in second opinion (skipped silently without node); keep it opt-in so changecheck stays fast, dependency-free and usable offline. Pin stays 1.15.0; re-run the trial before bumping.

## What I would cut
1. The `.openspec.yaml` file in every card (only needed for CLI archive; `changecheck new` writes it, hand-written cards can skip it and add it when a delta appears).
2. OWNERS lines for active changes (generate from cards).
3. `worktree:` in the header (derivable from the id); kept for now because the brief template prints it.
4. The 11 archive cards are low value as folders (merge hash and one result line would do in a single `archive/INDEX.md` table); I made them to prove the format, do not keep doing this for every tiny merge: archive only cards that carry a spec delta or a decision, and let the generated STATUS "recent merges" cover the rest.

## 2-week plan (warn-only first)
- Days 1 to 3: warn-only everywhere. The coordinator creates a card before launching an agent (`changecheck new`), runs `changecheck` at each merge, regenerates STATUS (`status -write`) at each merge. Collect: how many warnings were real, how often a card was skipped.
- Day 4: make `card` (required files and header) and `overlap` (owns and exclusive) fatal in the merge step: these are the failures that cost us tonight (shared socket, binary path, two agents in one dir).
- Day 7: make `merge` (merge names an existing card) and `diff -diff agent/X` (branch inside owns) fatal, with a `Change:` trailer escape for hotfixes.
- Day 10: make `status` and `owners` fatal only if they are generated; if still hand-maintained by then, delete them instead (cut 1 and 2).
- Day 14 review (decision rule from spec-approach-evaluation.md section 5): count failures a card or `owns:` check would have prevented versus ones it would not. If the layer saved fewer than about 2 collisions or restarts of context, drop the cards and keep the specs, ADRs, OWNERS and the generated STATUS.
