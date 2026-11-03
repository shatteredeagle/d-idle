# Credit for work carried upstream

> status: current (card creditkit, approved by the maintainer 2026-11-02).

The upstream decomp project (`psi29a/d2decomped`, GPL-3.0-or-later) credits by git author plus two trailers: `Reported-by: Name` or `Traced-by: Name` as the LAST line of the commit message. Its gap map counts author, Reported-by and Traced-by. AI lines (`Co-authored-by`, "generated with", an anthropic noreply address, a robot emoji) are not allowed in commit messages; the PR body ends with `Agent: <name>`. Contributors choose a name or pseudonym and may withdraw before the merge. Tags: V = read in the upstream rules by the maintainer's brief of 2026-11-02, not re-read by this card (S for us).

## Our rules

1. Our own agents' work: git author is the owner (numeric noreply identity), PR body says `Agent: Claude Code`.
2. A helper who WROTE the fix: their commit is cherry-picked with their authorship; needs their consent (`yes`).
3. A helper who traced or reported it: commit author is the owner, the last line is `Traced-by: <credited name>` or `Reported-by: <credited name>`.
4. Every bundle PR body has a Credits section: per component who traced, who wrote, who independently verified. Verifiers appear in the body and in [../../CONTRIBUTORS.md](../../CONTRIBUTORS.md) only; they do not count for the gap map.
5. Nobody is lost: every change card whose work may go upstream carries a `credit:` block (schema in [../../openspec/README.md](../../openspec/README.md)). People and consent status: [../community/credits-ledger.md](../community/credits-ledger.md).

## The tool

```
python3 tools/credit/credit.py trailers CARD [CARD...]   # Traced-by / Reported-by lines
python3 tools/credit/credit.py credits  CARD [CARD...]   # "## Credits" markdown + Agent: line
python3 tools/credit/credit.py check    CARD [CARD...] [--lenient] [--message FILE]
```

`check` exits 1 when a component has no author, nobody traced it (a person or an agent), no `verified_by` (unless `--lenient`), a person withdrew, a non-owner author has consent other than `yes`, a person credited by name was never asked (`none`), or an AI line shows up in a trailer, a credited name or a commit-message file. Tests: `python3 -m unittest discover -s tools/credit -t tools/credit -p test_credit.py` (run by `tools/ci/scripts.sh`).
