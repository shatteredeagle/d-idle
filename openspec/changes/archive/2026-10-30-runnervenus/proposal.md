## Why

Windows installer and portable builds (blowlink, golau, nms-expo, ...) use `windows-latest` and have no runner. The user approved a per-user forge runner on his PC windows-worker (2026-10-30).

## What Changes

- per-user, hidden, BelowNormal runner on windows-worker (no admin tooling, portable Go and MinGit), see docs/process/ci-runners-plan.md "Applied: a Windows worker"
