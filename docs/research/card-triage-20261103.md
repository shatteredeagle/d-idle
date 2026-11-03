# Open card triage, 2026-11-03

> status: proposal for the lead (card `applyreview`), docs only. 38 open cards at the start; 3 were archived by this job (merged branch and every task ticked): actgate, forknext, wpcooldown. goldfloor (3/3, merged) was left: its card says parked, delta spec not applied. the rest are in the table, the cap is 12 (G-F). Goals G-A..G-F: [../adr/0015-goals-acceptance-on-game-exe.md](../adr/0015-goals-acceptance-on-game-exe.md). Tags: V = read from the card or `git log` here (`git log main --merges --grep "Merge agent/<x>:"`); I = judgement. "merged" means a merge commit of the card's branch is on main (V); the branch itself is deleted for most. Recommendations are the lead's to apply; nothing below was archived except the three.

Key: KEEP stays open; MERGE-INTO folds the remaining task into another card then archives; ARCHIVE the card is finished or superseded; DROP not wanted under the new goals.

| card | status, tasks, merged | goal | rec | reason |
|---|---|---|---|---|
| armorhit | in-progress 6/7, merged | G-E | KEEP | last task is the PR claim; close with the upstream batch that carries it |
| buildv3 | in-progress 3/4, merged | G-B | ARCHIVE | open task is a clvl 19 to 30 soak no plan schedules; v3 default switch is deferred (ADR 0015); move the task to BACKLOG |
| d2d-container-soak | proposed 0/3 | none (deferred) | ARCHIVE | container deferred until Gate A (ADR 0015); revive after G-A |
| d2d-default-backend | proposed 0/3 | G-C | DROP | d2d is already the lab backend (ADR 0013); I: the default flip is trivial and not a goal (U whether already flipped) |
| d2dmesh | deferred 7/7, branch meshdesign not merged (V) | none | ARCHIVE | parked, tasks all done; lead merges the design branch or drops it |
| d2d-replicated-soak | proposed 0/3 | G-D, G-C | MERGE-INTO | the multi-seed Gate 1 card (plan S3) is the replicated soak with holdouts |
| drills6 | in-progress 4/5, merged | G-C | ARCHIVE | live rerun merged; last task a note, move to BACKLOG |
| fixbelt | in-progress 3/4, merged (2 merges) | G-C | MERGE-INTO | live proof is the multi-seed Gate 1 (S3); archive after it |
| fixcastgap | in-progress 4/5, merged | G-C | MERGE-INTO | same: remaining task is live proof, covered by S3 |
| fixmanamodel | in-progress 4/5, merged | G-C | MERGE-INTO | same |
| fixprogress | in-progress 2/3, merged | G-C | MERGE-INTO | same (progress criterion is in the Gate 1 judge) |
| fixskilllvl | in-progress 3/4, merged | G-C | MERGE-INTO | same |
| gate1run-20261101 | in-progress 2/3, no branch merge (records merged via gaterecords, V trailer) | G-C | ARCHIVE | record exists; seed-11 single runs are dropped |
| gate1run-20261102 | in-progress 2/3, branch unmerged (V) | G-C | ARCHIVE | same; lead decides whether to merge the branch first |
| goldfloor | deferred 3/3, merged | none | ARCHIVE | levers off by default after the failed A/B; the card was deliberately parked (result text), the lead applies or drops its unapplied delta spec first |
| ledgerkey | in-progress 2/3, no merge found (V) | G-F | KEEP | small cheat-ledger key fix; finish or archive when its branch state is checked |
| m1-liveproof | in-progress 13/15, merged | G-C | MERGE-INTO | M1 exit (15 C rows) is an input to the gate list, not a goal; fold open rows into the Gate 2 fix list |
| m1-trips | deferred 0/5 | G-C | DROP | trip trigger waste is minor next to gate stability; revisit if S3 shows it |
| m2-latebase | deferred 0/3 | G-D | DROP | a single-seed late baseline is dropped; the statistical A:B replaces it |
| m2-roam | deferred 0/4 | G-B | MERGE-INTO | the area-choice card (plan S4) |
| m2-tripwp | deferred 0/3 | G-B | MERGE-INTO | the area-choice card (plan S4) |
| m3-act1chain | deferred 0/3 | G-B | MERGE-INTO | one Act 1 card: m3-act1run |
| m3-act1run | deferred 0/3 | G-B | KEEP | the G-B card; promote when S3 and S4 pass |
| m3-andariel | deferred 0/3 | G-B | MERGE-INTO | m3-act1run (first on a cheat-levelled throwaway, then fresh seeds) |
| m3-merc | proposed 0/4 | G-B | KEEP | free Rogue is on the Act 1 path; needs a soak slot |
| m3-readiness-model | deferred 0/4 | G-B | MERGE-INTO | m3-act1run (route guard and Andariel floor replaced by readiness) |
| gameexe-retire-soak | deferred 0/2 | G-A | MERGE-INTO | becomes the Gate A card (game.exe 2 h, 8 h, 24 h); retirement no longer the aim |
| orig-decision-probes | deferred 0/2 | G-D | DROP | superseded by the A:B and Gate A on game.exe |
| mapitem | in-progress 4/5, merged | none (stream) | ARCHIVE | contiguous map merged; the live check is a stream nicety, not a goal |
| nexthit | in-progress 6/8, merged | G-E | KEEP | upstream batch item; close with the batch |
| skmeleewire | in-progress 4/5, merged | G-C | MERGE-INTO | live proof via S3 |
| skmiss | in-progress 5/7, merged | G-E | KEEP | upstream missile finding; close with the batch |
| skresistfix | in-progress 6/7, merged | G-E | KEEP | upstream curse fix; close with the batch |
| soak-minutes-virtual | in-progress 3/5, merged | G-C | KEEP | the Gate 1 judge fix the seed-set run depends on; finish before S3 |
| stream-grid | deferred 4/7 | none | DROP | parked; stream work is not a goal |
| projreview | in-progress 6/6, branch unmerged | G-F | ARCHIVE | merge then archive (lead); this job merged it into its own branch |
| applyreview | in-progress | G-F | KEEP | until merged |

Count after the cuts: KEEP 9 (armorhit, ledgerkey, m3-act1run, m3-merc, nexthit, skmiss, skresistfix, soak-minutes-virtual, applyreview) plus the new S3 and S4 cards, under the cap of 12. ARCHIVE, MERGE-INTO and DROP rows need the lead: archive needs the merge hash and the card's last tasks.

## Not archived by this job and why
Rule: merged branch AND every task ticked. Cards with open tasks (the fix cards, drills6, mapitem, upstream items) and cards with an unmerged or unknown branch (projreview, gate1run-20261102, d2dmesh, ledgerkey) were left. Cards with 0 tasks done were left even if the lead will drop them.
