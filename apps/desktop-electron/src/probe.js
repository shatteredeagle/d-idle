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
    return 'down';
  }
}

module.exports = { probe };
