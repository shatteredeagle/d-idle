# Credits ledger

> status: current (card creditkit). One row per contributor handle: how they want to be credited and whether they agreed. The maintainer edits it by hand; `tools/credit/credit.py` reads the same facts from each card's `credit:` block. Rules: [../process/credit.md](../process/credit.md).

| handle | credited name / pseudonym | preference | consent | since | notes |
|---|---|---|---|---|---|
| shatteredeagle | (owner; numeric noreply identity) | git author of our own agents' work | yes | n/a | `Agent: Claude Code` in the PR body |
| chuckado | not given yet (default: the handle) | not given yet (author vs Traced-by) | asked; owner-approved for the upstream Traced-by credit on 2026-11-02; no reply received to the consent question on #19 | 2026-11-02 | PRs #19 (PALAURA), #22 (PALDUR), #23 (PALSKILL) in this repo; upstream: Traced-by on the Smite commit of psi29a/d2decomped #177 (merge commit 3e073888); no other upstream use until `yes` |

Consent values: `none` (not asked), `asked` (waiting), `yes`, `withdrawn` (nothing of theirs goes upstream or into a trailer). A person with `asked` may be named only in this repo; as upstream git author the consent must be `yes`.
