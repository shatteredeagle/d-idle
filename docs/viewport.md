# Viewport: capture and stream library (agent VIEWPORT)

`internal/viewport` (stdlib only) captures one X window and streams it to many viewers. It is used by `cmd/d2watch`
and is the implementation behind the server's `Instance.Mounts["viewport"]` (docs/api.md, WP-SERVER). No input path.

## API
- `Capturer{Open(ctx, Spec) (Session, error); Persistent() bool}`, `Session{Next(ctx) (Frame, error); Close()}`.
  - `ImportCapturer`: one `import -window <id>` per frame (`OMP_NUM_THREADS=1`, `-resize WxH>` for size limits).
  - `FFmpegCapturer`: one persistent `ffmpeg -f x11grab -window_id <id> ... -f mjpeg -` per session, output split into JPEGs
    (`jpegSplitter`). Grabs the window only (`-window_id`), mouse pointer not drawn.
  - Later H.264: add a Capturer whose `Frame.Mime` is `video/mp4` (fragmented) and a handler route; `Spec.Format`, `Frame.Mime`
    and the Session shape are the seam. Nothing else in `Stream` assumes JPEG.
- `Stream` (`New(Config)`): ONE capture loop shared by all viewers. Lazy: it starts on the first `Frame`/`Next`/`Watch` call and
  exits (closing the ffmpeg process) `Idle` (3 s) after the last request with no stream open. One capture in flight, never queued
  back to back (pull capturers are paced to `FPS`; ffmpeg paces itself). Errors are published to waiters, retried every
  max(period, 500 ms); `Config.Target` is re-evaluated at every attempt (restarted game = new window); `OnError` hook.
  Limits: `FPS`, `MaxW`/`MaxH` (downscale only), `Quality`; `Limiter` = global concurrency cap (pull capturer: one slot per
  capture; persistent: one slot per open session).
- `Handler(HandlerConfig)`: routes relative to the mount: `/frame.jpg`, `/stream.mjpg`, `/overlay`, `/status`; GET/HEAD only.
  Mount it as `Instance.Mounts["viewport"] = viewport.Handler(...)`. `ServeFrame`/`ServeStream` are exported for hosts with own routing.
- Overlay: producers give a `Scene` (player or game viewport scroll VX,VY + `WorldShape`s in subtiles: point, path, circle, polygon);
  `Project(input.Transform, Scene)` returns screen-space pixels of the 800x600 client area (`sx = 16(dx-dy)`, `sy = 8(dx+dy)`, V live
  per docs/input.md). A world circle becomes an ellipse with rx = 16 r sqrt2, ry = 8 r sqrt2 (algebra, U against the game).
  The client scales by frameWidth / `w`. Nothing in the viewport reads game memory.

## Verified (2026-10-28)
- Unit tests with a fake capturer: lazy, fps cap, shared frames across 8 viewers, one in flight, slow capture, global cap, persistent slot,
  errors to waiters, handler routes (`go test -race ./internal/viewport`). V.
- Live X (no game): scratch Xvfb :199 on the worker host (load about 19) with a tk dummy window 800x587, `TestLiveX11` (`D2VP_DISPLAY`,
  `D2VP_WINDOW`, `D2VP_SECS`; skipped otherwise). Both capturers deliver valid JPEGs and honour MaxW/MaxH. V.

| capturer | fps | CPU of a core (whole viewer process + children) | per frame |
|---|---|---|---|
| import | 4 | 8.4 % | 21.6 ms |
| import | 10 | 21.1 % | 21.4 ms |
| ffmpeg | 4 | 4.1 % | 10.5 ms |
| ffmpeg | 10 | 7.7 % | 7.7 ms |

  ffmpeg is about 2 to 3x cheaper per frame and gets cheaper with rate (process start amortised); import is cost-linear but has no
  resident process and no warm-up. Measured on a mostly static dummy window; a real game frame changes more (U). Not measured on the
  game itself.

## Not verified / next
- `ffmpeg x11grab` against Wine's game window on a private Xvfb, and on the user's XWayland i0 (`-window_id` of an overlapped window
  may include overlapping windows, as `import` does). U.
- d2watch default stays `-capturer import`; switch with `-capturer ffmpeg` after a live check (see watch.md).
