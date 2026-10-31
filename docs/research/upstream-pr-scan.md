# Upstream PR scan: psi29a/d2decomped (as of 2026-10-30)

> status: research, read-only. Nothing was posted, commented, starred or forked. Data: `gh pr list/view` and `gh api` GET via the worker host's gh (public data), plus a scratch clone of upstream main (507e3c06) under /tmp. Tags: V = fetched and read (GitHub PR/API JSON, the repo's own files at that commit), S = secondary/inferred from those, U = unverified. Classic 1.14d context only.

## 1. Headline (V)

- 95 PRs numbered 1..125 (some numbers are issues): 88 merged, 4 closed unmerged, 3 open (#119, #120, #125). Counted from `gh pr list --state all` JSON.
- Closed unmerged: #55, #64, #108, #111. None was closed for being AI-written.
- The maintainer (psi29a, Bret Curtis, 1277 of 1362 commits) runs the project "agents-first": AGENTS.md is for Claude Code, Codex, Cursor, Copilot, Gemini; PR template has an `Agent:` line; the maintainer's own PRs #123 and #125 say `Agent: Kimi` / `Agent: Claude Code`.
- Confirmed agent-submitted by explicit disclosure: #111 (closed), #119, #120 (open), #124 (merged), plus the maintainer's own #123, #125. Closed #64 is an app bot (Sentry).
- Policy as observed: AI help is welcome and invited; review is identical for everyone; PRs are judged on evidence from game.exe, one claim each, scope and process. No PR was rejected because an agent wrote it.

## 2. All closed (unmerged) and non-maintainer PRs (V)

Closed unmerged:

| # | Author | Opened / closed | Title | Size (+/- , files) | Verdict | What the maintainer did |
|---|---|---|---|---|---|---|
| 55 | psi29a | 10-19 / 10-19 | Coverity mana | +721/-182, 39 | maintainer's own, UNCLEAR | Closed: its pad work was merged into a branch after #52 was squashed, so it moved to #54; "the rest is already on main". CodeRabbit rate-limited. |
| 64 | app/sentry | 10-21 / 10-21 | Fix mini automap clip_y0 underflow crash | +1/-1, 1 | CONFIRMED automated (Sentry bot, Co-authored-by sentry[bot]) | Thanked, closed: "fixes the symptom and moves the map" (clamping clip_y0 also shifts clip_y1 by 21 rows); real fix landed in #65 (09e7d93) with the reason. CodeRabbit skipped ("Bot user detected"). |
| 108 | mendescotta | 10-28 / 10-28 | Docs: counts and claims as the code stands | +42/-23, 8 | LIKELY agent (same author as #111) | Self-closed "Superseded: landed with #109". |
| 111 | mendescotta | 10-28 / 10-30 | Index research records and review defects in a checked JSON ledger | +17380/-24, 20 | CONFIRMED (`Agent: Codex`; author said "just let the agent go brr, treat this as a draft or close it") | psi29a first asked "Why toml? Let's discuss before unilaterally pulling in another dep... slow our roll". Closed two days later with a long, polite explanation (see section 4). |

Non-maintainer merged or open PRs (author accounts: mendescotta created 2013, 22 PRs, 79 commits; CascaMori created 2026-07, 1 PR; Gradipoo 2015, 1 PR):

| # | Author | State | Title | Size | AI signs | Maintainer |
|---|---|---|---|---|---|---|
| 62 | Gradipoo | merged 10-28 | switches.md: game.exe command-line table | +685/-3, 9 | UNCLEAR, human-voiced, asked questions | Draft first ("not ready, no code"), "solid first cut, the research note is great", Discord invite |
| 88 | mendescotta | merged | Walk/run animations at walk's rate | +98/-17, 11 | LIKELY: template-like "What changes / How / Checks", FUN_ addresses, emu oracle 1203/1203, commit bodies cite addresses | "awesome, thank you" |
| 89, 90 | mendescotta | merged | Golden Bird; Rally no leader | +55, +28 | LIKELY (same style) | merged |
| 91-94, 97 | mendescotta | merged | tooling, MonProp.txt stats, block/kick sounds, Jab dosound, emu helpers | 150-474 lines | LIKELY | merged after review rounds (CodeRabbit + psi29a) |
| 100 | mendescotta | merged 10-29 | Follow-ups to #88/#90/#92 | +136/-47 | LIKELY | CHANGES_REQUESTED (thorns used wrong damage reduction), then merged |
| 101 | mendescotta | merged 10-29 | tools/presubmit.py + review-checklist.md | +790/-6 | LIKELY | CHANGES_REQUESTED (2 bugs), re-reviewed, merged; maintainer: "Oh dang! That's awesome" |
| 102, 106, 109, 110 | mendescotta | merged | sound traces, test skip, docs | 41-118 lines | LIKELY | merged |
| 113-115, 118 | mendescotta | merged | Item charges as skills; sound ducking; converted monster no target; overlay clocks | 90-573 lines | LIKELY; #115 openly tags "Unverified: redivivus's reading" | merged |
| 119 | mendescotta | OPEN | Port reviewed fixes from fork PR #6 | +238/-65, 18 | CONFIRMED `Agent: Codex` | CHANGES_REQUESTED: re-checked claims against 1.14d, most hold, "the shrine does queue event 1" claim is wrong, with address FUN_00583c70 and a fix |
| 120 | mendescotta | OPEN | Andariel/Diablo quest shake and audio | +672/-28, 27 | CONFIRMED `Agent: Codex`; mentions "Fresh prereview" | COMMENTED: "Nice work, thank you for shipping the oracle"; re-ran the oracle byte for byte; nits plus rebase request |
| 124 | CascaMori | merged 10-30 | Trace Throw base level separately | +144/-3, 3 | CONFIRMED `Agent: Codex` | APPROVED "lgtm, thankyou" |

Maintainer-authored PRs (rest of the 88 merged): most are large (e.g. #3 +16923/-1065, 101 files; #4 227 files; #122 gap map +73312). Disclosure only exists since the template (10-30): #123 `Agent: Kimi`, #125 `Agent: Claude Code` (open). Earlier ones: S, almost certainly agent-assisted (cadence of dozens of PRs per day, AGENTS.md/CLAUDE.md present, `.claude/skills/reverse-engineering`), but not disclosed, so UNCLEAR, not CONFIRMED.

## 3. What the maintainer asks of AI-assisted PRs (V: AGENTS.md, CONTRIBUTING.md, .github/pull_request_template.md, tools/pr_guard.py at 507e3c06)

- Any agent is welcome; same rules for every contributor, "maintainers and their agents included". `Agent: <name>` or `Agent: none` in the PR body "is for the record; it doesn't change the checks". The README advertises contributing with an agent without being a programmer.
- Three kinds of work only: play and report, trace a claim, fix or port (with a test). Pick a task from the gap map, search issues first so two people do not take the same one.
- Evidence bar (AGENTS.md section 5): game.exe 1.14d is ground truth; other projects (OpenDiablo2, D2MOO, Redivivus, community specs) "can confirm a reading, never replace one"; anything taken on their word must be tagged `// unverified (source: ...)` and listed in `unverified.md`. "Behaviour alone is not evidence. Neither is 'it looks like'." A match with game.exe needs an oracle (`tools/emu`); CI has no game.exe so a maintainer runs it.
- One claim per PR, with commands that re-check it; reviewer re-runs them on his own copy.
- Process: PR body follows the template (What changes, Evidence table by address, How to re-check, Screenshots, Checklist, `Agent:`), commits with short subject and at most 3 lines body, git author is the human, `Reported-by:`/`Traced-by:` trailers for credit; agent must ask the human before any outward action; CodeRabbit is advice not a gate; answering the review is part of the work.
- `tools/pr_guard.py` (CI): fails impossible FUN_/LAB_ addresses (outside .text 0x401000-0x6cb5a1) and any address between sections, binary files, game files; requires a maintainer `security-reviewed` label for URLs, downloads, networking, process launches, base64, long byte tables on added non-.md lines and for edits to .github, AGENTS.md, CLAUDE.md, CONTRIBUTING.md. WARN for FUN_ not a function entry.
- LEGAL.md: no mention of AI-written code (grepped README/LEGAL for AI/agent/LLM: only the README contributing line). Contributions are GPL-3.0-or-later; no game files or decompiled C in git.
- No stated ban or limit on AI contributions, no label for AI, no automated "please read AGENTS.md" replies seen. The only wording on violations: "A rejection comes with a reason."

## 4. How the maintainer reacted to problems (V)

- #111 (Codex, ledger of 6 JSON files, +17k lines, introduced a toml dependency): first reply asked for discussion before a new dependency; the author answered flippantly; the maintainer still closed it politely after 2 days, citing the agents-first process (gap map `tools/gaps.py` makes the ledger redundant, review comments are not evidence under AGENTS.md section 5, "delete the row when done" in todo.md), listing the doc fixes that landed anyway, and pointing to a `todo`/`unverified` gap row as a good next PR. So: reasons given, duplication/scope was the cause, not AI authorship.
- #64 (Sentry bot): closed with a correct technical explanation and the real fix.
- #119, #100, #101: CHANGES_REQUESTED for a wrong claim or a bug, each with the game.exe address and a suggested fix. The maintainer re-checks claims against his own decompile; wrong claims get caught.
- Tone throughout: warm, thanks plus Discord invite. No hostility to agents.

## 5. Our two prepared PRs (V: ~/d2d-lab/pr/{price,cast} on the worker host; both on base 3306e60; upstream main now 507e3c06, 3 newer commits touching only AGENTS.md, throw.md, unverified.md, fight.hpp)

Overlap check (V): upstream main has no `cast_ticks`, no CostPerCharge, and `item_price` still multiplies by quantity only for non-stackables (rules.hpp). No open or closed PR or issue is about cast timing, FCR or vendor prices (#86 is walk animation, #85 item charges). Nobody else is duplicating either. The newest AGENTS.md added a review-loop section (4b2e9a11), so rebase before posting.

| | price (fix/vendor-prices) | cast (fix/cast-timing) |
|---|---|---|
| Size | 4 files, +53/-4, one commit | 6 files, +84/-13, one commit |
| Template | complete; `Agent: Claude Code (Sonnet 5.5), working for the maintainer` | complete; `Agent: Claude Code (Claude Sonnet 5.5), working for the maintainer` |
| Evidence | game.exe addresses (0062efb0, 0062f2a2, ...) read with tools/emu/disasm.py; live prices 450 and 81 on the original | explicitly "Not traced in game.exe by this PR"; D2MOO, community tables, AnimData and Skills.txt |
| Claims | one: tome and quiver price by quantity | three: -1 tick for plain casts, Lightning uses the 19-step sequence, doc corrections to skills.md (Druid speed etc.) |
| Oracle | none; honestly says offsets come from table layout and live prices, "a maintainer with game.exe can settle it" | none; "no source checks the tick count in play" |
| Risks | stackable multiply left as `ponytail:` (good: shows scope control) | commit has `Co-Authored-By: Claude` trailer and the maintainer's name as author (price: the same name capitalised); AGENTS.md section 6 asks for the human as author and does not forbid trailers, but no merged commit has one |

Will they conform? Mostly yes on form. Differences from what the maintainer enforces:

1. price: strongest. One claim, addresses, a test, live numbers, small. Weak spot: struct offsets (+0x14 CostPerCharge, ItemTypes +0xe) not read from loader code; he said in #119 he re-checks against decompile, so he will run it. A wrong offset would be caught, so state it as the PR already does. Likely outcome: merged, perhaps after one CHANGES_REQUESTED round.
2. cast: weaker under section 5: the evidence is another project (D2MOO) plus community tables, which AGENTS.md says "can confirm a reading, never replace one", and it admits no game.exe trace. It also bundles three claims (timing offset, sequence use, doc corrections) against "one claim per PR". The Druid 208 speed and breakpoints rows replace upstream's existing skills.md table, which the maintainer wrote from his own work. Closest to a rejected pattern: #115 style "unverified, redivivus" was accepted only because it was tagged and listed; #111 shows he closes broad PRs that overlap his own process. The honest disclosure helps, but the maintainer's bar is "game.exe is ground truth".
3. Neither PR uses URLs in non-.md added lines (cast's code comments: U, run pr_guard); run `python3 tools/pr_guard.py` after rebasing since the checklist box in cast is unticked.
4. Size is fine (his merged median is about 200-500 lines; ours are well under).

What would make him close or bounce them: a claim that game.exe contradicts (he checks, e.g. #119 shrine), an unlabeled reliance on community tables, bundled claims, a new dependency or URL, missing `unverified.md` row, or a diff that fights an in-flight upstream change.

## 6. Recommendation (confidence in brackets)

- price: post after rebasing onto 507e3c06 and re-running pr_guard and the tests. As is otherwise. [high that it conforms; medium-high that it merges]
- cast: do not post as is. First (a) trace the cast timing in game.exe with tools/emu or read FUN_006216E0 / 006217C0 yourself, or at least run the oracle for `cast_ticks` against it so the evidence is not D2MOO-only; (b) split into one claim (cast ticks / Lightning sequence) and move the skills.md Druid/table corrections to a second PR or issue, or trim them; (c) drop the Co-Authored-By trailer or keep it consistent with author name (cosmetic). Alternative: post as a "trace a claim" issue/draft with the measured table, which fits his invitation to report differences. [medium; the numbers look right, the process is what bounces it]
- Order: price first; wait for his reaction (shows how he treats Claude-authored work, which he evidently accepts), then cast.
- Posting needs the user's go (AGENTS.md of this repo); nothing was posted by this scan.

## Links

- Repo https://github.com/psi29a/d2decomped ; PRs: #111 https://github.com/psi29a/d2decomped/pull/111 , #64 .../pull/64 , #119 .../pull/119 , #120 .../pull/120 , #124 .../pull/124 , #62 .../pull/62
- AGENTS.md https://github.com/psi29a/d2decomped/blob/main/AGENTS.md ; CONTRIBUTING.md ; .github/pull_request_template.md ; tools/pr_guard.py
