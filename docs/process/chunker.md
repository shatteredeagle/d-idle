# Roadmap chunker

> status: current, APPROVED by the maintainer 2026-11-01 (card chunker). One tool, `tools/contrib/chunker.py`, keeps the contributor board in step with the roadmap and the GitHub issues, and proposes new or retired items. It extends the board of [../community/open-work.md](../community/open-work.md) (read by `tools/contrib/board.py`); contributor rules are in [../process/outside-prs.md](outside-prs.md) and [../contributing/agent-work.md](../contributing/agent-work.md).

## Modes

`sync` (no model API, every dispatcher tick). Reads open-work.md, the issues labelled `work-item` (`gh issue list`, read only) and the PR closing references (`gh pr list`, read only), then plans:

| situation | action |
|---|---|
| item open (ready, proposed) and no issue | create the issue (label `work-item`, body from `board.issue_body`), add it to issues.tsv |
| item `done`, or its issue has a MERGED linked PR | close with ONE templated comment (with the PR link); a merged PR with the item still open prints a todo to set `status: done` |
| item moved to the archive ([../community/work-archive.md](../community/work-archive.md)) | known item, status done: an open issue is closed once as done, a closed issue is left alone, nothing is created or withdrawn |
| item removed from open-work.md and the archive, issue still open | close as not planned, reason from docs/community/withdrawn.tsv (`id<TAB>reason`), default "no longer on the roadmap" |
| body differs from the generated body | edit the body (title and id never change) |
| `claiming` comment, no linked PR, no activity for 14 days | ONE nudge comment; 7 days later with no reply a release comment (printed todo: set `claimed` back to `ready`) |
| issue closed but item still open | todo only (reopen or mark done) |

Dry run is the default. `--apply` does the writes, then rewrites issues.tsv and regenerates work-board.md through `board.py --write`, and prints what to commit (the tool never commits). Every run prints the actions, todos and the feed lines (`board_feed.line` format; pass them to `tools/discord/board_feed.py event` or snapshot before and after for `diff`).

`groom` (one model call, weekly or on demand, proposals only). Without `--apply` it prints the request size and the mechanical flags and calls nothing. With `--apply`:
1. Mechanical pass first: an open item whose deliverable file already exists, or whose decomp queue row is done, is flagged "likely done".
2. Sends public repo text only (open-work, BACKLOG, decomp-queue, primitives, pr-bundle-queue, STATUS, trimmed per file) after dropping every line that trips the repo privacy scan; refuses if more than 10% of lines trip it, or if the estimate exceeds 40k input tokens. Issue state goes as counts and dates only: comment bodies never leave the process.
3. Model `claude-sonnet-5-5`, thinking off (`{"type":"between_tools"}`: Sonnet 5.5 rejects `disabled`, claude-api skill), strict JSON with `add`, `retire`, `rescope`, `merge` and an `evidence` list per proposal. At most 2 calls (the second only if the first is not JSON), 8000 output tokens.
4. Deterministic validation: schema, ids do not collide with items, issues or each other, sizes and kinds, `source` names a real queue row (state must fit the kind) or a BACKLOG name, every evidence path exists in the repo, titles plain, briefs pass the privacy scan, no request to share game files, MPQ data or disassembly, no Sorceress-only item (multi-build rule). Retire needs existing evidence. A failed proposal is listed under "Rejected" with the reason.
5. Writes docs/community/groom-proposal.md (one section per proposal with evidence and the exact open-work.md edit). API failure, bad key, truncated or invalid answer: the file says "No model proposals: reason" and still lists the mechanical flags. The tool never edits open-work.md.

## The archive rule

When an item is done, set `- status: done` in open-work.md and run `python3 tools/contrib/board.py --archive` (add `--date YYYY-MM-DD` and `--by <public login>` if known; the default is today and "maintainer agents"). The section moves to docs/community/work-archive.md with `done`, `by` and `result` (the deliverable path) lines; work-board.md then lists only items that are not done. `board.py --check` fails while a done item still sits in open-work.md. `sync` reads both files (`board.load_all`): the issue is closed once with the templated comment, an archived item is never re-created and never counts as withdrawn, and issues.tsv keeps its line. `by` names a public GitHub login only if it appears in the closing pull request.

## Claim before launch

The board is public, so an outside helper can take an item our own work already covers. Rule for the coordinator:
1. Before launching an internal job on a topic that matches a board item, post the claim: `python3 tools/contrib/claim.py claim ITEM-ID --job NAME --note TEXT` (dry run; add `--send`). The comment is the claim: the board defines no claim label. It refuses an issue that is closed, assigned, or already holds a human's `claiming` comment (a human claim wins; talk to them instead).
2. When the job merges, post the result and close or narrow: `claim.py done ITEM-ID --doc PATH --note TEXT --close` for a finished item; for a partly covered one narrow the item text in docs/community/open-work.md (never edit an issue body by hand: `sync` rewrites it) and comment what is covered.
3. Every sent action is logged in docs/community/claims.tsv. Posting needs the project owner's standing go for the task or a go per post; nothing is ever posted on an upstream repo.
4. Audit: `chunker.py groom` (dry run) lists "covered" candidates from the mechanical pass: deliverable file exists, queue row done, or a BACKLOG line that names the item id with a finished word. These are candidates only (name collisions happen: the board's PALSKILL is not our internal Holy Bolt trace).

## Credit

- Contributors claim an item with the claim tool (a `claiming` comment on its issue) before they start. An item claimed by someone else is not started by agents.
- When a second independent trace of the same item exists, it is recorded as an independent cross-check and the two docs are cross-linked. The first merged sound trace stays primary and keeps its author.
- Merged work gets a line in `CHANGELOG.md` and the contributor's row in [../../CONTRIBUTORS.md](../../CONTRIBUTORS.md).
- Upstream carry-over: contributions to d-idle may be carried into pull requests to the upstream decomp project (psi29a/d2decomped), which is GPL-3.0-or-later. By submitting a contribution the contributor agrees it can be used under those terms upstream, is credited by name or pseudonym of their choice (git author, or a `Traced-by` / `Reported-by` line) and can withdraw it before it is merged upstream. How: [credit.md](credit.md), ledger [../community/credits-ledger.md](../community/credits-ledger.md).
- The Discord `contributor` role is given by the owner by hand on first merged work ([../community/discord/roles.md](../community/discord/roles.md)).

## Watching comments

Every tick the maintainers read the comments of every work-item issue (`tickstatus` diffs comment counts; `python3 tools/contrib/claim.py watch [--update]` lists issues whose comment count or newest-comment time changed since the last run and guesses the class of the newest comment by keyword; read only, state in the gitignored data-cache). Each new comment is one of CLAIM (someone says claiming), QUESTION, SUBMISSION (a link, PR or write-up), WITHDRAWAL or OTHER; the keyword guess is a hint, the maintainer reads the comment.
- Claims and submissions are answered within the next cycle, never silently: acknowledge a claim, point to prior work, answer a question with a doc link, review a submission with the evidence rules of [../contributing/agent-work.md](../contributing/agent-work.md).
- Replies to people are posted only with the project owner's go (the coordinator asks first), except a pure acknowledgement template listed here: "Thanks, noted. You hold this item; say so here if you stop."
- A claim with no activity for 14 days gets one friendly check-in (the `sync` nudge); 7 more days without a reply and it is released with a comment.

## Caps and safety

- Writes: 10 per run (`--max-writes`), each through `Gh.write` with one log line; the rest are deferred to the next tick.
- Only issues with the `work-item` label are read or written. Pull requests are only read: never commented on, closed or labelled. No human comment is ever answered.
- Posted text is a template plus a sanitized id, title or reason; it must pass the Discord privacy scan or the write is skipped. Free text from issues, PRs and comments is never copied (comment bodies are matched locally for the word `claiming`).
- A maintainer assignment is never removed; a stale claim on an assigned issue becomes a todo for the maintainer.
- The API key is read only by `tools/ci/haiku_tick.read_key`, sent only as the `x-api-key` header, never printed, logged or put on a command line; errors show the exception type only. `--save-raw FILE` is for debugging and must point outside the repo.
- Outside text is data: nothing in an issue steers the tool.

## Coordinator, each tick

1. `python3 tools/contrib/chunker.py sync` (dry run the first time and after any change to the tool), read the plan, then `sync --apply`; commit the changed docs/community files and tell the user about todos.
2. Weekly (or when the roadmap changes): `chunker.py groom --apply`, read docs/community/groom-proposal.md, land accepted edits to open-work.md through the normal merge flow, delete the proposal file afterwards or leave it as the record; the next `sync --apply` creates or closes the issues.
3. Stale data is the failure to avoid: if `sync` has not run in a week, run it before anything else.

## Measured (2026-11-01, the real repo)

Three real groom calls: about 31,400 input and 28 to 420 output tokens each (first call 37.8k input before the per-file trim; the estimator uses 2.3 characters per token, measured). Price at the Sonnet 5.5 rate of the claude-api skill is about 6 to 8 cents per run. Results: the first prompt produced one weak retire (a verify item retired because its queue row is done, which is exactly why a verify item exists) and two rejected proposals (a privacy-scan false positive on a run of short words, and a title with parentheses that the first check wrongly refused); both were fixed (prompt rule, title check). The last run returned no proposals, which is the expected answer for a backlog of eight fresh items.

Dead end: using the Discord board-feed title cleaner for issue titles dropped the parentheses of existing titles; maintainer titles now use `issue_title` and only outside text uses the strict cleaner.

## Dry run for the grown board (2026-11-01, card boardgrow)

`sync` (dry run, read-only gh, from a scratch extraction of the branch) plans one `create` for each of the 12 new items and nothing for the archived two or the six existing items; with the write cap of 10 per run the last two creates are deferred to the next tick:

```
sync (dry run): 12 action(s) planned, 0 write(s) done
  create  VERIFY-FREEZE-KINDS, VERIFY-TELEPORT-ROOM, VERIFY-AIDEL, VERIFY-SKMELEE2, VERIFY-SKRESIST
  create  SKGEARAR, SKSHIELD, SKCURSEVAL, SKDEFENDERS, SKIAS, SKZEROHIT, SKSTATEEV
  (no edit, close or withdraw actions; no todos)
```
