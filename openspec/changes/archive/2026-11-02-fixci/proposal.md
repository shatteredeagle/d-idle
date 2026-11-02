# fixci

## Why

Public CI Scripts job failed after the board cleanup: three archive rows in docs/community/work-archive.md cited research files under names that do not exist (sk-gear-ar-trace, sk-curse-calc-trace, sk-defender-resist-trace, sk-shield-damage-trace; real names sk-gearar-trace, skcurseval-trace, skdefenders-trace, skshield-trace), and the chunker groom tests failed because the mechanical-flag excerpt of a BACKLOG line tripped the fail-closed privacy scan (five short words in a row look like a CD key), so the proposal was never written (exit 2). Fix: correct the four paths; the flag no longer quotes the BACKLOG line.

## What Changes

- docs/community/work-archive.md: four research paths corrected.
- tools/contrib/chunker.py: the mechanical flag for a BACKLOG line no longer quotes the line.
