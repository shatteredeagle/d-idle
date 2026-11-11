'use strict';
// Every security decision of the shell as plain functions, so tests need no display. main.js and views.js only wire
// them to Electron. Rules: docs/desktop-electron.md section "Security posture".
const path = require('node:path');

const HELPER_SCHEME = 'd2idle-app';
const HELPER_ORIGIN = `${HELPER_SCHEME}://helper`;
const HELPER_URL = `${HELPER_ORIGIN}/index.html`;

// CSP of the helper page: it talks to the main process only through the preload, never fetches anything.
const HELPER_CSP = "default-src 'none'; style-src 'self'; script-src 'self'; connect-src 'none'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'";

// webPreferences per view kind. The game view (remote page) has NO preload; only the helper has one.
function webPreferences(kind, preloadPath) {
  const base = {
    contextIsolation: true,
    nodeIntegration: false,
    nodeIntegrationInWorker: false,
    nodeIntegrationInSubFrames: false,
    sandbox: true,
    webSecurity: true,
    allowRunningInsecureContent: false,
    webviewTag: false,
    experimentalFeatures: false,
    enableBlinkFeatures: '',
    devTools: false,
    spellcheck: false,
  };
  if (kind === 'helper') return { ...base, preload: preloadPath };
  if (kind === 'game') return { ...base, partition: 'persist:d2idle' };
  throw new Error(`unknown view kind ${kind}`);
}

// isAllowedNavigation: strict origin equality with the configured origin (an https page may not go to http, a port
// change is a different origin). A malformed or opaque URL is refused.
function isAllowedNavigation(origin, url) {
  if (!origin) return false;
  try {
    const u = new URL(url);
    return u.origin !== 'null' && u.origin === origin && u.username === '' && u.password === '';
  } catch (e) {
    return false;
  }
}

function isHelperURL(url) {
  try {
    const u = new URL(url);
    return u.protocol === HELPER_SCHEME + ':' && u.host === 'helper';
  } catch (e) {
    return false;
  }
}

// ipcSenderOk is the check every IPC handler runs first: the frame must be the helper page, in the helper view.
function ipcSenderOk(event, helperWebContents) {
  const f = event && event.senderFrame;
  return !!f && !!helperWebContents && event.sender === helperWebContents && isHelperURL(f.url);
}

// resolveHelperFile maps d2idle-app://helper/<file> to a file inside root, or null. Normalised so ".." cannot escape.
function resolveHelperFile(root, url) {
  let u;
  try {
    u = new URL(url);
  } catch (e) {
    return null;
  }
  if (u.protocol !== HELPER_SCHEME + ':' || u.host !== 'helper') return null;
  let rel;
  try {
    rel = decodeURIComponent(u.pathname);
  } catch (e) {
    return null;
  }
  if (rel === '/' || rel === '') rel = '/index.html';
  if (rel.includes('\0') || rel.includes('\\')) return null;
  const full = path.resolve(root, '.' + path.posix.normalize(rel));
  const base = path.resolve(root) + path.sep;
  return full.startsWith(base) ? full : null;
}

const MIME = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png' };

// Permissions: the watch pages need none, so everything is denied.
const denyPermissionRequest = (_wc, _permission, callback) => callback(false);
const denyPermissionCheck = () => false;

// openExternalAllowed is for the one deliberate user action (the update menu item): http(s) only.
function openExternalAllowed(url) {
  try {
    const u = new URL(url);
    return u.protocol === 'https:' || u.protocol === 'http:';
  } catch (e) {
    return false;
  }
}

module.exports = {
  HELPER_SCHEME, HELPER_ORIGIN, HELPER_URL, HELPER_CSP, MIME,
  webPreferences, isAllowedNavigation, isHelperURL, ipcSenderOk, resolveHelperFile,
  denyPermissionRequest, denyPermissionCheck, openExternalAllowed,
};
