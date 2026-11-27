# Difficulty readiness and the opt-in hop (diffready, diffbaal)

> status: current 2026-11-27, offline only. Design: [design/difficulty-transition-2026-11-27.md](design/difficulty-transition-2026-11-27.md) section 4. Cards TASK-1058 (diffready) and TASK-1059 (diffbaal).

User decisions 2026-11-27: Hell has no resist gate (clvl, Baal dead and the survival model decide); the bot never goes back down; the hop is opt-in: the bot shows "ready for X" and waits for the owner's go, it never hops alone.

- Data: `internal/diffready/difficulty_ready.json`, per target and class. Nightmare rows copy the SME "Enter Nightmare" values (S); Hell rows are U placeholders (Nightmare life x1.4, clvl 75) and have no resist field.
- Predicate: `diffready.Table.Check(target, Hero)` returns a verdict with `Missing` (blocks) and `Unknown` (unread, shown, does not block). Unknown class or difficulty is an error naming it. Resists use `d2.ResistTable.Effective` (penalty, then cap), as `leveler.ResistGate`.
- Baal: `quest.BaalDead(reader, diff)` reads the `eve_of_destruction` word (U: flag position not read live). The runner reads the same word through its QuestView.
- Wiring, knob `bot.diffReadyV2` (default 0, ledger row in data/darklaunch.json): `wantedDifficulty` and the goal chain (`taskChain` gate) use the predicate; the hop needs `Runner.GrantDifficultyGo(target)` (one shot). The line shows in `ExplainView.Difficulty` (ops why).
- The player's go (card diffgo, bot.diffReadyV2 only, no new knob): operator verb `d2bot ops go-difficulty <nightmare|hell> --why ".."` (audited, rate limited, internal/ops) and the Return screen button "Ready for Nightmare: Go" (POST /api/difficulty/go on the answer listener; shown only when ready, else the missing list; busy feedback from the shared toast.js dBusy) both call `Runner.RequestDifficultyGo`, which refuses with the reason unless ready and for the next difficulty only. The go is kept in `state/diffgo.json` until the hop consumes it or the difficulty moves on. `ExplainView.difficultyOffer` carries target, ready, pending, missing. U: not run live; no separate live-view panel yet (the Return screen is the only button).
- Not built: portal count, key skill level and merc alive are unread (listed as unread).

## Live checklist (not run)
1. On a throwaway character that killed Baal in Normal, read the word with `quest.BaalDead`: expect Done; on one that did not, Needed/Started. Record the raw word.
2. Set bot.diffReadyV2 1 on a sandbox; ops why shows "ready for nightmare ... waiting for your go" and no new game starts.
3. Grant the go; one Nightmare new game starts, verifyDifficulty passes; a second ready check does not hop again.
4. Check the post-penalty resists against the character sheet (TASK-216 basis).
