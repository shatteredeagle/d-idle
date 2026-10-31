# Draft PR: drlg-dump: open the folder's 1.14d patch like the game does

> status: draft, NOT filed (agent LUTPROOF, 2026-10-31). Branch `fix/lut-gholein-preset-rng` in the upstream clone `~/d2d-lab/pr/lut` on the worker host, commit 632d5cc2 on upstream/main 32768ddb. Nothing pushed. Body below follows upstream's `.github/pull_request_template.md`; commands and paths are the reviewer's (their own copy), not ours. Proof: docs/research/lutgholein-proof.md; go / no-go: docs/research/pr-lut-gholein-evidence-pack.md.

---

## What changes

`tools/drlg-dump` opened only `$D2_PATCH_INSTALLER` (then d2exp.mpq, d2data.mpq), never the `patch_d2.mpq` sitting in the MPQ folder, so without the installer it ran on CD-era data. `data\global\excel\monpreset.bin` is only in the 1.14d patch: with it missing, `ds1_units` kept every DS1 monster's raw id, none matched `rolled_unit`, `stays` never stepped the level seed, and Lut Gholein's rooms (two Act2Vendor monsters roll to stay at generation, FUN_00667620 through FUN_00667970) were each seeded two level-seed steps late; the `units` dumps of every town listed raw ids. The tool now opens the MPQs as `open_game_data` does: `$D2_PATCH_INSTALLER` when set and present (a note when set but missing), else the folder's `patch_d2.mpq` or `LODPatch_114d.exe`, then the base MPQs. game.exe reads the patch first; the d2d game already did.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_00667620` | the preset-unit roll to stay: a monster with id 0xcc, 0xcd, 0x173 or 0x174 (byte table at `0x6677dc`, cases at `0x6677d4`) steps the seed at `0x667721..0x667744` and stays when `low % 3 == 0`; MonPlace 0x21..0x23 and objects 0xc4 / 0x105 / 0x245 take their own rolls; every other unit is kept without a step |
| `FUN_00667970` | calls it on the level seed at generation for a LvlPrest row with Scan or Pops (Lut Gholein: Scan 1, Pops 3), before the rooms are allocated |
| `FUN_0066b3e0` | room allocation steps the level seed once per room (`0x66b451`) and seeds the room from its low word (`FUN_00650e40`) |
| `0x744304` + 0xa80 | the MonStats row count the roll compares the id against (`0x6676f3`): the id is the MonPreset-mapped MonStats id, which a DS1 only yields through `monpreset.bin` |
| emulator trace | `lut_trace.py 161464580 40` (memory-write watch on level +0x1c4, nothing hooked out): memset, two seedings (`FUN_00650e40`), two writes at `0x667744`, 49 at `0x66b451`; no file roll on the level seed for level 40 (the act layout sets its file) |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
# the tool against game.exe's own DRLG, with no installer in the environment and patch_d2.mpq in $D2_MPQ_DIR:
cd tools/emu
env -u D2_PATCH_INSTALLER D2_PATCH_INSTALLER= uv run python diff_drlg.py 1-20 40 tiles units seeds    # before: 0/20 each; after: 20/20 each
env -u D2_PATCH_INSTALLER D2_PATCH_INSTALLER= uv run python diff_drlg.py 1-10 2 tiles units           # units before 6/10, after 10/10; tiles 10/10 both
env -u D2_PATCH_INSTALLER D2_PATCH_INSTALLER= uv run python diff_drlg.py 1-10 1 units                 # 0/10 -> 10/10 (raw ids before); same for 75, 103, 109
# the mechanism: data\global\excel\monpreset.bin is in the patch only
build/tools/mpq-cat/mpq-cat "$D2_MPQ_DIR" 'data\global\excel\monpreset.bin' | wc -c                  # 920 with the patch, "not found" from d2exp + d2data alone
# the two steps in game.exe (the trace script is attached to the PR conversation, not committed):
uv run python lut_trace.py 161464580 40
```

The claim is conditional, not that the gate fails in general: `drlg-dump` opens only `$D2_PATCH_INSTALLER`, while the game's loader opens the data folder's patch; without the installer env the differential gate fails or mismatches (level 40 seeds 1-20: 0/20 before, 20/20 after), and with `D2_PATCH_INSTALLER` pointing at the installer it already passes 20/20 on unpatched upstream main. The patch makes `drlg-dump` open the folder's patch like the game.

(`diff_drlg.py` sets `D2_PATCH_INSTALLER` to a Downloads path when it is unset; an empty value means "the folder's patch" after this change, and was "MPQ open failed" before.)

## Screenshots

None: nothing visible changes in the game; the dumps do.

## Checklist

- [ ] A test in `tests/`: none added; the change is in a tool (`tools/` needs CODEOWNERS review). The re-check above is upstream's own differential.
- [ ] `docs/research/re/drlg.md` fidelity rows for Lut Gholein and the towns' units were only true with the installer in `$D2_PATCH_INSTALLER`; no row moved. A `tools/drlg-dump/README.md` line on the patch lookup can follow if wanted.
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN against upstream/main)

Limits: with `$D2_PATCH_INSTALLER` set and present the stack is the one it was (a no-op for a setup with the installer); with it set to a missing file the tool used to exit ("MPQ open failed") and now notes it and falls back. One tool, two call sites (the plain / tiles / units / seeds dump and `layout`); the other modes already went through `load_game_data`. The in-game generator was not re-measured here (its loader opens the folder's patch). The trace is one map seed; the differential 20 seeds for level 40 and 10 each for the other four towns and Cold Plains. `presubmit.py --quick` warns about the throwing `fs::exists` overloads: the same call the existing d2exp / d2data line makes; a missing path is `false`, not a throw.

---

Agent: Claude Code
