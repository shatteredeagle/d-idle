'use strict';
// The build icons are original pixel art made with Aseprite (assets/branding, docs/branding.md). This copies the packed set into build/;
// regenerate the set itself with tools/branding/build.sh on the worker host. Plain node, no display needed.
const fs = require('node:fs');
const path = require('node:path');

const from = path.join(__dirname, '..', '..', '..', 'assets', 'branding', 'out', 'electron');
const build = path.join(__dirname, '..', 'build');
const files = ['icon.png', 'icon.ico', 'icon.icns', 'tray.png', 'trayTemplate.png', 'trayTemplate@2x.png'];
for (const f of files) fs.copyFileSync(path.join(from, f), path.join(build, f));
console.log('copied', files.join(', '));
