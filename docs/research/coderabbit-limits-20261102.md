# CodeRabbit limits and bundle budget for the upstream d2decomped PRs

> status: research, 2026-11-02. Sources fetched that day. Tags: V = fetched primary source (URL given), S = secondary or snippet, U = unverified or inferred.

Question: the upstream maintainer asked for one PR per batch, within CodeRabbit's limits. What are the limits, and how big may a bundle be?

## 1. The maintainer's own configuration

| Item | Finding | Tag |
|---|---|---|
| `.coderabbit.yaml` / `.coderabbit.yml` in the repo | Absent (404). CodeRabbit's own run header on every PR says "File: No configuration file from this repository applied", "Layers: defaults", "Review profile: CHILL". So no path_filters, no path_instructions, no custom auto_review. | V (GitHub contents API on the upstream repo; PR comment headers) |
| Plan shown in the run header | "Advanced" (OSS projects get paid-tier features; see section 3) | V (same comments) |
| CONTRIBUTING.md / AGENTS.md on PR size | "One claim per pull request or issue ... Small and checkable beats big and vague." AGENTS.md section 6: CodeRabbit is "an AI reviewer, not a gate"; review loop: `gh pr checks <n> --watch`, then triage unresolved threads. No numeric size rule, no mention of batch size. | V (raw files via GitHub API) |
| `.github/` | CODEOWNERS, FUNDING.yml, flatpak, pull_request_template.md, workflows. No CodeRabbit config. | V |
| The batching request itself | Came from the maintainer to the user (Discord); not written in the repo. | U (hearsay via the user) |

Consequence: everything is CodeRabbit defaults for an OSS repository. Defaults include CodeRabbit's built-in path filters (lockfiles and similar are ignored); not checked here, see 3.

## 2. What CodeRabbit actually did on the maintainer's PRs

Observed from the bot's issue comments and reviews on psi29a/d2decomped (V, GitHub API, fetched 2026-11-02). Size is from `gh pr list`.

| PR | Files, +/-, commits | CodeRabbit result |
|---|---|---|
| 148 | 168 files, +15817/-14224, 7 | "Review skipped: Too many files! This PR contains 168 files, which is 68 over the limit of 100." The check says to split or change the base. The only hard size refusal observed. |
| 174 | 37, +333/-113, 14 | "Review limit reached ... used all free OSS reviews", next review in 49 min. "Files selected for processing: 37". No review posted before merge; the CodeRabbit check ended NEUTRAL. |
| 172 | 24, +700/-148, 10 | Limit reached once (next in 48 min), then a full review posted about 11 min after the comment. |
| 169 | 37, +818/-140, 18 | Limit reached (next in 17 min); no review posted. |
| 165 | 51, +405/-198, 40 | Reviewed (48 files selected, 3 inline comments). |
| 164 | 10, +678/-15, 12 | Limit reached once, then 5 review submissions over about 2 h (incremental, per push) with 6 inline comments. |
| 161 | 30, +793/-181, 21 | Limit reached (next in 43 min), then reviewed 1 actionable comment. |
| 151 | 25, +501/-83, 5 | Limit reached once, then reviewed (3 actionable). |
| 145 | 27, +441/-154, 15 | Limit reached once, then reviewed (1 actionable). |
| 141 | 21, +628/-128, 8 | Limit reached (next in 55 min); no review posted. |
| 133, 159 | 20 and 13 files | Reviewed or skipped without a limit message (159: "Review skipped" on one pass). |

Reading: below 100 files no size refusal was seen; line counts up to +1373 (PR 167) and 40 commits (PR 165) were accepted. The common failure is the rate limit: the free OSS allowance appears to be about one included review that refills in roughly one hour; each push (or the open event) spends it, so a PR with many pushes gets few or no reviews. U (inferred from the pattern, not stated for this repo).

Not found anywhere in the repo or the bot comments: a line limit, a commit limit, a statement of "too large" by lines.

## 3. CodeRabbit public documentation

| Limit | Value | Source | Tag |
|---|---|---|---|
| Max files per PR review, after path filters | Free 150; OSS 100 to 300 depending on project community and popularity; Essentials 150; Team, Advanced, Enterprise 300 | https://docs.coderabbit.ai/management/plans | V (official docs) |
| This repo's actual file cap | 100 (the bot's own message on PR 148) | PR 148 comment | V |
| Over the file cap | Review skipped. Usage-based paid review possible on paid plans up to 300 files, 0.25 USD per reviewed file; more than 300 files unsupported. Not available to the OSS plan in the PR 148 message ("Upgrade to a paid plan to raise the limit"). | plans page; PR 148 | V |
| Lines per review | Not documented. No line limit found. | plans and pricing pages | V (absence) |
| OSS PR reviews per developer per hour | 1 to 10, set per repo by star count | https://docs.coderabbit.ai/management/plans | V |
| Observed on this repo | One included review, then "Next included review available in N minutes" (6 to 55 min seen) | PR comments | V |
| Free (non-OSS) plan | 1 PR review per hour, summary only | plans page | V |
| Paid per-developer hourly refill | Advanced 10/h when under 50 reviews in 7 days, falling to 1/h at 80+; Team 8/h; Essentials 5/h | https://docs.coderabbit.ai/management/rate-limits | V |
| Scope of rate limit | Per developer identity, not per PR or org. Rate-limited pushes consume no review. A "Review rate limited" check passes, so it never blocks merge. | rate-limits page | V |
| Public repo under 10 stars | Reviews need a manual trigger (`@coderabbitai review`) | plans page | V |
| Commands | `@coderabbitai rate limit` (availability), `@coderabbitai review` (trigger when capacity returns) | rate-limits page | V |
| Default path filters (lockfiles, etc.) | Exist by default; exact list not fetched this run | none | U |

Note: the OSS hourly count is per developer identity, so the user's pushes share one allowance with whatever the maintainer also pushes under his own account only if the same identity; different accounts have separate allowances. U.

## 4. Recommendation: bundle budget

Binding constraint is the review count, not size. The size ceiling is 100 files (V); the maintainer's own accepted batches ran 24 to 51 files and up to +1373 lines.

| Budget per bundle | Target | Hard ceiling | Certainty |
|---|---|---|---|
| Files | 40 or fewer | 70 (30 percent margin under 100; path filters could only lower the count) | files cap V; margin is judgment |
| Added + deleted lines | 1000 or fewer total | none documented; keep under about 1500 (largest accepted upstream: +1373/-45) | U (no documented line limit; empirical only) |
| Commits | 15 or fewer, squash-style logical commits | 40 seen accepted | U (no documented commit limit) |
| Pushes to the PR branch | 1 initial push, then at most 1 push per hour | n/a | inferred from the refill pattern, U |

Practical rules:
1. Push the finished bundle once, as one branch with all commits, not commit by commit. Each push can spend the single hourly review. V that the allowance is a few reviews per hour, U on the exact refill.
2. After opening, do not push again for about an hour; if the limit comment appears, wait for the stated minutes and comment `@coderabbitai review` once.
3. Do not open two bundle PRs inside the same hour (same developer identity).
4. If a bundle would exceed 70 files or about 1500 changed lines, split along function boundaries (the maintainer's rule: two PRs in one function mean a hand rebase), each part self-contained, and open them at least an hour apart.
5. Keep the one-claim-per-PR evidence in the body per upstream AGENTS.md even in a batch: list each claim with its re-check command.

## 5. Our candidates

Sizes known from the brief only; the private branch diffs were not available.

| Candidate | Known size | Fit |
|---|---|---|
| Freeze rules | 8 files, +166/-9 | Far under every limit. |
| Vendor repair gate | 1 commit | Fits; size small by commit count. |
| Melee D1, Melee D2 | 1 commit each | Fit; combine with the others. |
| Ground bundle | 3 commits | Fits. |

Combined these are about 8 files plus 6 commits plus unknown file counts of the four small items: very likely well under 40 files and 15 commits, so one bundle PR is within budget. Check `git diff --stat` before opening: if the sum passes 40 files or 1000 changed lines, move the ground bundle to a second PR opened an hour later.

## 6. Open questions

- The exact OSS hourly refill for this repo (observed gaps of 6 to 55 min at the time of the comments) is not documented for it.
- Whether the default path filters change the file count of our diffs (not fetched).
- Whether `@coderabbitai review` bypasses the OSS allowance: no, per the rate-limits page it only works once capacity returns.
