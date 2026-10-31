# d2d container: licensing of the image

> status: summary and open question; the full analysis is `gpl.md` (NOT legal advice). Decision belongs to the user's grill 0002.

- The d2d binary is the private GPL-3.0-or-later fork. Fork source, patches and the built binary stay out of this repo. The container build bakes the binary in as a LOCAL build artifact only (`deploy/d2d-container/context.sh` puts it in a scratch context dir; nothing is committed, no registry push, no `docker push`).
- `licenses/` in the context (GPL-3.0 text, upstream LEGAL.md, `SOURCE.txt` naming fork ref and commit and vendored pins) is copied to `/usr/share/licenses/d2d/` in the image, next to the binary. `SOURCE.txt` states "NOT FOR DISTRIBUTION" until the question below is answered.
- Game data (d2data.mpq etc.) is never in an image or context: mounted read-only at `/game`; `context.sh` aborts if it finds a `.mpq` or `game.exe` in the context, `.dockerignore` excludes them.
- Our Go binaries sit in their own directory and layer (`/opt/d2lod`), talk to d2d over a unix socket as separate programs (position of `gpl.md`, not legal advice).

## Open question for grill 0002 (not decided here)
If the image is given to anyone else (a friend, a public registry, a NAS community template), GPLv3 section 6 requires offering the Corresponding Source of exactly that binary (fork commit, vendored pins, Dockerfile, CMake flags) to every recipient, plus the licence text. Options: publish the fork repo with a tag per image, ship a source tarball beside the image, or keep everything private (no duty). Also open: licence of our own Go code, the Freepik icon terms compiled into the binary, a notices file for Ubuntu packages. Until answered: image stays on the worker host and Tower, never pushed.
