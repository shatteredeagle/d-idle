## Why

Live soaks (i5 Fresha, G2 on 2026-10-30) show trade visits that fail after 8 clicks and 30 to 45 s: the dialog flag stays set and no click, from 30 px above to 15 px below the modelled line, changes anything. 18 such visits (15 G2, 3 Fresha) plus Gheed first clicks that miss by 5 px.

## What Changes

- pickEntry stops laddering pixels when the dialog ignores clicks: slow-hover retry, then ends the visit with a clear log line within a 9 s budget
- Gheed gets a +5 px first adjustment (V from the logs)
- docs/npc.md records the evidence and the live checklist
