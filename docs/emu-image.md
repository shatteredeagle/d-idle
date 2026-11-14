# The game.exe emulator image

> status: current (2026-11-14, TASK-406). Rebuilt after the Orion cleanup of 2026-11-13 removed the volatile scratch copies. Tags: V = run or checked on the worker host this session.

## What it is

"The emulator" is upstream d2decomped's `tools/emu` (Python, unicorn): it maps the 1.14d `Game.exe` PE image and calls game.exe's own functions natively, with the patched MPQ stack read through `mpq-cat`. Nothing is run under Wine and no game window opens (V). The old scratch copies lived in `/tmp` (`/tmp/ug-emu`, `/tmp/casttrace`, `/tmp/casttrace-venv`) or in per-job scratch dirs; the image replaces all of them.

The image holds NO game data in this repo; it lives only on the worker host and is built from the user's own installed game files, read-only.

## Where (persistent, NOT /tmp)

`$D2_EMU_DIR`, default `~/d2d-lab/emu` on the worker host. About 1.6 GB, of which about 1.4 GB are hard links of the MPQs in `~/d2d-lab/data` (no extra bytes while both names exist); the rest is 174 MB (venv, upstream checkout, `.cache` of extracted tables, mpq-cat). **The Orion cleanup must keep this directory** (and `~/d2d-lab/data`, `~/d2d-lab/src`, `~/d2d-lab/build-decomp12`, `~/d2d-lab/d12`). There was no keep-list in docs/process; this section is it.

| Path in the image | What | Source (read only) |
|---|---|---|
| `mpq/*.mpq`, `mpq/patch_d2.mpq` | the MPQ stack d2d and the emulator read (`D2_MPQ_DIR`) | `~/d2d-lab/data` (hard links; `patch_d2.mpq` is the lowercase name mpq-cat needs) |
| `mpq/bin/game.exe` | the emulator's PE (`emu.py` reads `$D2_MPQ_DIR/bin/game.exe`), mode 444 | `Game.exe` of the user's 1.14d install (`D2_GAME_DIR`) |
| `casttrace/game.exe` | the same file under the old cast-trace name (`--exe` arguments, `ls_run.sh`) | same |
| `patch/LODPatch_114d.exe` | mpq-cat reads the 1.14d tables out of it (`D2_PATCH_INSTALLER`) | `~/Downloads/LODPatch_114d.exe` |
| `upstream/` | git clone of upstream d2decomped pinned to `8a3cb678`; `tools/emu/.cache` fills on first use | `~/d2d-lab/pr/gaps`, else `~/d2d-lab/src`, else the public URL |
| `upstream/build/tools/mpq-cat/mpq-cat` | table reader the emulator calls | copied from an existing upstream build |
| `venv/` | python with unicorn 2.1.4, pefile, capstone | pip (the unicorn wheel from `~/d2d-lab/src` if present) |
| `env.sh`, `SHA256SUMS` | exports for the consumers; the hash list below | written by `emu_setup.sh` |

## Rebuild from scratch (under 10 minutes; V: setup took 6 s with the sources present)

```
labrun <worker host> -- bash tools/emu/emu_setup.sh setup      # idempotent; refuses a /tmp D2_EMU_DIR
labrun <worker host> -- bash tools/emu/emu_setup.sh verify     # sha256 + imports + the XP grid oracle (about 5 s)
labrun <worker host> -- bash tools/emu/emu_setup.sh env        # the exports
```

Overrides: `D2_EMU_DIR`, `D2_GAME_DIR`, `D2_DATA`, `D2_PATCH_SRC`, `D2_UPSTREAM_SRC`, `D2_UPSTREAM_REF`, `MPQCAT` (header of the script). If no built mpq-cat exists, `setup` stops and prints the cmake line (needs network for StormLib). The d2d-side probes (`zd_build.sh`, `d12_build.sh`, `*_build.sh`) additionally need a scratch d2d build; those trees and builds are NOT part of the image (they stay under `~/d2d-lab/wt-*`, `build-*`).

## Hashes (sha256; V, `emu_setup.sh verify` checks them)

```
631066c1649c4ea9ffe48bf97e24c00bca1f7a6759c21150f1a79982589adaaf  mpq/bin/game.exe  (= casttrace/game.exe; md5 2b3600c1...)
5ee5b6f09c338324db14ea3ec03a4b8a9b36e4eea249bb12f1fb592558d6434d  patch/LODPatch_114d.exe
531bad76d520eea8cb0601bea5a40e1ec52f51fe7b94a6892dc483d18170c607  mpq/Patch_D2.mpq
8ba07beff306231a50c170d50ac53dfd64c824ab5c9c590702563e34ce90ef61  mpq/d2data.mpq
04e35bffeae8d9959042832055702f8b5b4567ca5505ff1df0cf3fd6d5beb6c2  mpq/d2exp.mpq
41ee24239edba191e7d8d0b1ce35510ee95c17af264e09e35f15c34b5ba7b879  mpq/d2char.mpq
6c16ebc0ade4a54dd1afda0c3465d175099d6b17ff58c38350be6b6e7c2f8e34  upstream/build/tools/mpq-cat/mpq-cat  (a build artefact; will differ after a rebuild)
```

## Consumers and the variable they read

`D2_EMU_DIR` (default `~/d2d-lab/emu`) is read by 184 `tools/emu/*.sh` files and 18 `*.py` (the former `/tmp/ug-emu` becomes `$D2_EMU_DIR/mpq`, `/tmp/casttrace/game.exe` becomes `$D2_EMU_DIR/casttrace/game.exe`, the venv paths (`walk2-venv`, `casttrace-venv`) become `$D2_EMU_DIR/venv`, `UP` (upstream checkout) becomes `$D2_EMU_DIR/upstream`, `D2_PATCH_INSTALLER` becomes `$D2_EMU_DIR/patch/...`, `D2_BUILD` of the emulator becomes `$D2_EMU_DIR/upstream/build`). Fixed in particular: `zd_*`, `sk_tele_spot` family (`run-py.sh`), `ms_*`, `mpq-table.sh`, `ls_run.sh`, `d12_*`, `vb2_*`, `mc2_*`, `pets_merc` (via `run-py.sh`), `xpg_model`/`xp_award` (via `ms_run.sh`), `run-on-worker.sh` (no longer exports a `/tmp` data dir). The two C++ probes `d12_arrive_probe.cpp` and `vb2_mana_probe.cpp` fall back to `$D2_EMU_DIR/mpq`. Not touched: `shr_*` (on branch decomp17, same pattern, use `mpq-table.sh`/`run-py.sh` from here) and `.py` files that only name the old paths in a docstring example.

Still on main with older /tmp names only in comments or docs (harmless). Anyone's job can `source $D2_EMU_DIR/env.sh` to get `D2_MPQ_DIR`, `D2_PATCH_INSTALLER`, `UP`, `EMU`, `PY`, `MPQCAT`, `GAMEEXE`.

Verification record: `docs/research/emu-reverify-2026-11-14.md`.
