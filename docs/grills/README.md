# Grills

Decision interviews run in a dedicated session ([workflow](../process/grill-workflow.md)). One folder per topic: `NNNN-slug/prebrief.md`. Topic is archived when its ADR is accepted and its cards are merged.

| # | topic | status | why it is next |
|---|---|---|---|
| 0001 | [d2d primary backend](0001-d2d-primary-backend/prebrief.md) | prebrief ready | everything else (container, soak plan) depends on it |
| 0002 | GPL / licence plan for the private d2d fork (stub) | resolved by ADR 0010 | decisions.md: nobody is named to review; blocks any image leaving the NAS or any upstream PR; see [gpl.md](../d2d-container/gpl.md), [ADR 0003](../adr/0003-d2d-fork-private-gpl.md) |
| 0003 | [Merc strategy and Act 2 entry](0003-merc-and-act2/prebrief.md) | prebrief ready; SME-grill done ([digest](0003-merc-and-act2/sme-grill.md)), claims UNVERIFIED ([claims-to-verify](0003-merc-and-act2/claims-to-verify.md)) | the Monastery route and Andariel stay blocked without a merc or clvl 20; [ADR 0006](../adr/0006-free-rogue-merc.md), [merc.md](../merc.md), [act2.md](../act2.md) |
| 0004 | Container / NAS plan (stub) | to write | follows 0001 and 0002: ports, viewer, one image or fleet; [d2d-container](../d2d-container/README.md), [ADR 0001](../adr/0001-container-end-goal-native-dev.md) |
