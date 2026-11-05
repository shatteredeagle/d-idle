# Backlog.md web board on the NAS (read-only mirror)

> status: DEPLOYED 2026-11-05 as an NAS Docker-tab template (variant B: own br0 address behind Nginx Proxy Manager and Authelia). Files: `deploy/backlog-board/`. The Compose Manager sections below are SUPERSEDED. Tags: V verified (how), S secondary, U unverified.

The board is for viewing. Task edits stay in git through agents; nothing written through the web UI survives.

## Design
- One container (`backlog-board`): a shallow, credential-free clone of the public repo (main only) in `/mnt/user/appdata/backlog-board/data/repo`; every 5 minutes `git fetch` + `git reset --hard FETCH_HEAD` + `git clean -fdx`. Mirror lag is up to 5 minutes. V: ran `entrypoint.sh` against the real public repo with a stand-in nginx, a deleted task file was restored by the next sync, 21 tasks served.
- `backlog browser` has NO read-only flag and NO host flag (V: `backlog browser --help` on 1.55.3 lists only `--port`, `--no-open`, `--non-interactive`), and it binds 127.0.0.1 only (V: `ss -ltn`). It has no authentication (S: README).
- So nginx in the same container listens on 6420 and forwards only GET/HEAD to Backlog.md (6421, loopback); POST/PUT/PATCH/DELETE get 403. That makes the board read-only at the edge; the hard reset is the second layer. V for the sync, U for the nginx/Docker build (no Docker on the dev box; the first build on the NAS is the test).
- Backlog.md is restarted when the mirror head moves, because it appears to read tasks from git, not the work tree (V: deleting task files in a mirror did not change the served list; U whether it caches across commits).
- Hardening: unprivileged user (NAS nobody:users), read-only root filesystem, `cap_drop ALL`, `no-new-privileges`, 512 MB, 128 pids, no privileged flags. The binary is the pinned release `v1.55.3`, sha256 checked at build (V: the digest GitHub shows and a local `sha256sum` agree).

## Files
- `deploy/backlog-board/Dockerfile`, `entrypoint.sh`, `nginx.conf`: the image (built on the NAS, tag `backlog-board:1.55.3`).
- `deploy/backlog-board/my-backlog-board.xml`: the NAS template (network br0, `<br0-ip>` placeholder, hardening flags in Extra Parameters, 512 MB, appdata mount, image healthcheck).
- `deploy/backlog-board/deploy.sh`: build + template + `tpl-apply.sh`; idempotent; `BOARD_IP=<br0-ip> deploy/backlog-board/deploy.sh` (host from `tools/env.sh`).
- `docker-compose.yml`, `docker-compose.br0.yml`: SUPERSEDED, kept for reference only. Do not start them: that would create a duplicate container.

## Accepted method (deployed 2026-11-05)
The NAS runs its containers as Docker-tab templates (`my-<name>.xml` in the dockerMan user templates, applied with `/mnt/user/appdata/_NAS/tpl-apply.sh NAME`), not as Compose Manager stacks. Locally built images are built with `docker build`, then `tpl-apply.sh`. So:
1. Pick a free br0 address: list the addresses of running containers (`docker inspect`) and grep the template files; do not ping. Take the next one above the highest in use and below the DHCP pool. NPM runs on br0 and cannot reach the NAS's own address or bridge addresses, so this app needs its own br0 address (V: the other proxied apps work this way).
2. `BOARD_IP=<br0-ip> deploy/backlog-board/deploy.sh`. V: ran twice on 2026-11-05; the container went healthy in under a minute, the image build passed the Backlog.md checksum, limits as templated (read-only root, cap_drop ALL, 512 MB, 128 pids, user 99:100).
3. Folder View: add `backlog-board` to the "Infra" folder in `/boot/config/plugins/folder.view/docker.json` (back the file up first; append to that folder's `containers` list with `jq`). V.
4. DNS (Technitium, via the `labapi` helper): A record `backlog.lan.<domain>` -> `<br0-ip>`, `overwrite=true`. This is the only place the address is written. V: `dig` against the DNS server returns it.
5. NPM (via `labapi npm`): proxy host `backlog.<domain>` -> `http://backlog.lan.<domain>:6420`, own Let's Encrypt certificate (HTTP-01, like `home.<domain>`), SSL forced, HTTP/2, block exploits, websockets on, no access list; Advanced config = the same LAN-only include plus Authelia forward-auth block as the Homepage host (the `AUTHELIA` string in the portal repo's `deploy/npm-hosts.py`). V: the host's config was read back.
6. Authelia: add `- domain: backlog.<domain>` / `policy: one_factor` to `access_control.rules` in the live `configuration.yml`; back it up first, run `authelia validate-config --config <new file>` inside the container, then replace the file and restart only that container. V: validation passed; the auth page and `home.<domain>` answered as before afterwards. The portal repo's copy of the file was synced to the live one.
7. Homepage tile: not added (the portal repo's Homepage files had uncommitted work of other sessions at deploy time). Optional follow-up.

Verification of 2026-11-05 (V):
- LAN: `curl -fsS http://<br0-ip>:6420/api/tasks | grep -o '"id":"TASK-[0-9]*"' | sort -u | wc -l` printed 21.
- Write refused: a POST to `/api/tasks` returned 403.
- Via the proxy: `https://backlog.<domain>/` answers 302 to the Authelia login with `rd=` back to the board. The step after the login (the board loading behind Authelia) was not exercised: it needs the user's own login (U).

## Update the pinned version
Edit `BACKLOG_VERSION` and `BACKLOG_SHA256` in the Dockerfile (get the digest from the release page, check with `sha256sum`) and the image tag in `my-backlog-board.xml` and `deploy.sh`; merge; run `deploy.sh`. Re-check that `backlog browser --help` still has no host flag and that a POST still returns 403.

## Roll back a version
Re-run `deploy.sh` from the previous commit (it rebuilds the old pinned tag and recreates the container). The data folder is only a disposable mirror.

## Remove completely
`docker rm -f backlog-board`; delete the template `my-backlog-board.xml` from the dockerMan user templates and the image `backlog-board:1.55.3`; delete `/mnt/user/appdata/backlog-board`; remove `backlog-board` from the Folder View entry, the NPM proxy host and its certificate, the Authelia rule (then validate and restart Authelia), the DNS record (`labapi technitium zones/records/delete`) and any Homepage tile. Nothing else on the NAS is touched.

## Risks
- No authentication in Backlog.md: variant A is LAN-only by never forwarding the port; for anything beyond that use variant B behind Authelia.
- Mirror lag up to 5 minutes; a failed fetch keeps the old mirror (the log says so).
- The build pulls from GitHub main of the public repo; a broken merge to `deploy/backlog-board` breaks the next rebuild only, not the running container.
- Backlog.md's config in the mirror has `remote_operations: true`; in a read-only mirror this only affects how it reads other branches (U).
- The 403 on edits shows as an error toast in the UI; there is no way to hide the edit buttons (no flag).
