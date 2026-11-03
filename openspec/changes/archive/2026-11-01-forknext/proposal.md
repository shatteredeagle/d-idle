## Why

Upstream d2d lets a Nova or Frost Nova ring strike one monster 4 to 64 times per cast where game.exe strikes it once (NEXTHIT: 84,779 of 100,198 differential cases differ). Every soak or drill kill number measured with those skills on a stock or lab-debug-skills binary is inflated. User standing authorisation 2026-10-31: functionality may be added to the private fork when documented and vetted, never in an upstream PR without approval.

## What Changes

- Private fork: branch `lab-nexthit` (from `lab-debug-skills`) carries the NEXTHIT claim commit and the differential test mode; lab binary `bin/d2d.lab-nexthit`. No new debug verb.
- This repo: `docs/forknext.md` (what the fork carries, vetting, how runs select the binary, naming rule, how to drop it). Docs only, no patch text (GPL, docs/decisions.md).
