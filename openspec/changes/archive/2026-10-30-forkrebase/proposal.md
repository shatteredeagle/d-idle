## Why

Our lab binaries sit on an old upstream (3306e60) whose kill_exp is wrong from monster-level diff 6 up (docs/research/xp-award-trace.md); upstream also merged our price and cast PRs, so the lab branch carries redundant patches. lab6 re-applies only the still-needed lockstep, dump and verb patches on current upstream main.

## What Changes

- fork branch lab6 (private the forge <owner>/d2d-fork) and binary d2d.lab6 on the worker host; lab5 stays
- docs/d2d-fork.md gets a lab6 section, docs/d2d-lab6.md holds the verification record
