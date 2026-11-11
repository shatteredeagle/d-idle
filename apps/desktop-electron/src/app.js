'use strict';
const path = require('node:path');
const fs = require('node:fs');
const electron = require('electron');
const { app, BaseWindow, WebContentsView, Menu, Tray, nativeImage, dialog, protocol, session, ipcMain, shell: shell_ } = electron;
const config = require('./config');
const { parseArgs, userArgs } = require('./args');
const { normalizeAddress, pageURL } = require('./address');
const sec = require('./security');
const { Shell, POLL_MS } = require('./state');
const { appMenuTemplate } = require('./menu');
const { trayMenuTemplate } = require('./tray');
const { createAutostart } = require('./autostart');
const plat = require('./platform');
const retry = require('./retry');
const logs = require('./logs');
const { checkForUpdate, builtinFeed } = require('./update');

const HELPER_DIR = path.join(__dirname, 'helper');
const MIN = { width: 900, height: 600 };
const UPDATE_EVERY_MS = 24 * 60 * 60 * 1000;

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

  const ctx = { loadAttempts: 0, loadMessage: '', retryPending: false, userData, cfg, args, window: null, helperView: null, gameView: null, shell: null, pendingPage: args.page || 'watch', gameShown: false, quitting: false, tray: null, update: { available: false } };

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

  function layout() {
    const b = win.getContentBounds();
    helperView.setBounds({ x: 0, y: 0, width: b.width, height: b.height });
    gameView.setBounds({ x: 0, y: 0, width: b.width, height: b.height });
  }
  for (const ev of ['resize', 'maximize', 'unmaximize', 'enter-full-screen', 'leave-full-screen', 'show', 'restore']) win.on(ev, layout);
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

  // Diagnostics: a white window gives no error, so load failures and renderer crashes always go to the log file
  // (logs.js; kept short); --debug adds loads, console warnings and errors, and D2IDLE_SHOT=<dir> writes a PNG of each view.
  const logFile = opts.logFile || logs.logPath();
  const log = (line, verbose) => {
    if (verbose && !args.debug) return;
    try {
      fs.mkdirSync(path.dirname(logFile), { recursive: true });
      if (fs.existsSync(logFile) && fs.statSync(logFile).size > 200000) fs.renameSync(logFile, logFile + '.old');
      fs.appendFileSync(logFile, `${new Date().toISOString()} ${line}\n`);
    } catch (e) { /* logging must never break the app */ }
  };
  ctx.logFile = logFile;
  // A load that never reports anything (the macOS Local Network permission can stall it silently) is treated as failed.
  const armStall = () => {
    clearTimeout(ctx.stallTimer);
    ctx.stallTimer = setTimeout(() => { if (gameView.webContents.isLoading()) ctx.api.gameLoadFailed(`no answer after ${(opts.stallMs ?? retry.STALL_MS) / 1000} s`); }, opts.stallMs ?? retry.STALL_MS);
    ctx.stallTimer.unref?.();
  };
  for (const [name, view] of [['helper', helperView], ['game', gameView]]) {
    const wc = view.webContents;
    let failed = false; // Chromium can still fire did-finish-load after did-fail-load (the error page), which is no success
    wc.on('did-start-navigation', (e) => { if (e.isMainFrame && !e.isSameDocument) { failed = false; if (name === 'game') armStall(); } });
    wc.on('did-stop-loading', () => { if (name === 'game') clearTimeout(ctx.stallTimer); });
    wc.on('did-fail-load', (_e, code, desc, url, isMain) => {
      if (isMain) failed = true;
      log(`${name} did-fail-load ${code} ${desc} ${url}`);
      if (isMain && name === 'game' && retry.isRealFailure(code)) ctx.api.gameLoadFailed(desc);
    });
    wc.on('did-finish-load', () => {
      if (failed) return;
      log(`${name} loaded ${wc.getURL()}`, true);
      if (name === 'game') {
        wc.executeJavaScript('document.body ? document.body.innerText.trim().length + document.body.children.length : 0').then((n) => {
          if (n === 0 && ctx.gameShown) { log('game blank body after load'); ctx.api.gameLoadFailed('the page was empty'); }
        }).catch(() => {});
        ctx.loadAttempts = 0; ctx.loadMessage = ''; ctx.retryPending = false; push(); }
    });
    for (const ev of ['did-start-loading', 'did-stop-loading', 'dom-ready']) wc.on(ev, () => log(`${name} ${ev}`, true));
    wc.on('unresponsive', () => log(`${name} unresponsive`));
    wc.on('render-process-gone', (_e, d) => log(`${name} render-process-gone ${d.reason}`));
    wc.on('preload-error', (_e, file, err) => log(`${name} preload-error ${file} ${err && err.message}`));
    wc.on('console-message', (e) => { if (e.level === 'error' || e.level === 'warning') log(`${name} console ${e.level}: ${e.message}`, true); });
  }
  if (process.env.D2IDLE_SHOT) {
    setTimeout(async () => {
      for (const [name, view] of [['helper', helperView], ['game', gameView]]) {
        try { fs.writeFileSync(path.join(process.env.D2IDLE_SHOT, `${name}.png`), (await view.webContents.capturePage()).toPNG()); log(`${name} shot written, visible=${view.getVisible()}`); } catch (e) { log(`${name} shot failed ${e.message}`); }
      }
    }, 10000).unref?.();
  }

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
  // loadMessage (a failed game load) rides along as the error line; the poll rewrites shell.state, so it is kept apart.
  const push = () => { if (!helperView.webContents.isDestroyed()) helperView.webContents.send('d2idle:state', { ...shell.state, error: shell.state.error || ctx.loadMessage || '' }); };
  function onState(s) {
    push();
    refreshTray();
    if (s.status === 'up' && !ctx.gameShown && !ctx.retryPending) showGame();
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
    clearTimeout(ctx.retryTimer);
    ctx.retryPending = false; ctx.loadMessage = ''; ctx.loadAttempts = 0;
    gameView.webContents.session.clearCache().catch(() => {});
    showHelper();
    shell.setAddress(origin);
    await shell.tick();
    return shell.state;
  });
  handle('d2idle:retry', () => shell.tick());
  handle('d2idle:openPage', (name) => { ctx.api.openPage(String(name)); });

  // Tray: icon, menu with the pages, a reachability line, autostart and hide-to-tray toggles, update check.
  const autostart = (opts.autostart) || createAutostart(app);
  const hideToTray = () => cfg.hideToTray !== false;
  function refreshTray() {
    if (!ctx.tray || ctx.tray.isDestroyed()) return;
    let on = false;
    try { on = autostart.get(); } catch (e) { /* unreadable: show unchecked */ }
    ctx.trayTemplate = trayMenuTemplate({ state: shell.state, autostart: on, hideToTray: hideToTray(), update: ctx.update }, ctx.api);
    ctx.tray.setContextMenu(Menu.buildFromTemplate(ctx.trayTemplate));
    ctx.tray.setToolTip('d.idle watch: ' + ctx.trayTemplate[3].label);
  }
  if (!opts.noTray) {
    const img = nativeImage.createFromPath(path.join(__dirname, '..', 'build', plat.trayImageName(process.platform)));
    ctx.tray = new Tray(img.isEmpty() ? nativeImage.createEmpty() : img);
    ctx.tray.on('click', () => ctx.api.show());
  }

  // Update check: check only. It tells the user a newer version exists; it never downloads or installs anything.
  async function runUpdateCheck(manual) {
    try {
      ctx.update = await checkForUpdate({ feed: opts.updateFeed || cfg.updateFeed || builtinFeed(), keys: opts.updateKeys, fetchFn, currentVersion: app.getVersion() });
      if (manual && !ctx.update.available) await dialog.showMessageBox({ type: 'info', message: 'd.idle watch is up to date.' });
    } catch (e) {
      ctx.updateError = String(e.message || e);
      if (manual) await dialog.showMessageBox({ type: 'info', message: 'Could not check for updates.', detail: ctx.updateError });
    }
    refreshTray();
  }

  const dock = plat.dockControl(app);

  // A failed game load never leaves a blank window: show the helper with the reason and retry with backoff. The retry
  // loads the page while the helper stays visible and swaps the views only when it succeeded (no flicker, no relaunch).
  function gameLoadFailed(desc) {
    clearTimeout(ctx.retryTimer);
    ctx.loadMessage = retry.loadFailureMessage(process.platform, desc);
    ctx.retryPending = true;
    showHelper();
    push();
    const wait = retry.backoffMs(ctx.loadAttempts++);
    log(`game load failed (${desc}); retry in ${wait} ms`);
    ctx.retryTimer = setTimeout(() => {
      armStall();
      if (ctx.gameShown || shell.state.status !== 'up') return; // the helper poll drives other states
      gameView.webContents.loadURL(pageURL(shell.state.address, ctx.pendingPage)).then(() => {
        if (!ctx.gameShown) { helperView.setVisible(false); gameView.setVisible(true); ctx.gameShown = true; }
      }).catch(() => {}); // did-fail-load schedules the next try
    }, wait);
    ctx.retryTimer.unref?.();
  }

  // Actions shared by the menu, second instances and the tray.
  ctx.api = {
    gameLoadFailed,
    openPage(name) {
      pageURL('http://x', name); // throws on an unknown page
      ctx.pendingPage = name;
      if (shell.state.status === 'up') showGame(name);
      ctx.api.show();
    },
    changeAddress() { showHelper(); shell.state = { ...shell.state, status: 'setup' }; push(); ctx.api.show(); },
    reload() { if (ctx.gameShown) gameView.webContents.reload(); else shell.tick(); },
    show() { dock.show(); if (win.isMinimized()) win.restore(); win.show(); win.focus(); },
    quit() { ctx.quitting = true; app.quit(); },
    toggleAutostart() {
      try { autostart.set(!autostart.get()); } catch (e) { dialog.showErrorBox('Start with the system', String(e.message || e)); }
      refreshTray();
    },
    toggleHideToTray() {
      cfg.hideToTray = !hideToTray();
      try { config.save(userData, cfg); } catch (e) { /* keep the in-memory setting */ }
      refreshTray();
    },
    openLogFolder() {
      try { fs.mkdirSync(path.dirname(logFile), { recursive: true }); } catch (e) { /* shell.openPath reports it */ }
      shell_.openPath(path.dirname(logFile));
    },
    checkUpdate: () => runUpdateCheck(true),
    showUpdate() {
      dialog.showMessageBox({
        type: 'info',
        message: `d.idle watch ${ctx.update.version} is available.`,
        detail: `${ctx.update.notes}\n\nGet the new build from where you got this one. This app downloads and installs nothing.`.trim(),
      });
    },
  };
  Menu.setApplicationMenu(Menu.buildFromTemplate(appMenuTemplate(ctx.api, process.platform)));

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
  win.on('close', (e) => {
    if (plat.hideOnClose({ platform: process.platform, hasTray: !!ctx.tray, hideToTray: hideToTray(), quitting: ctx.quitting })) { e.preventDefault(); win.hide(); dock.hide(); }
  });
  win.on('closed', () => { ctx.closed = true; });
  app.on('before-quit', () => { ctx.quitting = true; });
  app.on('window-all-closed', () => { if (plat.quitOnAllClosed(process.platform, !!ctx.tray)) app.quit(); });
  app.on('activate', () => ctx.api.show()); // macOS: Dock icon click

  await helperView.webContents.loadURL(sec.HELPER_URL);
  shell.setAddress(cfg.address || '');
  if (args.setup) shell.state = { ...shell.state, status: 'setup' };
  push();
  refreshTray();
  if (!(args.hidden && ctx.tray)) { win.show(); layout(); } else dock.hide();
  log(`start: window ${JSON.stringify(win.getBounds())}, content ${JSON.stringify(win.getContentBounds())}, address ${cfg.address || '(none)'}`, true);
  if (!args.setup && cfg.address) shell.tick();
  // Poll fast while the helper is shown; slowly while the game page is up, so the tray status stays current.
  let n = 0;
  ctx.timer = setInterval(() => { if (!ctx.gameShown || ++n % 5 === 0) shell.tick(); }, POLL_MS);
  ctx.timer.unref?.();
  if (cfg.updateCheck !== false && !opts.noUpdate && (opts.updateFeed || cfg.updateFeed || builtinFeed())) {
    setTimeout(() => runUpdateCheck(false), opts.updateDelayMs ?? 15000).unref?.();
    setInterval(() => runUpdateCheck(false), UPDATE_EVERY_MS).unref?.();
  }
  ctx.runUpdateCheck = runUpdateCheck;
  return ctx;
}

function iconPath() {
  const p = path.join(__dirname, '..', 'build', 'icon.png');
  return fs.existsSync(p) ? p : undefined;
}

module.exports = { start };
