# 4. AutoExile (Path of Exile ExileCore plugin): ARPG farming bot with phone dashboard

Repo: https://github.com/Sikaka/AutoExile. No license file listed (V, `gh api` returned empty license), 12 stars, created 2026-03-11, pushed 2026-08-21 (V). Small and young, but the nearest ARPG match.
Runs inside/alongside the online game via ExileApi (memory-reading framework for PoE) (V README: "ExileCore plugin"). Online game, so ban risk applies there, not to us.

## What it does (V, README)
- **Modes**, each a stateful loop: Simulacrum (15-wave AFK farm), Boss Farming (stash -> take fragments -> open map device -> portal -> fight -> loot -> exit, repeat until fragments run out), Blight, Follower (follow a named party leader; "aura bot alongside your main"), Mapping (experimental).
- **Web dashboard on localhost:9876**, can be exposed on the network "to control from your phone": live status, change settings, session stats (V).
- **Config as data**: build (skill slots with conditions: buff/debuff, enemy count, rarity, range, priority), flasks, loot thresholds by chaos value, boss type, stock, stash tab names, max deaths (V).
- Stable/Experimental labels per mode, honest in README.
- Plugin-updater distribution: ExileApi auto-clones and rebuilds from the repo URL (V).

## Lessons
- The closest to "goal queue" is a **mode picker with a resource precondition** ("runs until you're out of fragments"). A goal needs a stop condition and a supply check; our goals ("farm Mephisto for runes") have the same shape.
- A tiny project already ships phone dashboard + live stats: that UX is cheap and users love it. Confirms our dashboard idea but offers no queue, schedule or meta layer.
- Death budget (`max deaths`) as a goal parameter is a good idea to steal: goal failure thresholds are user-facing config.
- Depends on a third-party memory framework whose updates follow game patches (S: ExileApi-Compiled updater). Same fragility as our offset table, mitigated here by only one game version; we have a frozen 1.14d, which is our big advantage.

## Reuse
- No licence => no code reuse; design only. Treat as inspiration, not a dependency.
- Other PoE bot repos in results were commercial-looking "AFK bot" pages with undetectability claims; ignore (S).
