package watch

// The inventory page (docs/ui-design-system.md components 13 and 14): the shared shell (tokens, the top band, theme)
// with a Diablo II paper doll, the backpack, belt and stash as cell grids, and the repair card. web/inventory.js draws
// /api/inventory and /api/repair; every text goes through textContent and nothing is inline (CSP style-src 'self': the
// old page's <style> block was refused, so it rendered unstyled). Writes go to the answer listener.

const invPage = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="dark light"><title>d.idle - inventory</title><link rel="icon" href="data:,">
<link rel="stylesheet" href="/watch/tokens.css"><link rel="stylesheet" href="/watch/reset.css"><link rel="stylesheet" href="/watch/shell.css">
<link rel="stylesheet" href="/watch/components.css"><link rel="stylesheet" href="/watch/inventory.css"><link rel="stylesheet" href="/watch/utilities.css">
</head><body>
<div class="d-shell">
<header class="top d-band-status" data-band="status">
 <div class="d-brand"><span class="d-brand-mark" aria-hidden="true"></span><h1>d.idle</h1></div>
 <div class="d-band-main"></div>
 <nav class="d-links" aria-label="Other pages">
  <a class="d-btn" data-size="sm" data-variant="ghost" href="/watch?live=1">Live screen</a>
  <a class="d-btn" data-size="sm" data-variant="ghost" href="/watch/policy">Policy</a>
  <button class="d-btn" data-size="sm" data-variant="ghost" id="btn-theme" type="button" aria-label="Switch colour theme">Theme: dark</button>
 </nav>
</header>
<main class="inv-page" id="inv">
 <p class="d-note" data-sev="alarm" id="msg" role="status" hidden></p>
 <section class="d-card inv-equip" aria-labelledby="t-equip">
  <div class="d-card-head"><h2 id="t-equip">Equipped</h2><span class="meta" id="equipmeta"></span></div>
  <div class="inv-doll-wrap"><div class="d-doll" id="equip" role="group" aria-label="Paper doll"></div><div class="inv-detail" id="detail"></div></div>
  <div class="d-legend" aria-label="Legend"><span><b class="lg-pin">PIN</b>player-equipped, the bot leaves the slot alone</span><span><b class="lg-sell">S</b>sell at the next vendor</span><span><b class="lg-keep">K</b>keep, never sold</span><span>pips: durability, amber under 30 percent, red and BROKEN at 0</span></div>
 </section>
 <section class="d-card inv-pack" aria-labelledby="t-pack">
  <div class="d-card-head"><h2 id="t-pack">Backpack</h2><span class="meta" id="packmeta"></span></div>
  <div class="d-cells" id="pack"></div>
  <p class="d-hint" id="note"></p>
 </section>
 <section class="d-card inv-repair" aria-labelledby="t-repair">
  <div class="d-card-head"><h2 id="t-repair">Repair</h2></div>
  <div class="inv-repair-body" id="repair"></div>
 </section>
 <section class="d-card inv-belt" aria-labelledby="t-belt">
  <div class="d-card-head"><h2 id="t-belt">Belt</h2></div>
  <div class="d-cells" id="belt"></div>
 </section>
 <section class="d-card inv-stash" aria-labelledby="t-stash">
  <div class="d-card-head"><h2 id="t-stash">Stash</h2><span class="meta" id="stashmeta"></span></div>
  <div class="d-cells" id="stash"></div>
 </section>
 <section class="d-card inv-queue" aria-labelledby="t-queue">
  <div class="d-card-head"><h2 id="t-queue">Equip requests</h2></div>
  <div id="queue"></div>
 </section>
</main>
</div>
<div id="toasts" class="d-toasts" role="status" aria-live="polite"></div>
<script type="module" src="/watch/shell.js"></script>
<script src="/watch/inventory.js"></script></body></html>`
