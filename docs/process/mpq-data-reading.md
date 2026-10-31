# Reading 1.14d tables with mpq-cat (trap and recipe)

Status: V (reproduced 2026-10-30 on the worker host, read-only, scratch dir deleted afterwards).

## The trap

`mpq-cat` (`~/d2d-lab/src/build/tools/mpq-cat/mpq-cat` on the worker host, usage `mpq-cat [-d OUT [-n BYTES]] <mpq dir> <file>...`) looks for the 1.14d patch layer by the exact lowercase name `patch_d2.mpq` in the MPQ directory. Our copy `~/d2d-lab/data/` has `Patch_D2.mpq`. Without a lowercase name the tool does not warn: it serves the CD-era (1.10) tables. The game binary `d2d` is NOT affected (it finds MPQs case-insensitively through `install::find_file`, and logs `Loading: Patch_D2.mpq`), and neither is our Go reader (`internal/data/extract.go` matches both spellings). Only `mpq-cat` is.

Symptoms of the wrong stack (V, both stacks diffed):
- `monstats.txt` first line starts `Class namco Type ...` and Andariel has MinHP 666. Patched: first line `Id hcIdx BaseId ...`, Andariel minHP 2562.
- `skills.txt` has 321 rows and no `seqtrans`/`seqinput` columns (`anim` values are words `cast`/`seq`). Patched: 358 rows, `anim` values `SC`/`SQ`, `seqtrans` column present.
- `books.txt` header ends `SpellOffset ... SkillScrollOffset SkillBookOffset BaseCost CostPerCharge`. Patched: `pSpell ... ScrollSkill BookSkill`.
- Column positions move (misc.txt `cost` is column 11 old, 12 patched; ItemTypes `Quiver` 13 old, 10 patched): never read by index, always by header name.

## Recipe (never touch the game directory)

```sh
ssh worker.example
S=$(mktemp -d); for f in ~/d2d-lab/data/*; do ln -s "$f" "$S/$(basename $f)"; done
ln -s ~/d2d-lab/data/Patch_D2.mpq "$S/patch_d2.mpq"        # the missing lowercase name
M=~/d2d-lab/src/build/tools/mpq-cat/mpq-cat
$M "$S" 'data\global\excel\skills.txt' > skills.txt         # stdout; or -d OUTDIR to write files
rm -rf "$S"
```

Binary files (`data\global\animdata.d2`) come out byte-exact the same way. Alternative: set `D2_PATCH_INSTALLER` to `LODPatch_114d.exe` (mpq-cat tries it first, see its header comment); not used here.

## One-line sanity check (run it before trusting any read)

```sh
$M "$S" 'data\global\excel\monstats.txt' | head -1 | cut -f1-2     # patched: "Id<TAB>hcIdx"; wrong stack: "Class<TAB>namco"
```

Equivalent: `skills.txt` must contain a `seqtrans` header, or Andariel minHP must be 2562 (not 666).

## What the wrong stack did NOT break (V, 2026-10-30)

`animdata.d2` is byte-identical in both stacks (md5 7d59e8c3...), and misc.txt prices/stack sizes, books.txt CostPerCharge, and the ItemTypes Quiver codes are the same in both. Only monstats, skills, books layout and similar patched tables differ. See docs/research/mpq-recheck-pr-claims.md for the PR claim table.
