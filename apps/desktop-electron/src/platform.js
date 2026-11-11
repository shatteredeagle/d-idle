'use strict';
// Platform rules kept pure so tests can run them on any OS. Only macOS differs today.

const isMac = (platform = process.platform) => platform === 'darwin';

// window-all-closed: macOS apps with a menu bar icon keep running after the last window closes (Cmd+Q quits).
const quitOnAllClosed = (platform, hasTray) => !(isMac(platform) && hasTray);

// Closing the window hides it while a tray exists and the user asked for it; on macOS always (the window must
// survive so the Dock icon or the menu bar icon can bring it back).
const hideOnClose = ({ platform, hasTray, hideToTray, quitting }) => !quitting && hasTray && (hideToTray || isMac(platform));

// Tray image file names: macOS needs a monochrome "Template" image (the @2x sibling is picked up by name).
const trayImageName = (platform) => (isMac(platform) ? 'trayTemplate.png' : 'tray.png');

// The Dock icon is hidden while the app lives only in the menu bar (window hidden), shown with the window.
function dockControl(app, platform = process.platform) {
  const dock = isMac(platform) && app.dock ? app.dock : null;
  return { hide: () => dock && dock.hide(), show: () => dock && dock.show() };
}

module.exports = { isMac, quitOnAllClosed, hideOnClose, trayImageName, dockControl };
