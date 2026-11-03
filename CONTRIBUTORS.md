# Contributors

Credit follows git authorship plus this file. AI agents are credited by the `Co-Authored-By` trailer in the commits of this repo and by the `Agent:` line in upstream pull requests; the rules of each upstream project decide how credit is written there. How credit works day to day: the "Credit" section of [docs/process/chunker.md](docs/process/chunker.md). The rows below come from the repo and its GitHub pull requests only.

## This repo

| handle | traced or built | where | status |
|---|---|---|---|
| shatteredeagle | owner; the bot, the emulator harness, the research docs and the contributor work board | this repo (`git log`) | merged |
| chuckado | PALAURA: Paladin aura filter, pulse damage, Prayer mana and state 85 | [PR #19](https://github.com/shatteredeagle/d-idle/pull/19) | open, in review |
| chuckado | PALDUR: 1.14d durability loss per hit and block | [PR #22](https://github.com/shatteredeagle/d-idle/pull/22) | open, in review |
| chuckado | PALSKILL: Smite, Zeal and Sacrifice dispatch in game.exe | [PR #23](https://github.com/shatteredeagle/d-idle/pull/23) | open, in review |

## Upstream

This work builds on the decomp of [psi29a/d2decomped](https://github.com/psi29a/d2decomped) (d2d). Our pull requests merged there: 24.

- [psi29a/d2decomped #177](https://github.com/psi29a/d2decomped/pull/177), merged 2026-11-02 (merge commit 3e073888; 11 commits, 21 files): vendor repair gate, melee fine physical D1 and undead/Blunt D2, freeze rules, standing missile footprint and burst roll, Smite lands on a block, each with game.exe differentials.
- chuckado: credited with a `Traced-by:` trailer on the Smite commit of #177 (upstream merge commit 3e073888) for the finding in [PR #23](https://github.com/shatteredeagle/d-idle/pull/23) (PALSKILL). Consent: the owner approved this credit personally.
