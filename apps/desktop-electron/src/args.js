'use strict';
const { PAGES, normalizeAddress } = require('./address');

// Chromium switches that are passed through (Chromium reads them itself from the real argv; the parser only skips
// them). Everything else unknown is still an error, so a typo does not silently do nothing.
const CHROMIUM_FLAGS = /^--(enable-logging|log-file|log-level|v|vmodule|disable-gpu|disable-software-rasterizer|remote-debugging-port)(=.*)?$/;

// parseArgs reads the user-visible flags (same as apps/desktop): --setup, --address host:port, --page NAME, --print, --hidden.
// Chromium switches and anything else are rejected: the app forwards no user switches to Chromium.
// argv is the list WITHOUT the executable and script (see userArgs).
function parseArgs(argv) {
  const out = { setup: false, print: false, hidden: false, debug: false, address: '', page: '' };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const [flag, inline] = a.startsWith('--') && a.includes('=') ? [a.slice(0, a.indexOf('=')), a.slice(a.indexOf('=') + 1)] : [a, undefined];
    const value = () => {
      if (inline !== undefined) return inline;
      if (i + 1 >= argv.length) throw new Error(`${flag} needs a value`);
      return argv[++i];
    };
    switch (flag) {
      case '--setup': out.setup = true; break;
      case '--print': out.print = true; break;
      case '--debug': out.debug = true; break; // write the verbose log (docs/desktop-electron.md)
      case '--hidden': out.hidden = true; break; // start in the tray (used by the autostart entry)
      case '--address': out.address = normalizeAddress(value()); break;
      case '--page': {
        const p = value();
        if (!Object.hasOwn(PAGES, p)) throw new Error(`unknown page "${p}" (watch, inventory, policy, return)`);
        out.page = p;
        break;
      }
      default:
        if (CHROMIUM_FLAGS.test(a)) break;
        throw new Error(`unknown argument ${JSON.stringify(a)}`);
    }
  }
  return out;
}

// userArgs drops the electron binary (and the app path when running unpackaged) from process.argv.
function userArgs(argv, isPackaged) {
  return argv.slice(isPackaged ? 1 : 2);
}

module.exports = { parseArgs, userArgs };
