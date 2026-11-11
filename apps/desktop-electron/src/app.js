'use strict';
const path = require('node:path');
const fs = require('node:fs');
const electron = require('electron');
const { app, BaseWindow, WebContentsView, Menu, protocol, session, ipcMain } = electron;
const config = require('./config');
const { parseArgs, userArgs } = require('./args');
const { normalizeAddress, pageURL } = require('./address');
const sec = require('./security');
const { Shell, POLL_MS } = require('./state');
const { appMenuTemplate } = require('./menu');

const HELPER_DIR = path.join(__dirname, 'helper');
const MIN = { width: 900, height: 600 };

// start runs the whole app and resolves to a context object (the smoke test inspects it; nothing else does).
// opts.argv overrides process.argv (tests), opts.userData overrides the data dir (tests).
async function start(opts = {}) {
  app.setName('d2idle-watch'); // userData must be <config dir>/d2idle-watch, where the Go launcher keeps config.json
  const userData = opts.userData || path.join(app.getPath('appData'), 'd2idle-watch');
  app.setPath('userData', userData);

  let args;
  try {
    args = parseArgs(opts.argv || userArgs(process.argv, app.isPackaged));
  } catch (e) {
    process.stderr.write(`d2idle-watch: ${e.message}\n`);
    app.exit(2);
    return null;
  }

  let cfg;
  try {
    cfg = config.load(userData);
  } catch (e) {
    process.stderr.write(`d2idle-watch: cannot read config: ${e.message}\n`);
    app.exit(1);
    return null;
  }
  if (args.address) {
    cfg.address = args.address;
    config.save(userData, cfg);
  }

  if (args.print) {
    if (!cfg.address) process.stderr.write('d2idle-watch: no address configured (use --address host:port)\n');
    else process.stdout.write(pageURL(cfg.address, args.page || 'watch') + '\n');
    app.exit(cfg.address ? 0 : 1);
    return null;
  }

  if (!opts.noSingleInstance && !app.requestSingleInstanceLock()) {
    app.exit(0); // the running instance gets 'second-instance'
    return null;
  }

  protocol.registerSchemesAsPrivileged([{ scheme: sec.HELPER_SCHEME, privileges: { standard: true, secure: true } }]);
  await app.whenReady();

  const ctx = { userData, cfg, args, window: null, helperView: null, gameView: null, shell: null, pendingPage: args.page || 'watch', gameShown: false };

  // Default-deny permissions on the sessions we use.
  const gameSession = session.fromPartition('persist:d2idle');
  for (const s of [session.defaultSession, gameSession]) {
    s.setPermissionRequestHandler(sec.denyPermissionRequest);
    s.setPermissionCheckHandler(sec.denyPermissionCheck);
  }
  const probeSession = session.fromPartition('d2idle-probe'); // in memory, never holds the game cookie
  const fetchFn = (url, init) => probeSession.fetch(url, init);

  // Helper pages come from a custom protocol that serves only helper/ files, with the CSP as a header too.
  protocol.handle(sec.HELPER_SCHEME, async (req) => {
    const file = sec.resolveHelperFile(HELPER_DIR, req.url);
    if (!file) return new Response('not found', { status: 404 });
    try {
      const body = await fs.promises.readFile(file);
      return new Response(body, { headers: { 'content-type': sec.MIME[path.extname(file)] || 'application/octet-stream', 'content-security-policy': sec.HELPER_CSP, 'cache-control': 'no-store' } });
    } catch (e) {
      return new Response('not found', { status: 404 });
    }
  });

  // Nothing opens a window or attaches a webview, anywhere.
  app.on('web-contents-created', (_e, wc) => {
    wc.setWindowOpenHandler(() => ({ action: 'deny' }));
    wc.on('will-attach-webview', (ev) => ev.preventDefault());
  });

  const size = config.windowSize(cfg);
  const win = new BaseWindow({ width: size.width, height: size.height, minWidth: MIN.width, minHeight: MIN.height, title: 'd.idle watch', icon: iconPath(), show: false });
  ctx.window = win;

  const helperView = new WebContentsView({ webPreferences: sec.webPreferences('helper', path.join(HELPER_DIR, 'preload.js')) });
  const gameView = new WebContentsView({ webPreferences: sec.webPreferences('game') });
  ctx.helperView = helperView;
  ctx.gameView = gameView;
  win.contentView.addChildView(helperView);
  win.contentView.addChildView(gameView);
  gameView.setVisible(false);

  const layout = () => {
    const b = win.getContentBounds();
    helperView.setBounds({ x: 0, y: 0, width: b.width, height: b.height });
    gameView.setBounds({ x: 0, y: 0, width: b.width, height: b.height });
  };
  win.on('resize', layout);
  layout();

  // Navigation lock. The helper stays on its own scheme; the game view stays on the exact configured origin.
  const lock = (wc, allowed) => {
    const deny = (e, url) => { if (!allowed(url)) e.preventDefault(); };
    wc.on('will-navigate', deny);
    wc.on('will-redirect', deny);
    wc.on('will-frame-navigate', (e) => { if (!allowed(e.url)) e.preventDefault(); });
  };
  lock(helperView.webContents, sec.isHelperURL);
  lock(gameView.webContents, (url) => sec.isAllowedNavigation(ctx.shell.state.address, url));

  const shell = new Shell({ fetchFn, onChange: (s) => onState(s) });
  ctx.shell = shell;

  const showGame = (page) => {
    const url = pageURL(shell.state.address, page || ctx.pendingPage);
    helperView.setVisible(false);
    gameView.setVisible(true);
    ctx.gameShown = true;
    return gameView.webContents.loadURL(url).catch(() => {});
  };
  const showHelper = () => {
    gameView.setVisible(false);
    helperView.setVisible(true);
    ctx.gameShown = false;
  };
  const push = () => { if (!helperView.webContents.isDestroyed()) helperView.webContents.send('d2idle:state', shell.state); };
  function onState(s) {
    push();
    if (s.status === 'up' && !ctx.gameShown) showGame();
    if (opts.onState) opts.onState(s);
  }

  // IPC: every handler checks the sender first and validates its arguments.
  const handle = (channel, fn) => ipcMain.handle(channel, (event, ...a) => {
    if (!sec.ipcSenderOk(event, helperView.webContents)) throw new Error('forbidden');
    return fn(...a);
  });
  handle('d2idle:state', () => shell.state);
  handle('d2idle:saveAddress', async (input) => {
    let origin;
    try {
      origin = normalizeAddress(input);
    } catch (e) {
      shell.setError(e.message);
      return shell.state;
    }
    cfg.address = origin;
    config.save(userData, cfg);
    gameView.webContents.session.clearCache().catch(() => {});
    shell.setAddress(origin);
    await shell.tick();
    return shell.state;
  });
  handle('d2idle:retry', () => shell.tick());
  handle('d2idle:openPage', (name) => { ctx.api.openPage(String(name)); });

  // Actions shared by the menu, second instances and (PR2) the tray.
  ctx.api = {
    openPage(name) {
      pageURL('http://x', name); // throws on an unknown page
      ctx.pendingPage = name;
      if (shell.state.status === 'up') showGame(name);
      ctx.api.show();
    },
    changeAddress() { showHelper(); shell.state = { ...shell.state, status: 'setup' }; push(); ctx.api.show(); },
    reload() { if (ctx.gameShown) gameView.webContents.reload(); else shell.tick(); },
    show() { if (win.isMinimized()) win.restore(); win.show(); win.focus(); },
    quit() { app.quit(); },
  };
  Menu.setApplicationMenu(Menu.buildFromTemplate(appMenuTemplate(ctx.api)));

  app.on('second-instance', (_e, argv) => {
    try {
      const a = parseArgs(userArgs(argv, app.isPackaged));
      if (a.setup) ctx.api.changeAddress();
      else if (a.page) ctx.api.openPage(a.page);
      else ctx.api.show();
    } catch (e) {
      ctx.api.show();
    }
  });

  let saveTimer;
  win.on('resize', () => {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => {
      if (win.isDestroyed() || win.isMaximized() || win.isFullScreen()) return;
      const { width, height } = win.getContentBounds();
      cfg.window = { width, height };
      try { config.save(userData, cfg); } catch (e) { /* a read-only profile must not break the app */ }
    }, 800);
    saveTimer.unref?.();
  });
  win.on('closed', () => { ctx.closed = true; });
  app.on('window-all-closed', () => app.quit());

  await helperView.webContents.loadURL(sec.HELPER_URL);
  shell.setAddress(cfg.address || '');
  if (args.setup) shell.state = { ...shell.state, status: 'setup' };
  push();
  win.show();
  if (!args.setup && cfg.address) shell.tick();
  ctx.timer = setInterval(() => { if (!ctx.gameShown || opts.keepPolling) shell.tick(); }, POLL_MS);
  ctx.timer.unref?.();
  return ctx;
}

function iconPath() {
  const p = path.join(__dirname, '..', 'build', 'icon.png');
  return fs.existsSync(p) ? p : undefined;
}

module.exports = { start };
