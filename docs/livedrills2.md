# livedrills2: RT.leave and D8.h/D8.h2 live on d2d (2026-11-04)

> status: current. Tags: V = ran live, S = secondary, U = unverified. d2d lab fork binary d2d.lab-cheats3 on the worker host, hidden slot 3 (free, capacity OK), the throwaway character, not game.exe. Run script: docs/probes/livedrills2/2026-11-04-run.sh (`labrun <worker> -- bash docs/probes/livedrills2/2026-11-04-run.sh LABEL GROUP ONLY REPEAT`). Evidence: docs/probes/livedrills2/2026-11-04-*.{txt,json}. No drill or bot code was edited.

| drill | repeat | verdict | numbers | tag |
|---|---|---|---|---|
| RT.leave (P29) | 1 | PASS | decision chicken at life 30/300, acted, left true, ticks to act 0, to effect 0 (limit 200), life floor 30 (above 0), dead false; character back in town life 40/40 | V |
| RT.leave | 2 | PASS, runs identical | two Save and Exit then rejoin cycles in one invocation; the rig reconnected to the new d2d process each time ("leave: re-entered the game (err nil)"); slot 3 free afterwards | V |
| D8.h (P18 paid hire) | 1 | FAIL | `no hire line in Kashya's menu`; setup gold 5000 applied, gold 5000 -> 5000, no merc | V |
| D8.h2 (cant afford) | 1 | FAIL | same error before the quote step (wanted ErrHireCantAfford); gold 1 -> 1, no merc | V |

## RT.leave notes
- The old d2d process quits by design; the log shows the control socket missing right after the exit (WARN "channel not answering"), then the new process and "re-entered the game". Each rejoin took about 1 s (V). The "end 0" in "life floor 30 end 0" is the drill's end-of-fight field after the process swap; the END line shows life 40/40 (S, not examined further).
- Cheats used (SETUP, in the drill, ledgered with reasons): debug belt clear, clearinv, goto 2, kill, spawn zombie1, stat 6/7 (life/mana).

## D8.h first deviating step
Walking to Kashya worked (the drill reached the hirer and opened the menu). The menu the hirer offers has exactly three lines: `Kashya` (400,115), `talk` (400,133), `cancel` (400,148). There is no `hire` line, so `hire_lab.go` stops before any quote. No quote, no class code, and so no confirmation of 0A or IW: still U. Hypothesis (U): in 1.14d the Act 1 hire window sits behind the `talk` line (or the menu shows hire only for a character state this one lacks); needs a follow-up that clicks `talk` and reads the next menu or the window. The right fix is in cmd/d2drill/hire_lab.go (out of scope here: no code edits).

## Writes
`writes after start: not measurable`: cmd/d2drill has no `runstart` path (grep found none), so the run was not marked. These are drill runs, not measured runs. Writes were only SETUP inside the drill before each action (`debug goto`, `debug stat 14` gold 5000 or 1, and the RT.leave setup verbs above), each ledgered by actor drills:<id>; nothing wrote during the action steps by design (S, from the drill code, not audited against the ledger).
