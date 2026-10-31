# F: headless container and setup wizard, usable by a friend

Exit: on an empty volume, a person who is neither the user nor an agent goes from their own game files to a running goal through
the web wizard, no shell; d2doctor passes on two hosts; upgrades keep state. 41 % today, unchanged by the rescore (image spike done with a probe; wizard
backend, nine steps and d2doctor offline; no real game in the image yet; the new managed mode makes F-2 straightforward).

## Work packages
| wp | what | owner | effort (U) |
|---|---|---|---|
| F-1 | Real D2 1.14d in the image: plain Wine vs a pinned GE-Proton, `-w` on Xvfb, 800x600 vs 1024x768 (R11), CPU per instance | d2-worker on the worker host docker | 2 |
| F-2 | Bot in the container: `d2container run` starts d2bot --managed as PID 1 child, one goal 30 min, d2watch frame | d2-live | 1.5 |
| F-3 | Installer viewport: view plus input of the installer window in the browser (noVNC/Xpra or our own capture + click forwarding during setup only) | d2-worker | 3 |
| F-4 | Wizard live pass on a scratch volume (docs/setup.md live checks 1-5) | d2-live | 1 |
| F-5 | Release-to-release upgrade test; pinned runner checked at start | d2-worker | 1 |
| F-6 | A friend install from docs; collect friction; fix docs | main + user | 1 + their time |
| F-7 | NAS template or LXC deploy (ask the user first; NAS only through its UI) | main | 0.5 |

F-1 and F-2 can start now (they need Docker on the worker host, not a game instance); F-3 is the long pole.

## Risks
- Plain Wine may not run 1.14d as well as GE-Proton11-11 (only verified runner): fall back to shipping a fetched Proton build.
- CD key entry needs the installer's own UI: if the viewport cannot type into it reliably, accept "bring an installed game dir" as the beta path.
- Image size 1.16 GB (Debian wine32 deps): acceptable for beta.
