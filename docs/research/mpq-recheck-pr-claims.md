# Recheck of the two upstream PR data claims with the patched 1.14d tables

Status: V. Method: docs/process/mpq-data-reading.md; both stacks (with and without the lowercase `patch_d2.mpq` symlink) read and diffed on the worker host, 2026-10-30. PR sources: `~/d2d-lab/pr/PR_DESCRIPTION-price.md` (branch fix/vendor-prices, 4884b1d) and `~/d2d-lab/pr/cast/PR_DESCRIPTION.md` (fix/cast-timing, b0d932d).

## Price PR

| Claim | Old 1.10 | Patched 1.14d | Verdict |
|---|---|---|---|
| books.txt tbk (Town Portal) BaseCost 250, CostPerCharge 50; BookSpellCode tbk, ScrollSpellCode tsc | 250 / 50 | 250 / 50 | MATCHES |
| books.txt ibk BaseCost 200, CostPerCharge 40 | 200 / 40 | 200 / 40 | MATCHES |
| misc.txt tbk cost 250 (stack 1..20, spawn 5), ibk 200, tsc 100, isc 80 | same | same | MATCHES |
| misc.txt aqv cost 256 (min 100, max 350, spawn 200), cqv 256 (60/250/150) | same | same | MATCHES |
| ItemTypes bowq Quiver = bow, xboq Quiver = xbow (book, scro empty) | same | same | MATCHES |

Arithmetic holds: 250+4x50 = 450, 350x256/1024 = 87 then the PR's x960/1024 = 81 (the 960 factor is the Charsi multiplier per the PR, not re-derived here). No change needed to the price PR text, tests or inputs. One note: the PR describes the ItemTypes Quiver "+0xe" offset; in the patched file Quiver is column 10 (old file: 13), so any cross-check by column index must use the patched layout; by header name nothing changes.

## Cast PR

| Claim | Old | Patched | Verdict |
|---|---|---|---|
| animdata SC frames SO 14, AM 20, NE 16, PA 16, BA 14, AI 17, DZ 15 at speed 208, others 256 (records SOSCHTH, AMSCHTH, NESCHTH, PASCHTH, BASCHTH, AISCHTH, DZSCHTH) | identical | identical (file byte-identical, 3529 records) | MATCHES |
| Skills.txt: Fire Bolt, Charged Bolt, Fire Ball, Nova, Teleport, Thunder Storm `anim` SC | `cast`, seqnum 0 | SC, seqtrans SC, no seqnum | MATCHES |
| Lightning, Chain Lightning `anim` SQ, `seqtrans` SC, seqnum 12 | `seq`, 12, no seqtrans column | SQ, SC, 12 | MATCHES |
| Inferno seqnum 6, Arctic Blast 18, `seqtrans` SQ | seq 6 / 18 | SQ/SQ, 6 (seqinput 10) and 18 (seqinput 10) | MATCHES |
| "1.14d's Skills.txt has no seqtrans column" (PR text, Not-checked list, code tag) | true of the 1.10 table we were reading | FALSE: patched 1.14d Skills.txt (358 rows) has `seqtrans` and `seqinput` | DIFFERS (text only) |

The PR took `seqtrans` values from the 1.13 table at fabd/diablo2 because our local read showed no such column; that was the wrong-stack artefact. The 1.14d values equal the 1.13 ones, so formulas, tests and numbers are unchanged. What must change in the PR: delete "1.14d's has no `seqtrans` column" and the "1.14d Skills.txt itself (the seqtrans values are the 1.13 ones)" not-checked bullet, cite `data\global\excel\skills.txt` of the 1.14d patch as the V source (V for Lightning/Chain Lightning/Inferno/Arctic Blast rows above), and update the `unverified` tag/row for the sequence-FCR rule to say the seqtrans data is verified (the FCR rule itself from D2MOO stays unverified). The edit goes into ~/d2d-lab/pr/cast and was not made here.

Not rechecked: charstats (neither PR text relies on it); the in-game tick counts and the game.exe addresses (not table data).

## Other docs that read tables with mpq-cat

On master only two docs name it: `docs/d2d-gaps.md` (section (a): misc.txt prices, stack sizes, npc.txt multipliers) and `docs/research/d2moo-fit.md` line 113 (a plan to use it, no data read). The misc.txt values are identical in both stacks, so d2d-gaps (a) stands (the `npc.txt` multipliers 512/1024 were not re-read here: U). `docs/sme/cast-timing-sources.md` (branch castsources) read animdata.d2 from the 1.14d MPQ: byte-identical in both stacks, OK; its line 151 "1.10 vs 1.14 data diffs were not examined" is now answered: animdata none, Skills.txt gained `seqtrans`/`seqinput` and 37 rows. VERIFY-0003's own tables were read with the symlink (its notes). Docs that read other tables (monstats, skills, books layout, itemtypes by column index, charstats, levels) with mpq-cat and no mention of the symlink should be re-read: none found on master; branches that mention mpq-cat: only d2d-gaps.md and d2d-moofit's copy of d2moo-fit.md (same lines). Docs that did not name the tool (d2d-fidelity.md, d2d-combat.md, d2d-town.md) have no mpq-cat reads by grep; where they quote table values, ask the author which tool read them.
