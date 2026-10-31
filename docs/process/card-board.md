# Card board: reading the user's decisions

> status: current. The page is the `/cards` route of the status board (the board host name, port 8897); its code is in the private ops repo.

The user approves, defers or rejects proposed and deferred change cards from a phone. The coordinator reads the answers with
`tools/cards-decisions.sh` (a stub into the ops repo):

- `tools/cards-decisions.sh pending` lists decisions not yet acted on: `card decision time PENDING | note`.
- `tools/cards-decisions.sh list` lists every current decision, acted on or not.
- `tools/cards-decisions.sh ack ID` marks a decision as acted on; `clear ID` removes it (the user can decide again).
- The same data is at `GET /api/cards/decisions` on the board (JSON map by card id), the full card list at `GET /api/cards`.

Meaning: `approve` = launch when a slot is free; `defer` = keep queued; `reject` = archive the card. A card the dispatcher already
runs (status not proposed or deferred, with a `docs/OWNERS` line) is read-only on the page and refuses decisions.

Storage: `decisions.jsonl` (append-only log) and `current.json` (latest per card) in the board's state directory under `cards/`.
Writes need the shared token in the `X-Board-Token` header (file `cards/token`, mode 600, created by the board on first start;
`tools/cards-decisions.sh token-path` prints its location, never its value). Ack and act on every pending decision, then `ack` it.
