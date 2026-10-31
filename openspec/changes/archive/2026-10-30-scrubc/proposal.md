## Why

Work package C of docs/process/public-split.md: the committed text must not name a private machine, person, host, IP, home path or domain before the repository goes public.

## What Changes

- Prose and docs use role words: the worker host, the NAS, the head box, the forge, a Windows worker, a Mac worker, the status board, the maintainer. Technical content (cores, RAM, OS, constraints) is kept.
- Hosts become worker.example, nas.example, forge.example, board.example; the domain becomes `<domain>`; home paths become `~` or `<user>`; the LAN IPs become role names or documentation addresses.
- the deploy directory named after the NAS product is renamed deploy/nas; archived card ids that carry a machine name (two ids, see leakcheck-allow.txt) are kept because changecheck ties ids to commit subjects: rename them at the fresh-history cut.
- ADRs, decisions.md and GLOSSARY.md: only leak-class strings were replaced, no wording of a decision changed (the coordinator named these files for that purpose).
- leakcheck-allow.txt gets one-line reasons for the false positives (game terms, MIME type, OAuth grant name); the manifest is regenerated (no scrub-needed entry is left).

## Out of scope

The ops repo (the heading "Running a soak on the NAS" in its docs/nas-container.md; the code comments already quote the new heading). leakcheck stays in warn mode.
