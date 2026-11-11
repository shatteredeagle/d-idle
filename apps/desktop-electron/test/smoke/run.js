'use strict';
// Starts smoke-main.js in Electron, under xvfb-run when there is no display. Exit code is the smoke result.
const { spawnSync } = require('node:child_process');
const path = require('node:path');
const electron = require('electron');

const main = path.join(__dirname, 'smoke-main.js');
// The Chromium sandbox is decided before any app code runs, so the switch must be on the command line (an
// appendSwitch in the main script is too late). Only the CI smoke sets D2E_SMOKE_NO_SANDBOX; the app never does.
const flags = process.env.D2E_SMOKE_NO_SANDBOX ? ['--no-sandbox'] : [];
const headless = process.platform === 'linux' && !process.env.DISPLAY;
const cmd = headless ? ['xvfb-run', ['-a', electron, ...flags, main]] : [electron, [...flags, main]];
const r = spawnSync(cmd[0], cmd[1], { stdio: 'inherit', timeout: 120000 });
process.exit(r.status === null ? 1 : r.status);
