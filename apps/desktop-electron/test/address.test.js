'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { normalizeAddress, pageURL } = require('../src/address');

test('normalizeAddress accepts (vectors of apps/desktop/desktop_test.go)', () => {
  const ok = {
    'game.example:8891': 'http://game.example:8891',
    '  GAME.example:8891/watch?x=1 ': 'http://game.example:8891',
    'https://Game.example': 'https://game.example',
    'http://game.example:80': 'http://game.example:80',
    '[::1]:8891': 'http://[::1]:8891',
  };
  for (const [input, want] of Object.entries(ok)) assert.equal(normalizeAddress(input), want, input);
});

test('normalizeAddress refuses', () => {
  for (const bad of ['', '  ', 'ftp://x:1', 'http://user:pass@host:1', 'host:99999', 'host:abc', 'http://', 'host:0']) {
    assert.throws(() => normalizeAddress(bad), undefined, JSON.stringify(bad));
  }
});

test('pageURL', () => {
  assert.equal(pageURL('http://h:1', 'inventory'), 'http://h:1/watch/inventory');
  assert.equal(pageURL('http://h:1', 'return'), 'http://h:1/watch/return');
  assert.throws(() => pageURL('http://h:1', 'nope'));
  assert.throws(() => pageURL('http://h:1', '__proto__'));
});
