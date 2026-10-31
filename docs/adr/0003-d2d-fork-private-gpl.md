---
status: accepted
date: 2026-10-30
decision-makers: user (decisions made in conversation), coordinator
---
# 0003: The d2d fork stays private on the forge; no d2d code in this repo; nothing goes upstream without the user

## Context and Problem Statement

We need lockstep, a dump verb and later vendor and layout verbs that upstream does not have. d2d is GPL-3.0, so copying its code into our repo or shipping the binary has licence consequences, and patches that live only on one machine can be lost.

## Considered Options

* Patch upstream first
* Vendor d2d into this repo
* Private fork repo, bot talks over the socket

## Decision Outcome

Private forge repo <owner>/d2d-fork (base e36922e); the bot is a separate program talking over the socket; our docs give own summaries with file-path citations; no distribution until the user names a GPL reviewer (none yet); upstream issues and PRs are drafted in the repo and posted only on the user go; every fork or lab patch is pushed to the forge the same day (AGENTS.md, spec lab-rules).

## Consequences

Fork maintenance and rebase cost; the container cannot ship the fork until the review is decided.

## Sources

decisions.md user answers 2026-10-30; docs/d2d-fork.md; docs/d2d-container/README.md
