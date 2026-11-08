package watch

// The Pact offer page (TASK-168, docs/pact-ui.md): a static shell and a script that draws /api/pacts. Every text goes
// through textContent. The script builds a proposal from the clicked ranks, asks the viewer route for a preview (current
// set and proposed set side by side, heat against the cap, the reason a rank is refused) and posts the choice to the
// answer route like the policy page does.

const pactPage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Pacts</title>
<style>body{font:15px/1.4 system-ui,sans-serif;margin:1rem auto;max-width:60rem;padding:0 1rem;background:#15130f;color:#e8e0cc}
h1,h2,h3{font-weight:600}.bar{background:#2b261c;border-radius:4px;height:1.1rem;position:relative;overflow:hidden}
.bar i{display:block;height:100%;background:#a0472f}.bar.ok i{background:#5b7a3a}.bar b{position:absolute;left:.5rem;top:0;font-size:.8rem;font-weight:400}
.fam{margin:1.2rem 0}.pact{border:1px solid #3a3326;border-radius:6px;padding:.6rem .8rem;margin:.5rem 0}.pact.locked{opacity:.55;border-style:dashed}
.rank{display:flex;gap:.6rem;align-items:baseline;padding:.2rem 0;border-top:1px solid #2b261c}.rank button{min-width:5rem}
.rank.bad{color:#9a8f78}.rank.sel{background:#2d2a17}.note{color:#9a8f78;font-size:.85rem}table{border-collapse:collapse;width:100%}
td,th{border-bottom:1px solid #2b261c;padding:.25rem .5rem;text-align:left}.err{color:#e07a5f}button{font:inherit}</style></head>
<body><h1>Pacts</h1><p id="note" class="note"></p><div id="msg" class="err" role="status"></div>
<h2>Heat</h2><div id="heat"></div>
<h2>Current and proposed</h2><div id="sets"></div><div id="levers"></div>
<p><button id="apply" disabled>Choose this set</button> <button id="clear">Start over</button></p>
<h2>Catalogue</h2><div id="fams"></div>
<script src="/watch/pacts.js"></script></body></html>`

const pactJS = `(function(){'use strict';
var sel={},msg=document.getElementById('msg'),note=document.getElementById('note');
function el(t,c,x){var e=document.createElement(t);if(c)e.className=c;if(x!==undefined)e.textContent=x;return e;}
function say(t){msg.textContent=t||'';}
function base(){return fetch('/api/config',{cache:'no-store'}).then(function(r){return r.json();}).then(function(c){
 if(c.answerBase)return c.answerBase;return location.protocol+'//'+location.hostname+':'+c.answerPort;});}
function query(){var p=Object.keys(sel).map(function(k){return k+':'+sel[k];}).join(',');return p===''?'-':p;}
function load(init){var q=init?'':'?propose='+encodeURIComponent(query());
 fetch('/api/pacts'+q,{cache:'no-store'}).then(function(r){return r.json();}).then(function(v){
  if(v.error){say(v.error);return;}if(init)(v.current.picks||[]).forEach(function(p){sel[p.id]=p.rank;});draw(v);}).catch(function(){say('cannot reach the bot');});}
function setLine(s){var t=(s.picks.length?s.picks.map(function(p){return p.name+' '+p.rank;}).join(', '):'no Pact')+' | heat '+s.heat+' | tally +'+s.tallyPct+'%';
 if(s.charmPoints)t+=' | charm points +'+s.charmPoints;return t;}
function draw(v){note.textContent=v.note;
 var h=document.getElementById('heat');h.textContent='';var shown=v.proposed||v.current;
 var bar=el('div','bar'+(shown.error?'':' ok'));var i=el('i');i.style.width=Math.min(100,100*shown.heat/v.maxCap)+'%';bar.appendChild(i);
 bar.appendChild(el('b','','heat '+shown.heat+' of cap '+v.heatCap+' (the cap can grow to '+v.maxCap+'), '+v.slots+' slots'));h.appendChild(bar);
 var s=document.getElementById('sets');s.textContent='';var tb=el('table');
 [['Now',v.current],['Proposed',v.proposed]].forEach(function(r){if(!r[1])return;var tr=el('tr');tr.appendChild(el('th','',r[0]));tr.appendChild(el('td','',setLine(r[1])));
  if(r[1].error)tr.appendChild(el('td','err',r[1].error));tb.appendChild(tr);});s.appendChild(tb);
 var l=document.getElementById('levers');l.textContent='';
 if(v.levers&&v.levers.length){l.appendChild(el('h3','','Policy now and proposed'));var t=el('table');var hr=el('tr');['Lever','Now','Proposed','From'].forEach(function(x){hr.appendChild(el('th','',x));});t.appendChild(hr);
  v.levers.forEach(function(x){var tr=el('tr');[x.lever,x.now,x.proposed,x.source].forEach(function(y){tr.appendChild(el('td','',y));});t.appendChild(tr);});l.appendChild(t);}
 document.getElementById('apply').disabled=!v.proposed||!!v.proposed.error;
 var f=document.getElementById('fams');f.textContent='';
 v.families.forEach(function(fam){var d=el('div','fam');d.appendChild(el('h3','',fam.family));
  fam.pacts.forEach(function(p){var pd=el('div','pact'+(p.locked?' locked':''));pd.appendChild(el('strong','',p.name));
   if(p.locked)pd.appendChild(el('div','note',p.unlock));else{pd.appendChild(el('div','note',p.learn));pd.appendChild(el('div','note','Away: '+p.away));}
   p.ranks.forEach(function(r){var rd=el('div','rank'+(r.status==='ok'?'':' bad')+(sel[p.id]===r.rank?' sel':''));
    var b=el('button','',sel[p.id]===r.rank?'Drop':'Rank '+r.rank);b.disabled=p.locked;
    b.onclick=function(){if(sel[p.id]===r.rank)delete sel[p.id];else sel[p.id]=r.rank;load(false);};rd.appendChild(b);
    rd.appendChild(el('span','','heat '+r.heat+', tally +'+r.tallyPct+'%'+(r.charmPoints?', charm +'+r.charmPoints:'')+(r.title?', title "'+r.title+'"':'')));
    rd.appendChild(el('span','',r.effect||''));if(r.status!=='ok'&&!p.locked)rd.appendChild(el('span','err',r.status));pd.appendChild(rd);});
   d.appendChild(pd);});f.appendChild(d);});}
document.getElementById('clear').onclick=function(){sel={};load(false);};
document.getElementById('apply').onclick=function(){var picks=Object.keys(sel).map(function(k){return {id:k,rank:sel[k]};});
 base().then(function(u){return fetch(u+'/api/pacts/select',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({picks:picks})});})
 .then(function(r){return r.json();}).then(function(j){say(j.ok?'Chosen.':(j.error||'failed'));if(j.ok)setTimeout(function(){location.reload();},500);}).catch(function(){say('the answer route is not reachable');});};
load(true);
})();`
