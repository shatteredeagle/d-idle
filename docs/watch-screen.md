# Watch screen (TASK-20)

> status: offline on the simulated runner; not verified live. Design, contract and API needs: [research/watch-screen-design.md](research/watch-screen-design.md). Tags: V = proven by a test, U = unverified.

The player screen of ADR 0016 items 20 to 23: Thoughts feed, orders box, uptime, rules list, journal. Code: `internal/watch` (Hub, handlers, simulated runner, page in `web/`), flag `-watch` of `cmd/d2dview` (off by default).

```
go run ./cmd/d2dview -watch                       # screen on :8891 (loopback), answer route on 127.0.0.1:8892, simulated Sorceress
go run ./cmd/d2dview -watch -watch-class paladin  # hardcore Paladin: risky retreat point, confirm card
go run ./cmd/d2dview -upstream $SOCK -watch       # same routes next to the game frame stream
```

The viewer side serves GET only (V: every other method is 405); every write goes to the separate `-watch-answer-addr` listener, which checks origin, content type, body size and the optional token. d2dview hosts the screen; d2dash was not used (U whether d2dview can take the goals sim).

Opening the page from another host: pass `-watch-origin http://HOST:8891` (the answer route refuses any other page origin) and, off loopback, a token (`D2DVIEW_TOKEN`; the answer route reuses it, the page sends the `d2dview` cookie).

Tests: `go vet ./internal/watch/... ./cmd/d2dview/... && go test -p 1 ./internal/watch/... ./cmd/d2dview/...`; golden files `internal/watch/testdata` (rewrite with `go test ./internal/watch -run Golden -update`).
