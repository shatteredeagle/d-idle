# Backlog.md web board on the NAS (read-only mirror)

> status: prepared, NOT deployed. Files: `deploy/backlog-board/`. Tags: V verified (how), S secondary, U unverified.

The board is for viewing. Task edits stay in git through agents; nothing written through the web UI survives.

## Design
- One container (`backlog-board`): a shallow, credential-free clone of the public repo (main only) in `/mnt/user/appdata/backlog-board/data/repo`; every 5 minutes `git fetch` + `git reset --hard FETCH_HEAD` + `git clean -fdx`. Mirror lag is up to 5 minutes. V: ran `entrypoint.sh` against the real public repo with a stand-in nginx, a deleted task file was restored by the next sync, 21 tasks served.
- `backlog browser` has NO read-only flag and NO host flag (V: `backlog browser --help` on 1.55.3 lists only `--port`, `--no-open`, `--non-interactive`), and it binds 127.0.0.1 only (V: `ss -ltn`). It has no authentication (S: README).
- So nginx in the same container listens on 6420 and forwards only GET/HEAD to Backlog.md (6421, loopback); POST/PUT/PATCH/DELETE get 403. That makes the board read-only at the edge; the hard reset is the second layer. V for the sync, U for the nginx/Docker build (no Docker on the dev box; the first build on the NAS is the test).
- Backlog.md is restarted when the mirror head moves, because it appears to read tasks from git, not the work tree (V: deleting task files in a mirror did not change the served list; U whether it caches across commits).
- Hardening: unprivileged user (Unraid nobody:users), read-only root filesystem, `cap_drop ALL`, `no-new-privileges`, 512 MB, 128 pids, no privileged flags. The binary is the pinned release `v1.55.3`, sha256 checked at build (V: the digest GitHub shows and a local `sha256sum` agree).

## Files
- `deploy/backlog-board/Dockerfile`, `entrypoint.sh`, `nginx.conf`
- `docker-compose.yml`: variant A, bridge network, port 6420 published (6420 is unused by the running containers, V: container list on 2026-11-05).
- `docker-compose.br0.yml`: variant B, own address on br0 for Nginx Proxy Manager.

## What the NAS has today (read-only findings, 2026-11-05)
- Compose Manager plugin is installed (V: plugin list). Most apps were converted to Docker-tab templates on 2026-10-15; Immich is the compose example (S: operator notes). Compose Manager is fine for this stack because it builds from a Dockerfile.
- Appdata lives under `/mnt/user/appdata/<app>`; per-app bridge networks exist. NPM, Authelia and Homepage run on `br0` with their own addresses; NPM cannot reach the NAS's own address or bridge addresses (S: operator notes), which is why variant B exists.
- Names: `*.birbhub.com` resolves to NPM; `<x>.lan.birbhub.com` records point at the target host itself (S). Authelia is the central login (`one_factor`, default deny).

## Choose a variant
- Direct LAN use only (simplest): variant A, then open `http://<nas-host>.lan.birbhub.com:6420`. No login: acceptable on the LAN only, never forward this port.
- Behind NPM + Authelia (recommended, the board has no login of its own): variant B.

## Steps in the Unraid web UI (variant A: 9 steps; B adds 6)
1. Docker tab, Compose section (Compose Manager): "Add New Stack", name `backlog-board`.
2. Edit the stack's compose file: paste `deploy/backlog-board/docker-compose.yml`, save.
3. Settings of the stack: autostart on.
4. "Compose Up" (builds the image from the public repo, a few minutes). Watch the log in the popup.
5. Docker tab: `backlog-board` is listed, state healthy after about 90 s. Add it to the Folder View folder "Infra".
6. Logs of the container should show `serving <commit>` and `Backlog.md browser interface running`.
7. Verify: `curl -fsS http://<nas-host>.lan.birbhub.com:6420/api/tasks | grep -o '"id":"TASK-[0-9]*"' | sort -u | wc -l` prints 21 or more.
8. Open `http://<nas-host>.lan.birbhub.com:6420`: the board loads. Drag a card: the move is refused (403) and nothing changes.
9. Mirror check: after a push to main, the task shows up within 5 minutes (or restart the container).

Variant B extras (NPM + DNS + Authelia):
10. Replace the compose file with `docker-compose.br0.yml` and set `ipv4_address` to a free br0 address (check the Docker tab for taken ones; UniFi's DHCP pool starts at .150). Compose Up again.
11. Technitium (web console): zone `lan.birbhub.com`, add A record `backlog` pointing at that address. This is the only place the address is written; everything else uses the name.
12. NPM: new proxy host `backlog.birbhub.com` (the wildcard already resolves to NPM), forward to `http://backlog.lan.birbhub.com:6420`, LAN-only access list like the other hosts, certificate as for `home.birbhub.com`. Websockets on.
13. Authelia: in `configuration.yml` access_control add a rule for domain `backlog.birbhub.com` with `one_factor`; paste the forward-auth snippet in the proxy host's Advanced tab, the same one the Homepage host uses. Restart Authelia.
14. Homepage (optional): add a tile pointing at `https://backlog.birbhub.com` in the services file.
15. Verify through the proxy: the page redirects to the Authelia login, then loads the board.

## Update the pinned version
Edit `BACKLOG_VERSION` and `BACKLOG_SHA256` in the Dockerfile (get the digest from the release page, check with `sha256sum`), plus the `image:` tag in the compose file; merge; in Compose Manager use "Compose Up" with rebuild (or "Update Stack"). Re-check that `backlog browser --help` still has no host flag and that nginx still blocks writes (step 8).

## Remove completely
Compose Manager: "Compose Down" then delete the stack; Docker tab: delete the image `backlog-board:1.55.3`; delete `/mnt/user/appdata/backlog-board`; remove the Folder View entry, the NPM host, the Authelia rule, the DNS record and the Homepage tile (variant B). Nothing else on the NAS is touched.

## Risks
- No authentication in Backlog.md: variant A is LAN-only by never forwarding the port; for anything beyond that use variant B behind Authelia.
- Mirror lag up to 5 minutes; a failed fetch keeps the old mirror (the log says so).
- The build pulls from GitHub main of the public repo; a broken merge to `deploy/backlog-board` breaks the next rebuild only, not the running container.
- Backlog.md's config in the mirror has `remote_operations: true`; in a read-only mirror this only affects how it reads other branches (U).
- The 403 on edits shows as an error toast in the UI; there is no way to hide the edit buttons (no flag).
