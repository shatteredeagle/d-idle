# Warp-arrival fix on the play sandbox (devp4), 2026-11-12

Consumer: docs/probes/fid/fid-warp-arrival.md (decomp12) and the upstream warp-arrival PR draft.

## Setup (V, commands run)
- Fork branch `fork-only/sandbox-warp` (private forge, pushed): `c418e235` (nocursor lineage = devp3) + `feba9539` cherry-picked (`World::warp_arrival` uses `Level::warp_spot`) + a hand-copied `Level::warp_spot` from `fid/warp-arrival-r1` (it lives in the Teleport commit `57d7c39b`, which conflicts on this lineage). Tip `f8ea366c`.
- Image `d2dbot:devp4` built on the worker host with `nice 19` (`~/play-build-20261108/build4.sh`), deployed with `up.sh feed` (volume and game mount untouched), then `tools/dev-update.sh --commit origin/main` (7d2b526e; mergeq #634 hourfix2 had failed its race gate, so not waited for). Hero not reset.
- Rollback: `sh ~/play-build-20261108/up.sh old` (devp3 + feed, the previous state; `up.sh stock` = devp1), then `tools/dev-update.sh --commit origin/main --to d2play-20261108`.
- ExitWalk: d2d already does it (`arrive` leaves `walking` set toward spot + ExitWalk, decomp doc Q1), so no fork change is proposed for that.

## Before (V, container log of devp3, 2026-11-11 20:01Z to 2026-11-12 18:42Z)
Blood Moor arrivals from the Den: 28, "arrival bounce" in Blood Moor: 27 (96 %); 26 further bounces back in the Den. Bounces total 54 lines, cumulative counter 105 arrivals / 48 bounces. Arrival spot (136,140) or (145,141).

## After (V, 18:48Z to 19:16Z, 24 arrivals)
- Total: 24 arrivals, 2 bounce lines (both one episode, 19:06:48Z). All 22 other arrivals (Stony Field, Passage, Cold Plains, Den entries) stayed.
- Blood Moor from the Den: 2 arrivals, 1 bounced (arrived (135,141), left from (138,141) after 1.5 s; the bot had just run "stepping away from a cave mouth" (140,141) -> (160,145), a first click 20 subtiles east; the bounce was 0.3 s later), the second arrival at (136,142) stayed. The Den re-entry at (144,151) then also bounced because the bot's `nav exit` to area 2 was still pending (a bot action, not placement).
- The spot is different from before (135,141 / 136,142 against 136,140), so the fix is active. N is far too small for a rate: 1 of 2 against 27 of 28. Hero healthy, no errors, nav traces normal.

## Reading (U)
The fix does not by itself remove the Blood Moor bounce: the one bounce still came 0.3 s after a first click that points at the cave body, matching the decomp doc's open point (the box side of the arrival). Next evidence needs more Den visits (about 2 per 30 min now); the sandbox keeps devp4, and `grep '"arrival' ` on the container log gives the rate (cumulative counters reset at each bot restart). Not done: 10 bounce cases with the return-warp unit position (only 1 occurred).
