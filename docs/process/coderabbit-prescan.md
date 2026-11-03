# CodeRabbit pre-scan runbook (d-idle repo and the d2decomped fork)

> status: current, card crsetup, 2026-11-02. Owner go given 2026-11-02. Nothing here runs until the owner installs the app (step 1). Limits and quota facts: [../research/coderabbit-limits-20261102.md](../research/coderabbit-limits-20261102.md). Tags: V = fetched or run this session (source named), S = secondary, U = unverified.

Goal: let CodeRabbit review our own work first, so the one upstream PR per bundle arrives already cleaned and spends the maintainer's hourly allowance only once.

## 0. What was configured and how it was checked
- `.coderabbit.yaml` at the repo root (d-idle): en-US, profile chill, auto review for non-draft PRs on `main`, `path_filters` for generated files, lockfiles and binary fixtures, `path_instructions` for `docs/research/**`, `tools/**`, `internal/**/*.go`, `cmd/**/*.go`, `.github/**`. V.
- Fork config: `openspec/changes/crsetup/fork-coderabbit.yaml` (NOT at the repo root of this repo). It must live on the fork's `main` only and never in a commit that goes upstream; upstream has no `.coderabbit.yaml` (V, [limits doc](../research/coderabbit-limits-20261102.md) section 1), so the file only changes reviews on the fork.
- Schema check (V, run 2026-11-02): schema fetched from https://coderabbit.ai/integrations/schema.v2.json; both files validated with python `jsonschema` (Draft 2020-12), 0 errors. The schema has `additionalProperties: false` at the top level only, so a typo under `reviews:` would pass; the keys under `reviews`, `auto_review` and `chat` were also checked by name against the schema's property lists (all present).
- `reviews.*` has no commit-message check (V: no such key in the schema; the only text checks are `pre_merge_checks` for title, description and custom checks). The commit rules (short subject, at most three body lines, no co-authored-by or AI lines, `Reported-by:`/`Traced-by:`, PR body ends `Agent: <name>`) are therefore in a `path: "**"` instruction on the fork and are only checked as far as the PR title and body show them. The real gate stays `tools/pr_guard.py`, `tools/presubmit.py` and the maintainer.
- d-idle `auto_review.ignore_usernames` keeps the owner's own pushes unreviewed (existing line); remove it if the owner wants his own PRs scanned.

## 1. Step 1: the owner installs the app (only the owner can)
1. GitHub, Settings, Applications, Installed GitHub Apps (or the CodeRabbit dashboard at coderabbit.ai, sign in with GitHub); install or configure CodeRabbit.
2. Repository access: "Only select repositories"; select `d-idle` and the fork of `d2decomped` (two repositories). Nothing else.
3. Approve the permissions prompt (organisation admin approval where it asks). Tell the coordinator when done.

## 2. Step 2: put the fork config on the fork's main (coordinator, after step 1)
One commit on the fork's `main` containing only `.coderabbit.yaml` (copy of `openspec/changes/crsetup/fork-coderabbit.yaml`), by a clone or the contents API, for example (author identity per the fork's rules, run by the coordinator, not by agents):

    gh api -X PUT repos/<owner>/d2decomped/contents/.coderabbit.yaml \
      -f message='Add CodeRabbit config for staging PRs' -f branch=main \
      -f content="$(base64 -w0 fork-coderabbit.yaml)"

Rules: never include the file in a branch that goes upstream (a PR from the fork's `main` would carry it; always branch from `upstream/main`, never from the fork's `main`). Check before each upstream PR: `git diff upstream/main --stat -- .coderabbit.yaml` is empty.

## 3. Step 3: THE QUOTA TEST (coordinator runs it)
Question: does a CodeRabbit review on the fork spend the same per-developer allowance as a review on the upstream repository? Known: limits are per developer identity (V, https://docs.coderabbit.ai/management/rate-limits: "Review limits apply per developer identity, not per organization"); OSS PR review limits are "additionally scoped per repository" (V, https://docs.coderabbit.ai/management/plans, fetched 2026-11-02); the rate-limit page does not mention forks (V, absence). So the outcome is open (U) and the test measures it.

Procedure:
1. Baseline. Note the time (UTC) of CodeRabbit's last review or limit comment on any upstream PR of ours, and the exact limit text if any ("Review limit reached", "Next included review available in N minutes"). Do the test only when no upstream PR of ours is waiting for a review and none was pushed in the last hour. Also post `@coderabbitai rate limit` on one existing upstream PR only if the owner agrees (it is a comment; GitHub writes need the owner's go).
2. Staging PR. On the fork create a branch from the fork's `main` with a trivial docs-only change (one line in a scratch doc), open a non-draft PR against the fork's `main`. Record: time opened, time of CodeRabbit's first reaction, and every bot comment verbatim, including any text with "rate limit" or "review limits" and the "Plan:" and "Configuration used" lines of the run header (they show whether the fork config was applied).
3. Second push, 10 minutes later: one more trivial commit to the same branch. Record whether CodeRabbit re-reviews (incremental review) or reports a limit and with how many minutes.
4. Cross-check, 1 hour after step 2: `@coderabbitai rate limit` on the staging PR; compare the allowance shown with step 1.
5. Close the staging PR and delete the branch.

Interpretation (S/U, to be read against the recorded text):

| Observation | Meaning |
|---|---|
| Step 2 reviewed at once, step 3 re-reviewed incrementally, no limit text | Fork has its own allowance or the allowance was not exhausted; fork does not share (or not visibly). Repeat once right after an upstream review to separate the two cases. |
| Step 2 reviewed, step 3 reports "next review in N minutes" with N near 60 minus the time since step 2 | The fork has an allowance of 1 per hour of its own (OSS per-repository scope); it does not share with upstream if the step 1 upstream state is unchanged. |
| Step 2 gets a limit message although no upstream review ran in the last hour | The fork shares the per-developer allowance (or a stale upstream spend counts). Pre-scan costs upstream capacity: stage only when upstream is idle. |
| Step 2 reviewed, and an upstream PR pushed within minutes afterwards reports a limit with a refill time matching step 2 | Shared. Order matters: stage, wait out the hour, then open upstream. |
| Step 2: no reaction at all or "review skipped" | Fork has under 10 stars (public repo rule, V plans page): trigger with `@coderabbitai review`; the quota effect still shows in the reply. Also check the app install covers the fork. |

To be sure of "does not share", the decisive case is the fourth row run in reverse: an upstream review within minutes after a fork review must still be granted without a limit text. One such observation is evidence, not proof (one sample, U).

## 4. Step 4: pre-scan workflow for a bundle
1. Finish the bundle locally: all commits on one branch from `upstream/main`, within the budget of the limits doc (40 files or fewer, about 1000 changed lines, 15 commits or fewer).
2. Stage it on the fork: push that branch to the fork, open ONE PR against the fork's `main` (non-draft; drafts are skipped by `auto_review.drafts: false`). The fork's `main` carries the CodeRabbit config; the branch must not.
3. Wait for the review once (a few minutes; `gh pr checks <n> --watch`). If a limit text appears, wait the stated minutes, then comment `@coderabbitai review` once (owner's go for the comment).
4. Triage the threads with the person or the owner; collect every fix into ONE commit set and ONE push. Do not push to the staging PR more than once per hour; a second review only when the first produced real findings.
5. Open the real PR upstream with the same commits (the staging branch after fixes, rebased on `upstream/main`, fork config absent). Do not open it within the hour after the last staging review if the quota test showed sharing.
6. Close the staging PR with a link note; keep the evidence rows in the upstream PR body per the upstream AGENTS.md.
Nothing is posted to GitHub without the user's go; staging PRs are GitHub writes.

## 5. Step 5: the CodeRabbit CLI alternative
What it is (V, https://docs.coderabbit.ai/cli, fetched 2026-11-02): a local reviewer of uncommitted or committed Git changes before a PR exists. Commands `coderabbit` and alias `cr`; install `curl -fsSL https://cli.coderabbit.ai/install.sh | sh` or `brew install coderabbit`; sign in with `cr auth login` (browser) or `--api-key` (Agentic API key); marked open beta. Reviews are described as differing in depth from PR reviews.

| Question | Answer | Certainty and source |
|---|---|---|
| Free? | Free tier exists: "Basic static analysis with limited daily usage"; free users may see an upgrade prompt when limited | V, CLI docs |
| Limit | 3 CLI reviews per developer per hour on Free and on OSS; Essentials 5 | V, https://docs.coderabbit.ai/management/plans |
| Avoids PR quota? | Probably yes: the limits are listed separately (CLI per hour) and the Enterprise adaptive-limit text says "Only native Git pull request and merge request reviews count; CLI and IDE reviews do not". For Free/OSS no sentence says the pools are separate. | S/U, https://docs.coderabbit.ai/management/rate-limits (inference from separate table rows; not stated) |
| Needs the GitHub app? | No app install needed to sign in; for public repos you do not administer, the CLI falls back to "limited/free review behavior" | V, CLI docs |
| Reads our `.coderabbit.yaml`? | Not stated on the pages fetched | U |
| Where does the code go? | The diff is sent to CodeRabbit's service: same exposure as a PR review; owner's decision | S |

Recommendation: try the CLI on the bundle first (3 per hour, one account login by the owner, no GitHub write), then stage on the fork only if the CLI findings were not enough. Whether it spends the PR allowance is part of the quota test: run `cr` once, then check step 3 row results afterwards.

## Limits of this runbook
Nothing was run against GitHub or CodeRabbit by the author; the quota test and the CLI sign-in are unrun (U). The interpretation table is a hypothesis list, not a result.
