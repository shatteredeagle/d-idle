'use strict';
// "Start with the system". Windows and macOS: Electron's login item. Linux (Electron has no login item there): an
// XDG autostart entry in ~/.config/autostart. The entry starts the app with --hidden, so it sits in the tray.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const ENTRY = 'd2idle-watch.desktop';

// quoteExec quotes one Exec argument per the Desktop Entry spec (the AppImage path may contain spaces).
const quoteExec = (s) => (/[\s"'\\><~|&;$*?#()`]/.test(s) ? '"' + s.replace(/(["`$\\])/g, '\\$1') + '"' : s);

function desktopEntry(execPath) {
  return ['[Desktop Entry]', 'Type=Application', 'Name=d.idle watch', 'Comment=Start d.idle watch in the tray', `Exec=${quoteExec(execPath)} --hidden`, 'Terminal=false', 'X-GNOME-Autostart-enabled=true', ''].join('\n');
}

function entryPath(env = process.env, home = os.homedir()) {
  return path.join(env.XDG_CONFIG_HOME || path.join(home, '.config'), 'autostart', ENTRY);
}

// The binary to start: the AppImage file when running from one (process.execPath is then inside a temp mount).
function execTarget(env = process.env, execPath = process.execPath) {
  return env.APPIMAGE || execPath;
}

function createAutostart(app, { platform = process.platform, env = process.env, home = os.homedir() } = {}) {
  if (platform === 'linux') {
    const file = entryPath(env, home);
    return {
      supported: true,
      get: () => fs.existsSync(file),
      set(on) {
        if (!on) return fs.rmSync(file, { force: true });
        fs.mkdirSync(path.dirname(file), { recursive: true });
        fs.writeFileSync(file, desktopEntry(execTarget(env, process.execPath)));
      },
    };
  }
  return {
    supported: true,
    get: () => app.getLoginItemSettings().openAtLogin,
    set: (on) => app.setLoginItemSettings({ openAtLogin: on, args: ['--hidden'] }),
  };
}

module.exports = { desktopEntry, entryPath, execTarget, createAutostart, quoteExec, ENTRY };
