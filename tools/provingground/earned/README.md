# Earned saves (card pgearned, TASK-756)

`milestones.json` is the catalogue of export points (`den-done`, `clvl-12`, `cain-freed`, `before-andariel`). The saves themselves are binaries and live on the builder under `$PG_EARNED_DIR/<milestone>/<class>/` with a `manifest.json` (`internal/scenario/earned.go`, format `didle.pg-earned/1`); nothing from a save is committed. Rules: `docs/process/proving-ground.md` section 11.
