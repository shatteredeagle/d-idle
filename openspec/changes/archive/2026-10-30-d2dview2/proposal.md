## Why

The user wants to watch the sped-up d2d bot play. d2dview exists; it lacked a d2watch tile, a way for a bot-managed d2d restart to go through the tap, and a one-command demo.

## What Changes

- cmd/d2watch/web/app.js: a d2d tile (img of d2dview /stream.mjpg), shown only while d2dview answers
- --d2d-dial PATH (cmd/d2bot/d2dbackend.go, internal/d2denv/env.go Config.Dial): the bot's client dials the tap, d2d keeps its own socket
- tools/d2dview-demo.sh: build, ship, run d2d + viewer + d2bot under the d2d lab lock, time-boxed
