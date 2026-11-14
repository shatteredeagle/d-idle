# H. Open questions for the user

> status: waiting for answers (TASK-416, 2026-11-14). Each can be answered in a word. The recommendation is what the design assumes until answered.

| # | Question | Options | Recommendation | Why |
|---|---|---|---|---|
| 1 | **What is the primary command surface?** | hybrid / map-first / bar-first | **hybrid**: the Orders panel is primary, the act map is its place picker | One of the eight daily commands is spatial; acts 2 to 5 have no map data; the queue is a list either way (C.1). Map-first costs the act 2 to 5 map data first and a larger map tile |
| 2 | Is the local minimap ever a command input? | no / later | **no** | No order kind for a point; it would fight pathing and safety (C.2) |
| 3 | Which layout does a new hero (a friend's first run) start on? | simple / standard | **simple** | Eight numbers instead of sixteen; the onboarding card and Journey strip stay on it |
| 4 | When does Standard become the default for everyone without a stored layout? | after your review of S7 / with S1 / never | **after S7** | Nobody's screen changes until you have lived on it; a stored layout is never touched either way |
| 5 | Is the zone-order queue ("Add to queue") free for everyone, or does it stay an unlock-ladder reward? | free / unlock | **free** | A control surface that hides its queue behind a reward reads as broken. The code has it as an unlock today (`unlockorderqueue.go`) |
| 6 | Pause: what does the hero do when paused in the field? | walk to town then stop / stop where safe / town only | **walk to town then stop** | A hero standing still in a field dies; `safeToPause` exists but a route does not. This is bot work (card B1) and needs your rule |
| 7 | "Sell now" and "Revive mercenary" as player commands (bot work, cards B2 and B3)? | both / sell only / neither | **both** | They are in your own list of commands; both are small errands over the existing town trip |
| 8 | Do the old tiles ever get retired (stubs and aliases per D.3), or do they stay forever for Classic and Debug? | stay / retire later | **stay** | Zero risk to the promise; the cost is dead weight in `watch.html` and `app.js` |
| 9 | The freeze on new /watch panels runs to 2026-11-17 (decision 2026-11-10). Does this redesign start after it, or does it lift it? | after / lifts | **after** | The slices add eight panels; S1 (shell only, no panel) could start before |
| 10 | Tile look: borderless tiles with small caps labels (the mockup), or keep today's bordered cards with serif titles? | borderless / bordered | **borderless** | Fewer lines on screen; the serif stays for the brand. It applies to the new panels only, so Classic keeps its look |
| 11 | Should Needs you raise a desktop notification or sound for red rows? | no / red only / all | **red only**, off until switched on | The alert switches exist today in the Alerts card; they move to the menu |
| 12 | Clock times on the page: the viewer's own zone, or always Eastern? | viewer / eastern | **viewer** | A friend elsewhere reads local time; your own browser is Eastern anyway |
