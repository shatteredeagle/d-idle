## Why
The owner approved CodeRabbit as an in-house pre-scan on 2026-11-02: review our own bundles on the d-idle repo and on a staging PR on the downstream d2decomped fork before the one real PR goes upstream, so the maintainer's hourly review allowance is spent once on a cleaned bundle.

## What Changes
- `.coderabbit.yaml` revised and schema-validated (path filters, path instructions for research docs, tools, Go code).
- `openspec/changes/crsetup/fork-coderabbit.yaml`: the fork config, kept here and applied to the fork's main only.
- `docs/process/coderabbit-prescan.md`: install steps, quota test, bundle workflow, CLI alternative.
No GitHub writes by the agent.

Overlap (declared): `.coderabbit.yaml` was added by the communitywork card (outside-PR gate); crsetup revises it.
