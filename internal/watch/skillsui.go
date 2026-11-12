package watch

// The Skills page (docs/skilltrees.md): the shared shell with one card per skill tree of the hero's class. web/skills.js
// draws /api/skills; every text goes through textContent and nothing is inline (CSP style-src 'self').

const skillsPage = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="dark light"><title>d.idle - skills</title><link rel="icon" href="data:,">
<link rel="stylesheet" href="/watch/tokens.css"><link rel="stylesheet" href="/watch/reset.css"><link rel="stylesheet" href="/watch/shell.css">
<link rel="stylesheet" href="/watch/components.css"><link rel="stylesheet" href="/watch/utilities.css">
<link rel="stylesheet" href="/watch/inventory-tip.css"><link rel="stylesheet" href="/watch/skills.css">
</head><body>
<div class="d-shell">
<header class="top d-band-status" data-band="status">
 <div class="d-brand"><span class="d-brand-mark" aria-hidden="true"></span><h1>d.idle</h1></div>
 <div class="d-band-main"></div>
 <nav class="d-links" aria-label="Other pages">
  <a class="d-btn" data-size="sm" data-variant="ghost" href="/watch?live=1">Live screen</a>
  <a class="d-btn" data-size="sm" data-variant="ghost" href="/watch/inventory">Inventory</a>
  <a class="d-btn" data-size="sm" data-variant="ghost" href="/watch/policy">Policy</a>
  <button class="d-btn" data-size="sm" data-variant="ghost" id="btn-theme" type="button" aria-label="Switch colour theme">Theme: dark</button>
 </nav>
</header>
<main class="sk-page" id="sk">
 <p class="d-note" data-sev="alarm" id="msg" role="status" hidden></p>
 <section class="d-card sk-sum" aria-labelledby="t-sum">
  <div class="d-card-head"><h2 id="t-sum">Skills</h2><span class="meta" id="summeta"></span></div>
  <p class="d-hint" id="sumnote">Read only: points are spent by the player in the game, nothing here changes them.</p>
  <div class="d-legend" aria-label="Legend"><span><b class="sk-lg" data-state="learned"></b>learned</span><span><b class="sk-lg" data-state="learnable"></b>can be learned</span><span><b class="sk-lg" data-state="locked"></b>needs a level or a prerequisite</span><span><b class="sk-lg" data-state="bonus"></b>only +skills from items</span></div>
 </section>
 <div class="sk-trees" id="trees"></div>
</main>
</div>
<div id="toasts" class="d-toasts" role="status" aria-live="polite"></div>
<script type="module" src="/watch/shell.js"></script>
<script src="/watch/skills.js"></script></body></html>`
