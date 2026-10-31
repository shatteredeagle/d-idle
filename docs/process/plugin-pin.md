# Matt Pocock skills plugin: pin

Pinned (V, read from `~/.claude/plugins/installed_plugins.json` 2026-10-30): plugin `mattpocock-skills@claude-plugins-official`, version **1.3.1**, gitCommitSha `49dd158d1076134a641b33efb035946536778336`, installed 2026-10-30, scope user, cache `~/.claude/plugins/cache/claude-plugins-official/mattpocock-skills/1.3.1` (read-only for agents). Review of the plugin: `matt-pocock-skills-fit.md`.

## Why no settings edit was made

Findings from the Claude Code docs (code.claude.com plugins/install, loading, marketplace-reference; read by a claude-code-guide agent, 2026-10-30):

- Documented: auto-update is ON by default for `claude-plugins-official`, and can be switched off per marketplace (not per plugin) by the `/plugin` UI (Marketplaces > claude-plugins-official > Disable auto-update), which writes `autoUpdate` to `known_marketplaces.json`, or by `autoUpdate: false` on an `extraKnownMarketplaces` entry in settings.json (the settings entry wins).
- Not documented: pinning one plugin to a version or sha from user settings (`enabledPlugins` values are booleans only); a per-plugin auto-update opt-out. `ref`/`sha` belong to the marketplace maintainer's marketplace.json.
- Inferred only: that declaring the reserved official marketplace name in `extraKnownMarketplaces` is accepted. It is also not in `~/.claude/settings.json` today (that file lists the three other marketplaces only), so adding it would register a marketplace entry, not just flip a flag. Hand-editing `known_marketplaces.json` is a state file edit the docs do not support. No safe key-scoped edit exists, so nothing was changed (settings.json, installed_plugins.json and known_marketplaces.json untouched, no backup needed).

## Steps for the user (once)

1. In a Claude Code session: `/plugin`, Marketplaces, `claude-plugins-official`, "Disable auto-update". This holds every plugin of that marketplace at its installed version (the only granularity available).
2. Check: `grep -A6 claude-plugins-official ~/.claude/plugins/known_marketplaces.json` should show `"autoUpdate": false`; `installed_plugins.json` still says 1.3.1 and the sha above.
3. Alternative that touches nothing per marketplace: env `DISABLE_AUTOUPDATER=1` (turns off every auto-update, documented), in the `env` block of settings.json.
4. Updating later is deliberate: review the CHANGELOG diff, then `claude plugin update mattpocock-skills@claude-plugins-official`, and record the new version and sha here.

Anything the user-run skills write into this repo is reviewed like any other change; a version bump changes skill text, so re-read `docs/process/matt-pocock-skills-fit.md` sections 2 and 6 first.
