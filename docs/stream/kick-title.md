# Kick title and category from the stream service

Code: `tools/kick/` (Python stdlib; `kick_common.py`, `kick_auth.py`, `kick_title.py`). Research and API facts: `docs/stream/kick-api.md`.
Status: built and unit-tested against a local mock server only. NOT run against the real Kick API (U: endpoint shapes are as documented
in kick-api.md; first real run may need small adjustments, e.g. whether the category `name` search is exact or fuzzy).

## One-time user steps

1. Kick account: enable 2FA (needed for the Developer tab).
2. Kick Account Settings, Developer tab: create an app. Redirect URL exactly `http://localhost:8765/callback` (use `--port` and the
   same port in the app if you change it).
3. Put the Client ID and Client Secret into the desktop keyring (same keyring and `service=d2idle` as the stream key; paste at the
   prompt, never into chat):
   ```
   secret-tool store --label="D.idle Kick client id"     service d2idle account kick-client-id
   secret-tool store --label="D.idle Kick client secret" service d2idle account kick-client-secret
   ```
4. Log in once, on the machine whose browser you will use (on headless the worker host forward the port: `ssh -L 8765:localhost:8765 worker`):
   ```
   python3 tools/kick/kick_auth.py login        # prints the authorize URL; open it, log in as the channel owner, approve
   python3 tools/kick/kick_auth.py status       # shows yes/no, scopes, expiry; never a secret
   ```
5. Create the config and choose the category (the default search is `Diablo II`; Kick may name it differently):
   ```
   python3 tools/kick/kick_title.py init
   python3 tools/kick/kick_title.py categories "Diablo II"     # prints id<TAB>name
   python3 tools/kick/kick_title.py choose <id>                # pins category_id in the config
   ```
6. Test with a real title, then enable:
   ```
   python3 tools/kick/kick_title.py set "D.idle: test title"
   ```
   Edit `~/.config/d2idle/kick.json`: `"enabled": true`.

`logout` removes the stored tokens (`kick_auth.py logout [--all]`; `--all` also drops client id/secret). To revoke on Kick's side
use the Developer tab.

## Config `~/.config/d2idle/kick.json` (no secrets in it)

| key | default | meaning |
|---|---|---|
| enabled | false | `set_title` returns `skipped` until true (the CLI `set` ignores it) |
| category_name | "Diablo II" | searched by exact case-insensitive name; ambiguous or no match fails with the candidate list |
| category_id | null | if set, used directly (no search); `kick_title.py choose` writes it |
| title_template | `D.idle: {name}, {class} {lvl}, {goal}` | fields: name, class, level, lvl (`lvl 12`), goal; empty fields vanish cleanly |
| max_title | 100 | U: Kick publishes no limit |
| port | 8765 | login callback port |
| api_base / id_base | Kick hosts | only for tests |

`D2IDLE_CONFIG_DIR` overrides `~/.config/d2idle` (tests). The category id cache is `kick-category.json` in that directory.

## How streamctl calls it

```python
sys.path.insert(0, "<repo>/tools/kick")
from kick_title import set_title, build_title
status = {"name": "Booti", "class": "Sorceress", "level": 12, "goal": "clearing Act 1", "act": 1, "difficulty": "Normal"}
res = set_title(build_title(status))      # Result(ok, error, skipped, title, category_id); NEVER raises
if not res.ok and not res.skipped: log/board "title not updated: " + res.error
```

Call it after starting the RTMP push (or in a daemon thread); it needs a few seconds worst case (10 s timeout per call, up to 3
attempts) and must never gate going live. Re-send on level/act change at most every few minutes.

## Behaviour

- Token upkeep: the access token and the rotating refresh token live in the keyring. When the access token is within 60 s of expiry,
  `get_access_token` takes an exclusive file lock (`kick.lock` in the config dir), re-reads the keyring (another process may have just
  refreshed), refreshes, stores the NEW refresh token first and the access token after, and only then returns it. If persisting fails
  the new token is not used. A 401 on the PATCH triggers one forced refresh.
- Re-login: refresh answered 400/401 (`invalid_grant`), or no refresh token: error text is
  "Kick login is missing or expired: run `python3 tools/kick/kick_auth.py login`".
- PATCH `/public/v1/channels` `{stream_title, category_id}`; 204 or 200 is success. Retries: 5xx and 429, 3 attempts, backoff 2 s then
  4 s, or `Retry-After` seconds (capped at 30 s). 403 is reported as a scope problem. Everything else returns `Result(ok=False)`.
- Secret hygiene: secrets go to `secret-tool` over stdin, never argv or files. Every log line and exception passes through `redact()`
  (all secret values seen in-process, plus `Bearer ...`). Response bodies are never logged; only the HTTP status and a short OAuth
  `error` code are used. The authorize URL printed at login contains the public client id, state and PKCE challenge only.
- Public safety: `build_title` accepts only a letters-only name, a known class, level 1..99, a goal of plain words (ids like `i3`,
  paths, hostnames, tokens dropped) and runs the result through `tools/publish/sanitize.py`; on any hit it falls back to
  `D.idle: playing Diablo II`. `set_title` re-checks and refuses titles the sanitizer flags.

## Tests

```
cd tools/kick && python3 -m unittest -v
```

Covers: login code exchange with PKCE S256 verification, state mismatch, missing client creds, refresh rotation (new refresh token
stored before the PATCH), concurrent callers refreshing once, 401 re-login message, forced refresh on PATCH 401, 429 Retry-After and
5xx backoff, retry cap, category resolution and cache, disabled flag, unsafe titles refused, network down, secret hygiene (mock echoes
tokens in error bodies; none reach logs or errors), and `build_title` cases.
