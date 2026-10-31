# Domain docs: single-context

One project, one vocabulary. Hand-written in the shape of the setup skill's `domain.md` (the skill was not run).

## Before exploring, read these

In this order, skipping silently what does not exist:

1. `GLOSSARY.md` (repo root): the project's terms, each with where it lives and what it is not to be confused with. There is no `GLOSSARY-MAP.md` (single context).
2. `openspec/specs/<capability>/spec.md` for the area you touch: the contracts and numeric gates (soak-gates, death-rule, backend-contract, fidelity-gate, merc-policy, early-game-policy, lab-rules). Numbers live there, do not copy them.
3. `docs/adr/` (index: `docs/decisions.md`): the decisions the user made. Read the ones that touch your area.
4. `docs/plan/STATUS.md`, then `docs/plan/plan-v2.md`, then the doc of your area (`docs/INDEX.md` maps them). Evidence docs carry V (verified) / S (secondary) / U (unverified) tags.

In an agent run the read order of `AGENTS.md` comes first (`openspec/README.md`, STATUS, your change card); this file only adds where the domain vocabulary lives.

## File structure

```
/
├── GLOSSARY.md            vocabulary
├── AGENTS.md              agent rules (the only root rules file; no CLAUDE.md)
├── openspec/specs/        contracts, numeric gates
├── openspec/changes/      change cards (the work tracker, docs/agents/issue-tracker.md)
├── docs/adr/              decision records, MADR style, 0001 upward
├── docs/decisions.md      one-line index of the ADRs plus the dated verbatim entries
└── docs/                  evidence, designs, per-area docs
```

## Use the glossary's vocabulary

Name a domain concept with the term as `GLOSSARY.md` defines it, and do not drift to a synonym it lists under "Not to be confused with". A concept you need that is missing is a gap: note it in your report, do not add it yourself.

## Flag ADR conflicts

If your output contradicts an ADR, say so explicitly ("Contradicts ADR-0005, worth reopening because ...") instead of silently overriding.

## Who writes these files

Agents never create or edit `GLOSSARY.md` or `docs/adr/*` on their own. Only the user's own grill session (`docs/process/grill-workflow.md` when merged) or an explicit coordinator brief that names the file may. Skills that would write them (`domain-modeling`, `grill-with-docs`, `improve-codebase-architecture`) put the proposed text in the report instead. Rule text: `AGENTS.md`, section "Agent skills".
