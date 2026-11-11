'use strict';
const $ = (id) => document.getElementById(id);

function render(s) {
  const setup = s.status === 'setup';
  $('form').hidden = false;
  $('retry').hidden = setup;
  if (document.activeElement !== $('address')) $('address').value = s.address || '';
  const text = {
    setup: 'Type the address of the game server (for example host:8891). It is saved on this computer only.',
    checking: 'Checking ' + s.address + ' ...',
    down: s.address + ' is not reachable. Is the game running? This page tries again every few seconds.',
    warming: s.address + ' answers but the game is still starting. That takes one to three minutes; this page keeps trying.',
    up: 'Opening ' + s.address + ' ...',
  }[s.status] || '';
  $('msg').textContent = text;
  $('error').hidden = !s.error;
  $('error').textContent = s.error || '';
}

$('form').addEventListener('submit', async (e) => {
  e.preventDefault();
  render(await window.d2idle.saveAddress($('address').value));
});
$('retry').addEventListener('click', async () => render(await window.d2idle.retry()));
window.d2idle.onState(render);
window.d2idle.getState().then(render);
