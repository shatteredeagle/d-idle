---
id: TASK-121
title: 'updrelease: signed app release manifest, d2sign tool and release workflow'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-124
  - TASK-120
  - TASK-25
priority: high
ordinal: 114000
card: "updrelease"
owns:
  - "cmd/d2sign"
  - ".github/workflows/app-release.yml"
  - "docs/self-update-release.md"
  - "internal/selfupdate/manifest.go"
  - "internal/selfupdate/manifest_test.go"
  - "tools/release"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Tier 2 part 1 of docs/research/self-update-design.md sections 3.2, 3.3. Key custody decided 2026-11-07: the private key is an environment-scoped CI secret (see Decisions in the design); needs the updkey setup (TASK-124) before the first signed release.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 MANIFEST.json schema and asset layout written as a contract-style doc; cmd/d2sign (stdlib ed25519) keygen, sign, verify with a review summary; tests
- [x] #2 app-release.yml builds assets from a tag and uploads the assets; a signing job in the approval-gated `app-release` environment (required reviewer, environment-scoped secret) signs the manifest and publishes; signed `REVOKED` file and minimum version supported; nothing run or pushed
<!-- AC:END -->

result: offline only, NOT verified on GitHub. cmd/d2sign (keygen, manifest, revoked, sign, verify, summary), internal/selfupdate/manifest.go (schema, canonical marshal, revoke/min-version, keys, versions), tools/release/app-assets.sh (reproducible tarballs, checked by two runs), docs/self-update-release.md and a DRAFT inert .github/workflows/app-release.yml (workflow_dispatch only, needs repo variable APP_RELEASE_ENABLED, d2d build step intentionally fails until TASK-120 wiring). Owns widened to manifest.go, its test and tools/release (TASK-122 builds on manifest.go). AC2 is met as a draft: unrun, the environment/secret/reviewer are user steps (TASK-124).
