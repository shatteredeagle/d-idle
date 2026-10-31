# Setting the Kick title and category from our stream service

Researched 2026-10-28. Tags: V = fetched from a primary source (URL given), S = snippet or secondary, U = unverified or from memory. Not D2-specific.

## Verdict

POSSIBLE, via Kick's official public API with a user OAuth token (scope `channel:write`). The stream key and RTMP metadata cannot do it. The user must create a developer app by hand once and do one browser login; after that the service is headless.

## 1. Official public API

Sources: https://docs.kick.com (docs, `llms-full.txt` dump fetched), OpenAPI embedded in it.

- V Update: `PATCH https://api.kick.com/public/v1/channels`, scope `channel:write`, user access token only. JSON body, at least one of `stream_title` (string, min length 1), `category_id` (integer >= 1), `custom_tags` (array, max 10). Success is `204 No Content`; 400, 401, 403, 500 also documented. (https://docs.kick.com/apis/channels)
- V Read: `GET /public/v1/channels` (no params = the authenticated user's channel; or `broadcaster_user_id`/`slug`, up to 50). Returns `stream_title`, `category{id,name}`, `stream.is_live`, `stream.custom_tags`. Scope `channel:read`.
- V Categories: `GET https://api.kick.com/public/v2/categories?name=<3-100 chars>&limit=` returns `data[{id,name,tags,thumbnail}]`, cursor paginated; user or app token, no extra scope. v1 `GET /public/v1/categories?q=` is deprecated. (https://docs.kick.com/apis/categories)
- Not stated whether `name` matching is exact or fuzzy (U): cache the resolved id for "Diablo II" / "Diablo II: Resurrected" etc. in config and verify once by hand.
- V Scopes (full list): user:read, channel:read, channel:write, channel:rewards:read/write, chat:write, streamkey:read, events:subscribe, moderation:ban, moderation:chat_message:manage, kicks:read, ads:read/write. We need only `channel:write` (+ `channel:read` to verify). (https://docs.kick.com/getting-started/scopes)

### OAuth 2.1 (host is id.kick.com, separate from api.kick.com)

V from https://docs.kick.com/getting-started/generating-tokens-oauth2-flow:
- Authorize: `GET https://id.kick.com/oauth/authorize` with `client_id`, `response_type=code`, `redirect_uri`, `scope` (space separated), `code_challenge`, `code_challenge_method=S256`, `state` (listed as required).
- Token: `POST https://id.kick.com/oauth/token`, `application/x-www-form-urlencoded`, `grant_type=authorization_code`, `client_id`, `client_secret`, `redirect_uri`, `code`, `code_verifier`. Response: `access_token`, `token_type`, `refresh_token`, `expires_in`, `scope`. Client secret is sent even with PKCE (confidential client).
- Refresh: same endpoint, `grant_type=refresh_token`, `refresh_token`, `client_id`, `client_secret`; returns a new access AND refresh token.
- Also: `/oauth/revoke`, `/oauth/token/introspect`; app token via `client_credentials` (cannot call PATCH channels).
- `redirect_uri` must exactly match the one in the app settings; docs recommend `localhost` over `127.0.0.1`.
- V Changelog 25/11/2025: "Made refresh tokens reusable/flexible" (llms-full.txt changelog). S: third-party guide (skills.sh/goncy/skills/kick-api) says access tokens last 7200 s and each refresh returns a new refresh token that must be stored. Official page does not state a number for `expires_in` or refresh-token lifetime (U): always store the newest refresh token after each refresh, and treat `invalid_grant` as "re-run kick-auth".
- Rate limits: V the OpenAPI lists a `429 Too Many Requests` response on endpoints, but no numbers are published (the /getting-started/rate-limits page 404s). One title update per go-live is far below any plausible limit; back off on 429.

### App registration (manual)

V https://docs.kick.com/getting-started/kick-apps-setup: Kick account settings, Developer tab; 2FA must be enabled on the account first; creating an app yields a Client ID and Client Secret; the redirect URL is entered at creation. No app or usage limits are stated.

### Headless / server use

Yes (U for long-term, V for mechanism): after one interactive login the service only needs POST /oauth/token refreshes, no browser. Risk is the unknown refresh-token lifetime and rotation; mitigate by refreshing on each go-live and persisting the rotated token atomically.

## 2. Stream key / RTMP metadata

- V The stream key is only an ingest credential; the docs expose it only as a readable field (`streamkey:read`, `stream.key`), with no write path for title (https://docs.kick.com/getting-started/scopes). 
- S/U No Kick documentation of RTMP `onMetaData`/`@setDataFrame` or key suffixes (Twitch-style `?title=` args) being used for title or category; ingest reads only A/V. Every tool found sets title via the API or dashboard (Streamer.bot "Set Channel Title", Mix It Up, Lumia: https://docs.streamer.bot/api/sub-actions/kick/channel/set-channel-title). Conclusion: not possible, evidence is absence plus the existence of a dedicated API.

## 3. Other routes

- Dashboard (manual): title/category editable at kick.com dashboard; always works, zero risk. Fallback: tell the user the title to type.
- Chat command: S Kick's own `!title`/`!game` style commands are Kick-bot/Botrix-type features and require a bot to hold a token anyway; the official chat API (`chat:write`, `POST /public/v1/chat`) only posts messages (V OpenAPI), it does not change metadata by itself. No benefit over PATCH.
- Unofficial/internal web API (`kick.com/api/...` with session cookies, community libs like kick4k, KickLib): S exists. Cloudflare-protected, needs the user's session or password, breaks without notice, ToS risk (U: I did not fetch Kick's ToS; assume automation of login sessions is disallowed). Do not use.
- Official API usage falls under the developer terms the user accepts at app creation (U, not read).

## 4. Implementation plan (Python stdlib, the worker host)

Secrets in the keyring, `service=d2idle`: `kick_client_id`, `kick_client_secret`, `kick_refresh_token` (rotated each refresh), optional `kick_category_id` in the config file (not secret). Access token is kept in memory only.

Add `kick_api.py` to the stream control service:

1. `kick-auth` (one-time CLI): generate `code_verifier` (secrets.token_urlsafe(64)), S256 challenge, `state`. Start `http.server` on `127.0.0.1:PORT` (fixed port, e.g. 8765) handling `/callback`. Print/open the authorize URL (scope `channel:read channel:write`, redirect `http://localhost:8765/callback`; the user opens it in their browser on the worker host, or via SSH tunnel -L 8765 if the browser is elsewhere). Verify `state`, POST the code with `urllib.request`, store `refresh_token` via `secret-tool store --label=... service d2idle key kick_refresh_token`. Never log tokens; read secrets with `secret-tool lookup` and pass on stdin/args only within the process.
2. `_access_token()`: refresh (POST form) using the stored refresh token; on 200 persist the new refresh token before use; cache access token until `expires_in - 60`. On 400/401 `invalid_grant`: log "run kick-auth", return None.
3. `set_title(title, category)`: resolve category (config id, else `GET /public/v2/categories?name=` first exact name match, cached on disk), then `PATCH /public/v1/channels` with `{"stream_title": title[:N], "category_id": id}`. Title length limit unpublished (U): truncate to 100 chars to be safe, then verify with `GET /channels` and log mismatch.
4. Title builder uses only public-safe fields, e.g. `D.idle: Booti, Sorceress level 12, Normal Act 1`; no accounts, paths, IPs.
5. Failure behaviour: `set_title` wraps everything in try/except, 10 s timeout, one retry on 429/5xx with backoff, returns bool; go-live calls it in a daemon thread or after starting RTMP and never awaits it for the stream to start. Failures only log a warning and surface "title not updated" on the status board. Optional: re-send when level/act changes, at most once per few minutes.
6. Tests: stub `urllib` opener; no live calls without the user's credentials.

## What the user must do

1. Enable 2FA on the Kick account (required for the Developer tab).
2. Kick Account Settings, Developer tab, create an app (any name); set redirect URL exactly `http://localhost:8765/callback` (or the chosen PORT).
3. Copy Client ID and Client Secret into the keyring (`secret-tool store --label=d2idle service d2idle key kick_client_id`, same for `kick_client_secret`; paste at the prompt, do not paste in chat).
4. Run `kick-auth` once, log in as the channel owner, approve scopes `channel:read` and `channel:write`.
5. Once, confirm the category name and id (service prints the match).

Stored secrets: client id, client secret, rotating refresh token. The stream key stays separate for ingest.
