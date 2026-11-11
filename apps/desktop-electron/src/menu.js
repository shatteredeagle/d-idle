'use strict';
const { PAGES } = require('./address');

const LABELS = { watch: 'Watch', inventory: 'Inventory', policy: 'Policy', return: 'Return screen' };

// pageItems is the shared Watch/Inventory/Policy/Return list (application menu and tray). h.openPage(name).
function pageItems(h) {
  return Object.keys(PAGES).map((name) => ({ label: LABELS[name], click: () => h.openPage(name) }));
}

// appMenuTemplate is the native application menu (plain data, so tests can read it without Electron).
function appMenuTemplate(h) {
  return [
    {
      label: 'd.idle',
      submenu: [
        ...pageItems(h),
        { type: 'separator' },
        { label: 'Change address', click: () => h.changeAddress() },
        { label: 'Reload', accelerator: 'CmdOrCtrl+R', click: () => h.reload() },
        { type: 'separator' },
        { label: 'Quit', accelerator: 'CmdOrCtrl+Q', click: () => h.quit() },
      ],
    },
  ];
}

module.exports = { LABELS, pageItems, appMenuTemplate };
