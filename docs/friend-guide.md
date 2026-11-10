# d.idle friends pack: install guide (Linux)

> status: BUILT offline (TASK-274, 2026-11-10); the install script was run once on the worker host in a scratch folder with a throwaway project, ports and fake game files (see the card result), never by a friend. Plan: docs/research/friend-build.md. Windows and macOS are not supported in this version.

This is a private beta for a few friends. The pack is a single file, `didle-friends-<version>.tar.gz`, sent to you by a person you know. Do not post it anywhere.

## What you need

- Linux on x86-64 (the image is amd64 only), 3 GB of free disk space, 4 GB of RAM to spare.
- Docker Engine with the compose plugin (`docker compose version` must work) and permission to run `docker` without sudo.
- Your own copy of Diablo II: Lord of Destruction, patched to 1.14d. The installer needs these four files in one folder: `d2data.mpq`, `d2exp.mpq`, `d2char.mpq`, `Patch_D2.mpq` (case does not matter). The folder is mounted read-only; nothing is written to it.

## Licence note (read before you install)

The pack carries the GPL-3.0 program d2d and the bot that drives it, with their source (`src/`) and licences (`licenses/`). It contains nothing from Blizzard Entertainment: no game files, no art, no keys. Art shown on the watch page is read from your own game folder on your own machine at run time; never post screenshots of it or share the data volume. Diablo II is a trademark of Blizzard Entertainment; this project is not affiliated with it. If you pass the pack on, the source and notices go with it. The installer shows `licenses/licence-note.txt` and asks for a `y` once per version. This is a plain description, not legal advice.

## Install

```
tar xzf didle-friends-<version>.tar.gz
cd didle-friends-<version>
./install.sh --game-dir /path/to/your/Diablo-II
```

It checks Docker, the download (`SHA256SUMS`), your game folder and free ports, loads the image, starts the container, waits for the page and prints one address, for example:

```
http://localhost:8891/setup.html?t=<your token>
```

Open exactly that address. The token in it is your password for the page; do not share it. `./didle status` prints the address again at any time. The page address must be the one printed (the bot only accepts answers from that origin; a different host name gives a 403). The setup asks you for a hero name, class and goals, then there is a 1 to 3 minute warm-up before the hero walks out.

Options: `--port N` (watch page on N, answers on N+1, for when 8891 is busy), `--host NAME` (also publish on the LAN and accept the page at `http://NAME:PORT`), `--project NAME` (compose project name, default `didle`).

## Every day

| Command | What it does |
|---|---|
| `./didle status` | version, container state, update state and the page address with your token |
| `./didle stop` | stops and saves the hero (takes up to 45 s); your hero lives in the Docker volume `<project>_d2dbot-data` |
| `./didle start` | starts it again |
| `./didle update` | asks the update source for a signed release; shows the version; applies it after you type `y` |
| `./didle report` | writes `didle-report-<time>.tar.gz` (logs, container state, update state; tokens, your home path and the game path replaced) to send back when something breaks |

Update with the hero standing in town: the bot is stopped to swap the release, and an update while the hero fights has not been tested. An update is only installed when its signature matches the key built into the image; a failed start rolls back to the previous release automatically. Your hero and token are never part of a release. The only network call the bot makes is the update check to the source set in `.env` (`D2BOT_UPDATE_FEED`); `D2BOT_UPDATE_CHECK=off` in `.env` (then `./didle stop && ./didle start`) switches it off.

## When something goes wrong

| Symptom | Do this |
|---|---|
| `STOP: missing in ...` | point `--game-dir` at your own Diablo II folder with the four files |
| `port 8891 is in use` | `./install.sh --port 9100` |
| page says 401 | open the address from `./didle status` (it carries the token) |
| answering a card gives 403 | you opened the page under another name than the printed one; use the printed address, or reinstall with `--host NAME` |
| the page does not come up | `./didle report` and send the file |

## Remove it

`./didle stop`, then `docker compose down` in this folder removes the container. The volume (your hero) stays until you run `docker volume rm <project>_d2dbot-data`. `docker rmi` the image to free the disk.

## Building the pack (publisher)

`tools/friendpack/build.sh --version X.Y.Z --image NAME:TAG --d2d-src FORK_SOURCE.tar.gz --d2d-commit SHA --out DIR [--feed https://...]` writes `didle-friends-X.Y.Z.tar.gz` and its `.sha256`. It saves the image with `docker save`, adds the bot source of the shipped commit and the d2d fork source tarball you give it, licences, THIRD_PARTY.md, a written offer, the install script, `./didle` and this guide, refuses any game, art or key file, and writes `SHA256SUMS` (d2sign signs release manifests only, so the pack itself is covered by the checksums and the private route it travels; the real signing key stays outside the repo and is never used here). The pack is reproducible (sorted names, zero mtime, `gzip -n`). Handing it over is a private act by the user; nothing here publishes anything.
