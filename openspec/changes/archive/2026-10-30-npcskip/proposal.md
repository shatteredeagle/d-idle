## Why

First-contact NPC greetings held the dialogue menu back and the bot waited out the speech, 13 to 90 seconds per visit. The user says a left click skips the talking.

## What Changes

A configurable skip strategy (`town.greetingSkip`: wait, npc, esc, ground) with a bounded click-then-verify loop, time-to-panel logging, bounded helpers for the Act 2 Harem guard and cinematics (not wired yet), and a live probe tool `cmd/npcprobe` to measure which strategy works on first contact.
