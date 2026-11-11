'use strict';
// Starts smoke-main.js in Electron, under xvfb-run when there is no display. Exit code is the smoke result.
const { spawnSync } = require('node:child_process');
const path = require('node:path');
const electron = require('electron');

const main = path.join(__dirname, 'smoke-main.js');
// The Chromium sandbox is decided before any app code runs, so the switch must be on the command line (an
// appendSwitch in the main script is too late). Only the CI smoke sets D2E_SMOKE_NO_SANDBOX; the app never does.
// Test harness only, never the shipped app: the hosted runner has no GPU, and capturePage through the Viz/GPU path
// under Xvfb fails there (UnknownVizError), so the smoke renders in software.
const flags = ['--disable-gpu', '--disable-dev-shm-usage', ...(process.env.D2E_SMOKE_NO_SANDBOX ? ['--no-sandbox'] : [])];
const headless = process.platform === 'linux' && !process.env.DISPLAY;
const cmd = headless ? ['xvfb-run', ['-a', '-s', '-screen 0 1280x1024x24', electron, ...flags, main]] : [electron, [...flags, main]];
const r = spawnSync(cmd[0], cmd[1], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'], timeout: 120000 });
process.stdout.write(r.stdout || '');
// xvfb-run can fail while removing its temp dir after a good run; the smoke result is its own last line.
const passed = /^SMOKE PASS$/m.test(r.stdout || '');
process.exit(passed ? 0 : 1);
