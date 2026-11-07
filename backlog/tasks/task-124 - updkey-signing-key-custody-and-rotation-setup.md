---
id: TASK-124
title: 'updkey: generate the signing key and set up the CI environment, secret and tag protection'
status: Proposed
assignee:
  - user
created_date: '2026-11-07T23:28:00Z'
labels:
  - ops
  - chore
  - tester
milestone: m-1
dependencies: []
priority: high
ordinal: 117000
card: "updkey"
owns:
  - "docs/self-update-keys.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Decided 2026-11-07 (docs/research/self-update-design.md, Decisions): the Ed25519 private key is stored as a CI secret, scoped to a GitHub Actions environment with the user as required reviewer; not kept offline. Blocks updrelease signing and Tier 2. Docs go in docs/self-update-keys.md (rotation needs an image rebuild; revocation by signed REVOKED file and minimum version).

User steps (GitHub settings, repo shatteredeagle/d-idle):
1. Generate the key pair on your own machine with `go run ./cmd/d2sign keygen` (after TASK-121 lands the tool). Keep an encrypted backup copy of the private key; never commit it.
2. Settings > Environments > New environment, name `app-release`. Tick Required reviewers and add yourself. Optionally restrict Deployment branches and tags to the protected release tags.
3. In that environment, Environment secrets > Add secret `D2SIGN_PRIVATE_KEY`, paste the private key. Do NOT add it under repository secrets.
4. Settings > Rules > Rulesets > New tag ruleset: target tags `tester-v*` and `release-*`; enable Restrict creations, Restrict updates, Restrict deletions (bypass list empty or just you).
5. Account Settings > Password and authentication: confirm 2FA is on.
6. Hand the public key to the image build (committed file for the Dockerfile to bake).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 key pair generated offline by cmd/d2sign on the user machine; private key pasted into the `app-release` environment secret (not repo-wide), backup place named, public key handed to the image build (no secret in the repo)
- [ ] #2 GitHub settings done: environment `app-release` with the user as required reviewer, tag ruleset on `tester-v*` and `release-*`, 2FA confirmed; docs/self-update-keys.md states rotation (image rebuild needed) and revocation (signed REVOKED file, minimum version)
<!-- AC:END -->
