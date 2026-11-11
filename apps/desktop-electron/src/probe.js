'use strict';

// probe asks GET <origin>/watch once, without redirects or cookies. Below 500 is up (a login redirect counts), 5xx is
// warming (the proxy answers, the game side is not ready), no connection is down. fetchFn is injected so tests and the
// main process (net.fetch on a throwaway session) share the code.
async function probe(fetchFn, origin, timeoutMs = 3000) {
  try {
    const res = await fetchFn(origin + '/watch', {
      method: 'GET',
      redirect: 'manual',
      credentials: 'omit',
      signal: AbortSignal.timeout(timeoutMs),
    });
    if (res.body && typeof res.body.cancel === 'function') res.body.cancel().catch(() => {});
    return res.status >= 500 ? 'warming' : 'up';
  } catch (e) {
    // V (smoke test, Electron 45 on the worker host): net.fetch with redirect:'manual' REJECTS on a 3xx instead of
    // returning an opaque redirect. A redirect means the server answered (login redirect), so it is up.
    if (/redirect was cancelled/i.test(String(e && e.message))) return 'up';
    return 'down';
  }
}

module.exports = { probe };
