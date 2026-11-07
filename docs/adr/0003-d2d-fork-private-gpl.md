---
status: accepted (amended 2026-11-07: the fork is published as a public GitHub fork, see the Amendment below)
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

## Amendment 2026-11-07 (user approval 2026-11-07, "approve to publish the fork")

The original text above stays as written. This amends the Decision Outcome: "private forge repo" and "no distribution until a GPL reviewer is named".

1. The d.idle fork is published as a public GitHub fork of psi29a/d2decomped under the user's account, repository `d2d-didle`, GPL-3.0-or-later like upstream. Branches: `didle/main` (the d2d.lab4-overlay lineage), `didle/lab3-labverbs` and `didle/lab2-freezeresume` (provenance of the Gate 1 and Gate 2 binaries), with tags; the plan is in docs/research/fork-publication-scan.md.
2. The fork is offered to upstream. The maintainer keeps upstream a clean reimplementation until 1.0, so the fork's additions (lockstep, dump verbs, overlay, lab verbs) are not upstream PRs now. Stand-alone fidelity fixes still go upstream as drafted PRs, each posted only on the user's go.
3. Lab and cheat verbs ship flag-gated (`--lab-verbs`) and off by default; they are not stripped. Test cheats never go into an upstream PR without the user's approval.
4. Game data (MPQ, DC6, D2S, table files) is never in the repo; the fork carries source only. No d2d code is added to the d2lod-bot repo (unchanged).
5. The forge repo `<owner>/d2d-fork` stays the private dev remote; the same-day push rule for every fork or lab patch stands, and a doc may cite a commit only when it is on the forge remote or the public fork.
6. The publication happens only after a clean history scan (credentials, private hosts and paths, private information, Blizzard-owned data); the user's go is conditional on that. No GPL reviewer is named; that question stays open and is the user's. An image given to a tester still needs the user's go on the day of the first copy (docs/tester-licence.md).

Source: user decision 2026-11-07 (popups: "approve to publish the fork", "Yes: draft and record it now"); docs/research/fork-publication-scan.md; docs/tester-licence.md. Amends decisions.md "Tester build" (2026-11-06) where it says ADR 0003 needs an amendment first: this is that amendment.
