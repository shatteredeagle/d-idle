<!-- One topic per PR. Text here is read by people and by tools; keep it factual. Guide: docs/contributing/agent-work.md -->

Closes #ISSUE (the work item this PR delivers)

## What this adds
(one or two lines: the files and the question they answer)

## Evidence checklist
- [ ] every claim is tagged V (say how it was verified), S (secondary source) or U (unverified); no guess is presented as a fact
- [ ] every number can be reproduced with the command given in the document (case count and seed stated)
- [ ] every `V run` number has, in this PR, the script (under `tools/emu/` or `tools/`) or the exact command of an existing one, plus seed and case count; a count of game.exe against my own formula says "self-check, not compared with d2d"
- [ ] Authored as myself: my commits carry my own git identity (not `Claude` or another tool); an AI helper is named in an `Agent:` trailer and my research doc has an `Author:` line
- [ ] deviations from d2d (D..) and findings for the bot (B..) are listed apart, with the limits I did not cover
- [ ] no game code pasted: no disassembly, no decompiled listing, no data tables from the game files (address tags like `FUN_xxxxxxxx` and short formulas only)
- [ ] no game files, binaries, saves, keys, tokens, hostnames or home paths
- [ ] no changes outside the files the issue names (in particular nothing under `tools/` unless the issue asks for it)
- [ ] `go run ./cmd/leakcheck -mode fail` and `tools/ci/gamecode_guard.sh` are clean on my branch (or I say why not)

## How to reproduce
(exact commands, the harness you used and where it came from)

## Limits and open questions
(what this does not show)

## Agent use (optional)
Agent: (name or model, if an AI agent wrote this) / reviewed by a human: yes / no
