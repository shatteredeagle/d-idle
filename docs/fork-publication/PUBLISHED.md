# d.idle fork: published (phase 2b)

Date: 2026-11-07, on the user's explicit go ("Rename it back and use a standalone repo"). Licence GPL-3.0-or-later (upstream's LICENSE is in the tree). No game data.

## Public repo

https://github.com/shatteredeagle/d2d-didle : a STANDALONE public repository (not a GitHub fork; repo id R_kgDOVmS7Kg). Upstream: https://github.com/psi29a/d2decomped. Default branch `didle/main`.

| Ref | Hash | Source |
| --- | --- | --- |
| branch `didle/main` | 0c4f302ebad6e260b91afaa93fc9038af1ab3b7b | private `d2d.lab4-overlay` ece607de1832141fdc7867d48b90ee9ac9f398e0 plus one commit adding `README-DIDLE.md` |
| branch `didle/lab3-labverbs` | f75816c616c8253930d88277b90936e3eabe3bc8 | private `d2d.lab3-labverbs` |
| branch `didle/lab2-freezeresume` | ee5e06d8ef990200b699e61779c64ded1451c33d | private `lab2-freezeresume` |
| annotated tag `didle-gate2-lab3` | -> f75816c616c8253930d88277b90936e3eabe3bc8 | Gate 2 lab3 build |
| annotated tag `didle-gate2-lab2` | -> ee5e06d8ef990200b699e61779c64ded1451c33d | Gate 2 lab2 build |
| annotated tag `didle-overlay-1` | -> ece607de1832141fdc7867d48b90ee9ac9f398e0 | overlay release |

Exactly these 3 branches and 3 tags exist (V: `git ls-remote` after the push; no upstream tags, no other branches).

## README divergence

`README.md` at the root is upstream's, unchanged. `README-DIDLE.md` (from FORK-README.md in this directory, with the real URL and pinned hashes) exists only on the public `didle/main`; the private `d2d.lab4-overlay` does not have that commit (public is one commit ahead). Do not cite 0c4f302e as a private-branch hash.

## Rename incident

The earlier phase-2 command `gh repo fork --fork-name d2d-didle` renamed the user's existing fork shatteredeagle/d2decomped (id R_kgDOVdR0RA, the home of the upstream-PR branches) to d2d-didle. It was renamed back the same day (V: `gh repo view` shows the same id, still a fork of psi29a/d2decomped, public, default branch `main`, 84 refs as before); nothing in it was pushed or changed. The `shatteredeagle/d2decomped` strings in docs/research/upstream-prs.md and related notes are correct again.

## Tester image source offer

Tarball: `git archive --format=tar.gz --prefix=d2d-didle-<short>/ -o d2d-didle-<short>.tar.gz <tag>` from a clone of the public repo at tag `didle-gate2-lab3` (or the tag the image is built from); sha256 goes into SOURCE.txt (docs/tester-licence.md section 3). GPL source-offer URL: https://github.com/shatteredeagle/d2d-didle (the tag named in SOURCE.txt).

## Outside check (V, 2026-11-07)

Fresh `git clone --depth 1 -b didle/main`: LICENSE present; no .mpq/.dc6/.dcc/.d2s/.d2x/.dll/.exe files; sweep with the cmd/leakcheck patterns (domain, machine, person, home path, lab path, secret shapes): 0 hits in our content. Matches that exist are upstream's own: example LAN addresses in docs/READMEs (192.168.x, e.g. README.md line 134), one upstream maintainer address in a workflow, a public Sentry DSN in apps/launcher/CMakeLists.txt, and game text "jade figurine".
