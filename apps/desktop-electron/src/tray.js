'use strict';
const { PAGES } = require('./address');
const { pageItems } = require('./menu');

const STATUS_TEXT = { setup: 'Not set up', checking: 'Checking ...', up: 'Online', warming: 'Warming up', down: 'Offline' };

// The status line is reachability only: the watch pages need the login, and there is no token-less endpoint with the
// hero name or level on the watch server (V: grep of internal/watch routes, 2026-11-10; /healthz belongs to the d2d
// control server), so nothing here asks for one.
function statusLabel(state) {
  const t = STATUS_TEXT[state.status] || 'Unknown';
  return state.address ? `${t} (${state.address.replace(/^https?:\/\//, '')})` : t;
}

// trayMenuTemplate is plain data (a test and the real Menu share it). v: {state, autostart, hideToTray, update};
// h: {openPage, show, toggleAutostart, toggleHideToTray, checkUpdate, showUpdate, quit}.
function trayMenuTemplate(v, h) {
  const items = [
    { label: 'Open', click: () => h.show() },
    { label: 'Pages', submenu: pageItems(h) },
    { type: 'separator' },
    { label: statusLabel(v.state), enabled: false },
  ];
  if (v.update && v.update.available) items.push({ label: `Update available: ${v.update.version}`, click: () => h.showUpdate() });
  else items.push({ label: 'Check for updates', click: () => h.checkUpdate() });
  items.push(
    { type: 'separator' },
    { label: 'Start with the system', type: 'checkbox', checked: !!v.autostart, click: () => h.toggleAutostart() },
    { label: 'Closing the window hides it to the tray', type: 'checkbox', checked: !!v.hideToTray, click: () => h.toggleHideToTray() },
    { label: 'Open log folder', click: () => h.openLogFolder() },
    { type: 'separator' },
    { label: 'Quit', click: () => h.quit() },
  );
  return items;
}

module.exports = { trayMenuTemplate, statusLabel, PAGES };
