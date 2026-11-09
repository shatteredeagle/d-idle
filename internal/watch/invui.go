package watch

// The inventory page: a static shell and a script that draws /api/inventory. Every text goes through textContent.
// Writes go to the answer listener like the Pact page (base() asks /api/config for its address).

const invPage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>d.idle - inventory</title>
<style>body{font:15px/1.4 system-ui,sans-serif;margin:1rem auto;max-width:70rem;padding:0 1rem;background:#15130f;color:#e8e0cc}
h1,h2{font-weight:600}a{color:#d9b25f}.note{color:#9a8f78;font-size:.85rem}.err{color:#e07a5f}
.slots{display:grid;grid-template-columns:repeat(auto-fill,minmax(14rem,1fr));gap:.5rem}
.slot,.it{border:1px solid #3a3326;border-radius:6px;padding:.4rem .6rem;background:#1c1912}
.slot.pin{border-color:#d9b25f}.q-magic{color:#6f8fff}.q-rare{color:#f0dc5a}.q-unique{color:#c8a45a}.q-set{color:#5fd05f}.q-crafted{color:#f0a040}.q-runeword{color:#c8a45a}.q-low,.q-normal,.q-superior{color:#e8e0cc}
.grid{position:relative;border:1px solid #3a3326;background:#100e0a}.cell{position:absolute;box-sizing:border-box;border:1px solid #5a4d33;background:#262114;overflow:hidden;font-size:.65rem;cursor:pointer;padding:1px}
.cell.sell{outline:2px solid #e07a5f}.cell.keep{outline:2px solid #5b7a3a}.cell.sel{background:#3a3220}
.badge{font-size:.7rem;border:1px solid #5a4d33;border-radius:3px;padding:0 .25rem;margin-left:.25rem}.badge.sell{color:#e07a5f}.badge.keep{color:#8fbf5f}
button{font:inherit;cursor:pointer}table{border-collapse:collapse;width:100%}td,th{border-bottom:1px solid #2b261c;padding:.2rem .4rem;text-align:left;vertical-align:top}</style></head>
<body><h1>Inventory</h1><p><a href="/watch?live=1">Live screen</a></p><p id="note" class="note"></p><div id="msg" class="err" role="status"></div>
<h2>Equipped</h2><div id="equip" class="slots"></div>
<h2>Repair</h2><div id="repair" class="slot"></div>
<p class="note">Wearing something from here pins the slot: the bot will not swap it back. Unpin it to hand the slot back.</p>
<h2>Backpack</h2><div id="pack"></div><div id="detail"></div>
<h2>Belt</h2><div id="belt"></div>
<h2>Stash</h2><div id="stash"></div>
<h2>Equip requests</h2><div id="queue"></div>
<script src="/watch/inventory.js"></script></body></html>`

const invJS = `(function(){'use strict';
var msg=document.getElementById('msg'),sel=null,last=null,SLOTS=['head','neck','torso','rhand','lhand','rring','lring','belt','feet','gloves'];
function el(t,c,x){var e=document.createElement(t);if(c)e.className=c;if(x!==undefined)e.textContent=x;return e;}
function say(t){msg.textContent=t||'';}
function base(){return fetch('/api/config',{cache:'no-store'}).then(function(r){return r.json();}).then(function(c){
 if(c.answerBase)return c.answerBase;return location.protocol+'//'+location.hostname+':'+c.answerPort;});}
function post(path,body){return base().then(function(u){return fetch(u+path,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});})
 .then(function(r){return r.json().then(function(j){if(!r.ok)throw new Error(j.error||('error '+r.status));return j;});})
 .then(function(){say('');load();},function(e){say(e.message);});}
function name(it){return (it.name||it.code)+(it.ethereal?' (eth)':'')+(it.identified?'':' (unidentified)');}
function badges(e,it){if(it.mark)e.appendChild(el('span','badge '+it.mark,it.mark==='sell'?'to sell':'keep'));}
function lines(it){var o=[];if(it.maxDur)o.push('durability '+it.dur+'/'+it.maxDur+(it.dur<=0?' (broken)':''));if(it.level)o.push('item level '+it.level);if(it.sockets)o.push(it.sockets+' sockets');(it.props||[]).forEach(function(p){o.push(p);});return o;}
function grid(box,items,cols,rows,unit){box.textContent='';var g=el('div','grid');g.style.width=cols*unit+'px';g.style.height=rows*unit+'px';
 items.forEach(function(it){var c=el('div','cell q-'+it.quality+(it.mark?' '+it.mark:'')+(sel&&sel.id===it.id?' sel':''));
  c.style.left=it.x*unit+'px';c.style.top=it.y*unit+'px';c.style.width=Math.max(it.w,1)*unit+'px';c.style.height=Math.max(it.h,1)*unit+'px';
  c.textContent=name(it);c.title=name(it);c.tabIndex=0;c.addEventListener('click',function(){sel=it;draw(last);});g.appendChild(c);});
 box.appendChild(g);}
function detail(v){var d=document.getElementById('detail');d.textContent='';if(!sel)return;
 var it=null;[v.backpack,v.stash,v.belt].forEach(function(l){(l||[]).forEach(function(x){if(x.id===sel.id)it=x;});});if(!it){sel=null;return;}
 var h=el('h3','q-'+it.quality,name(it));badges(h,it);d.appendChild(h);
 lines(it).forEach(function(l){d.appendChild(el('div','note',l));});
 if(it.why)d.appendChild(el('div','',"Why: "+it.why));
 if(it.advice)d.appendChild(el('div','advice',it.advice));
 var bar=el('p');
 var inPack=v.backpack.some(function(x){return x.id===it.id;});
 function btn(t,f,dis){var b=el('button','',t);b.type='button';if(dis)b.disabled=true;b.addEventListener('click',f);bar.appendChild(b);bar.appendChild(document.createTextNode(' '));}
 btn(it.mark==='sell'?'Clear sell mark':'Mark for sale',function(){post('/api/inventory/mark',{id:it.id,mark:it.mark==='sell'?'':'sell'});},!inPack||!it.canSell&&it.mark!=='sell');
 btn(it.mark==='keep'?'Clear keep mark':'Keep',function(){post('/api/inventory/mark',{id:it.id,mark:it.mark==='keep'?'':'keep'});});
 d.appendChild(bar);
 if(inPack){var w=el('p');w.appendChild(document.createTextNode('Equip in: '));
  SLOTS.forEach(function(s){var b=el('button','',s);b.type='button';if((it.canWear||[]).indexOf(s)<0)b.disabled=true;b.title=b.disabled?'not possible right now':'queue this equip';
   b.addEventListener('click',function(){post('/api/inventory/equip',{id:it.id,slot:s});});w.appendChild(b);w.appendChild(document.createTextNode(' '));});
  d.appendChild(w);if(!v.inTown)d.appendChild(el('p','note','The hero is not in town: equip works in town only.'));}}
function draw(v){last=v;document.getElementById('note').textContent=(v.note||'')+(v.class?v.class+' - gold '+v.gold+', stash gold '+v.stashGold:'');
 var eq=document.getElementById('equip');eq.textContent='';var pins={};(v.pins||[]).forEach(function(p){pins[p.slot]=p;});
 SLOTS.forEach(function(s){var it=v.equipment[s],box=el('div','slot'+(pins[s]?' pin':''));box.appendChild(el('b','',s+': '));
  if(it){var n=el('span','q-'+it.quality,name(it));box.appendChild(n);lines(it).forEach(function(l){box.appendChild(el('div','note',l));});if(it.why)box.appendChild(el('div','note',it.why));}else box.appendChild(el('span','note','empty'));
  if(pins[s]){box.appendChild(el('div','note','pinned by you: the bot leaves this slot alone'));var b=el('button','','Unpin');b.type='button';b.addEventListener('click',function(){post('/api/inventory/unpin',{slot:s});});box.appendChild(b);}
  eq.appendChild(box);});
 grid(document.getElementById('pack'),v.backpack||[],10,4,34);
 grid(document.getElementById('belt'),v.belt||[],4,1,34);
 grid(document.getElementById('stash'),v.stash||[],6,8,34);
 detail(v);
 var q=document.getElementById('queue');q.textContent='';var rows=(v.queue||[]).concat(v.recent||[]);
 if(!rows.length)q.appendChild(el('p','note','Nothing queued.'));else{var t=el('table');rows.forEach(function(r){var tr=el('tr');[r.name,r.slot,r.state,r.reason||''].forEach(function(x){tr.appendChild(el('td','',x));});t.appendChild(tr);});q.appendChild(t);}}
function drawRepair(v){var b=document.getElementById('repair');b.textContent='';
 var q=v.request,pend=q&&q.state==='requested';
 b.appendChild(el('div','','Repair mode: '+((v.modes||[]).filter(function(m){return m.current;}).map(function(m){return m.label;})[0]||v.mode)+' (change it on the policy sheet)'));
 if(v.worst)b.appendChild(el('div',v.worst.indexOf('Broken')===0?'err':'note',v.worst));
 if(v.estimate)b.appendChild(el('div','note','Repair All costs at least '+v.estimate+' gold.'));
 if(q)b.appendChild(el('div','note',q.state==='requested'?'Requested: the hero repairs at its next town visit.'+(q.note?' ('+q.note+')':''):q.state==='done'?'Last request: done.':'Last request: '+q.state));
 var bt=el('button','',pend?'Cancel repair':'Repair now');bt.type='button';
 bt.addEventListener('click',function(){post(pend?'/api/repair/cancel':'/api/repair/request',{});});b.appendChild(bt);}
function loadRepair(){fetch('/api/repair',{cache:'no-store'}).then(function(r){return r.json().then(function(j){if(!r.ok)throw new Error(j.error||'error');return j;});}).then(drawRepair,function(e){document.getElementById('repair').textContent=e.message;});}
function load(){loadRepair();fetch('/api/inventory',{cache:'no-store'}).then(function(r){return r.json().then(function(j){if(!r.ok)throw new Error(j.error||'error');return j;});}).then(draw,function(e){say(e.message);});}
load();setInterval(load,4000);
})();`
