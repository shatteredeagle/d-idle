// Unit layer of the side-panel minimap (docs/senses-units.md). senses.js calls sensesUnitsDraw after the map; this file
// only draws what /api/senses sent (the server already cut it to the open tiers). Text via the canvas only, no HTML.
(function () {
  'use strict';
  // Real automap colours: monsters red, mercenary blue, NPCs green. Elites get a ring: gold champion, purple unique,
  // double ring superunique, crown (triangle) boss.
  var KIND = { monster: '#c0392b', merc: '#2f6fd0', npc: '#2e9d4b', player: '#e8e8e8' };
  function ring(ctx, x, y, r, col, w) { ctx.strokeStyle = col; ctx.lineWidth = w; ctx.beginPath(); ctx.arc(x, y, r, 0, 6.2832); ctx.stroke(); }
  function label(ctx, text, x, y, col) {
    ctx.font = '11px sans-serif'; ctx.textBaseline = 'bottom'; ctx.fillStyle = 'rgba(0,0,0,0.65)';
    var w = ctx.measureText(text).width; ctx.fillRect(x - w / 2 - 2, y - 13, w + 4, 13);
    ctx.fillStyle = col; ctx.textAlign = 'center'; ctx.fillText(text, x, y - 1);
  }
  window.sensesUnitsDraw = function (ctx, at, units) {
    var i, u, p, big;
    for (i = 0; i < units.length; i++) {
      u = units[i]; p = at(u); big = u.elite ? 5 : 3;
      ctx.fillStyle = KIND[u.kind] || '#c0392b';
      ctx.beginPath(); ctx.arc(p[0], p[1], big, 0, 6.2832); ctx.fill();
      if (u.elite === 'champion') ring(ctx, p[0], p[1], big + 3, '#d4a017', 2);
      else if (u.elite === 'unique') ring(ctx, p[0], p[1], big + 3, '#8e44ad', 2);
      else if (u.elite === 'superunique') { ring(ctx, p[0], p[1], big + 3, '#8e44ad', 2); ring(ctx, p[0], p[1], big + 6, '#d4a017', 2); }
      else if (u.elite === 'boss') { ctx.fillStyle = '#d4a017'; ctx.beginPath(); ctx.moveTo(p[0] - 6, p[1] - big - 2); ctx.lineTo(p[0], p[1] - big - 9); ctx.lineTo(p[0] + 6, p[1] - big - 2); ctx.closePath(); ctx.fill(); }
      if (u.target && u.lifePct) {
        var w = 30, f = Math.max(0, Math.min(100, u.lifePct)) / 100 * w;
        ctx.fillStyle = 'rgba(0,0,0,0.7)'; ctx.fillRect(p[0] - w / 2 - 1, p[1] + 7, w + 2, 5);
        ctx.fillStyle = '#c0392b'; ctx.fillRect(p[0] - w / 2, p[1] + 8, f, 3);
      }
      if (u.name && (u.target || u.elite)) label(ctx, u.name + (u.mods && u.mods.length ? ': ' + u.mods.join(', ') : ''), p[0], p[1] - big - 4, u.elite ? '#f1c40f' : '#eee');
    }
  };
})();
