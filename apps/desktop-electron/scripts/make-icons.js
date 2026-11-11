'use strict';
// Renders build/icon.svg to build/icon.png (512), build/tray.png (32), build/icon.ico and build/icon.icns, and build/tray-template.svg to build/trayTemplate.png (+@2x).
// Run on a machine with a display or Xvfb: `xvfb-run -a -s '-screen 0 1280x1024x24' npx electron scripts/make-icons.js`. The outputs are committed
// (small); this script exists so they can be regenerated from the SVG.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { app, BrowserWindow } = require('electron');

const build = path.join(__dirname, '..', 'build');
const dataURI = (file) => 'data:image/svg+xml;base64,' + fs.readFileSync(path.join(build, file)).toString('base64');

async function render(size, file = 'icon.svg') {
  const uri = dataURI(file);
  const win = new BrowserWindow({ show: false, width: size, height: size, useContentSize: true, transparent: true, frame: false, backgroundColor: '#00000000', webPreferences: { sandbox: true } });
  const page = path.join(os.tmpdir(), `d2idle-icon-${process.pid}.html`);
  fs.writeFileSync(page, `<body style="margin:0;background:transparent"><img src="${uri}" width="${size}" height="${size}" style="display:block">`);
  await win.loadFile(page).catch((e) => { throw new Error('size ' + size + ': ' + e.message); });
  fs.rmSync(page, { force: true });
  await new Promise((r) => setTimeout(r, 250));
  const png = (await win.webContents.capturePage({ x: 0, y: 0, width: size, height: size })).toPNG();
  win.destroy();
  return png;
}

// ICO with PNG-compressed entries (supported since Windows Vista).
function ico(entries) {
  const head = Buffer.alloc(6);
  head.writeUInt16LE(1, 2);
  head.writeUInt16LE(entries.length, 4);
  let offset = 6 + 16 * entries.length;
  const dirs = entries.map(({ size, png }) => {
    const d = Buffer.alloc(16);
    d[0] = size >= 256 ? 0 : size;
    d[1] = size >= 256 ? 0 : size;
    d.writeUInt16LE(1, 4);
    d.writeUInt16LE(32, 6);
    d.writeUInt32LE(png.length, 8);
    d.writeUInt32LE(offset, 12);
    offset += png.length;
    return d;
  });
  return Buffer.concat([head, ...dirs, ...entries.map((e) => e.png)]);
}

// ICNS with PNG-compressed entries (macOS 10.7 and later): 128, 256, 512 and 1024 px (ic07 to ic10), type then big-endian length.
function icns(entries) {
  const types = { 128: 'ic07', 256: 'ic08', 512: 'ic09', 1024: 'ic10' };
  const chunks = entries.map(({ size, png }) => {
    const head = Buffer.alloc(8);
    head.write(types[size], 0, 'ascii');
    head.writeUInt32BE(8 + png.length, 4);
    return Buffer.concat([head, png]);
  });
  const total = 8 + chunks.reduce((n, c) => n + c.length, 0);
  const head = Buffer.alloc(8);
  head.write('icns', 0, 'ascii');
  head.writeUInt32BE(total, 4);
  return Buffer.concat([head, ...chunks]);
}

app.on('window-all-closed', () => {}); // default would quit between renders
app.whenReady().then(async () => {
  const sizes = [16, 24, 32, 48, 64, 128, 256];
  const entries = [];
  for (const size of sizes) entries.push({ size, png: await render(size) });
  fs.writeFileSync(path.join(build, 'icon.ico'), ico(entries));
  fs.writeFileSync(path.join(build, 'icon.png'), await render(512));
  fs.writeFileSync(path.join(build, 'tray.png'), entries.find((e) => e.size === 32).png);
  // macOS: app icon (.icns) and the menu bar template image (black on transparent, 22 pt at @1x and @2x).
  const mac = [];
  for (const size of [128, 256, 512, 1024]) mac.push({ size, png: await render(size) });
  fs.writeFileSync(path.join(build, 'icon.icns'), icns(mac));
  fs.writeFileSync(path.join(build, 'trayTemplate.png'), await render(22, 'tray-template.svg'));
  fs.writeFileSync(path.join(build, 'trayTemplate' + String.fromCharCode(64) + '2x.png'), await render(44, 'tray-template.svg'));
  app.exit(0);
}).catch((e) => { console.error(e); app.exit(1); });
