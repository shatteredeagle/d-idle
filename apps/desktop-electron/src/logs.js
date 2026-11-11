'use strict';
// Log file location per platform (pure). The log holds load failures always and, with --debug, every page event.
const os = require('node:os');
const path = require('node:path');

function logDir(platform = process.platform, env = process.env, home = os.homedir()) {
  if (platform === 'darwin') return path.join(home, 'Library', 'Logs', 'd2idle-watch');
  if (platform === 'win32') return path.join(env.APPDATA || path.join(home, 'AppData', 'Roaming'), 'd2idle-watch', 'logs');
  return path.join(env.XDG_STATE_HOME || path.join(home, '.local', 'state'), 'd2idle-watch');
}

const logPath = (...a) => path.join(logDir(...a), 'main.log');

module.exports = { logDir, logPath };
