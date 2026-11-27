---
id: TASK-1042
title: >-
  palzeal: quest skill points spent on grant and the main attack selected by
  skill id on d2d
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T11:10:00Z'
updated_date: '2026-11-27T11:32:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 504000
card: "palzeal"
lane: next
owns:
  - "internal/leveler/"
  - "internal/plandata/build-paladin-zealot-lite.json"
  - "docs/plan/data/build-paladin-zealot-lite.json"
  - "internal/combat/palzeal_attack_test.go"
  - "internal/combat/fighter.go"
  - "internal/d2dcombat/input.go"
exclusive: []
contracts: []
result: "Not verified live (offline tests only). A build table skill entry may carry a quest key; such a point is bought only once the hero holds more skill points than its levels grant (Have.QuestPts from hard tree points plus unspent), so a missing Den never delays later rows and a granted bonus point is spent at once. Zealot-lite rows: clvl 4 Might, row 5 Might tagged den_of_evil, row 16 Zeal plus Holy Fire tagged radament. On d2d a right-hand attack skill with no hotkey yet is cast by its id (Teeth learned mid-game no longer waits for the next binding); Zeal was already swung by id. Build-specific: only the zealot-lite table data; other tables that fold a quest point into a row (paladin normal30 v2 row 5 and others) are not tagged yet. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: live 2026-11-27 the sandbox zealot-lite Paladin (Doofer) killed 4954 monsters with plain Attack and spent its clvl 12 point on Might, not Zeal: row 5 of the table assumed the Den of Evil point, so without that point every later row came one level late. Done when quest bonus points are tagged in the table and spent on grant, the rows count level points only, and the main attack on d2d is cast by skill id (left by swing, right by id) as soon as it is learned.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Quest-tagged table points are bought only with quest bonus points, so a missing Den of Evil never shifts later rows; a granted bonus point is spent at once
- [x] #2 Zealot-lite rows: clvl 4 Might, clvl 16 Zeal plus Holy Fire (radament), Den point tagged on row 5
- [x] #3 On the d2d swinger backend the main attack is chosen by skill id from the profile band (Attack before Zeal is learned, Zeal from then), tested on Paladin and Necromancer Teeth
- [ ] #4 Live: a fresh zealot-lite Paladin logs last_skill=Zeal at kills from clvl 12
<!-- AC:END -->
