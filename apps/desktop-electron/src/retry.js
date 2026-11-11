'use strict';
// Retry rules for a failed game page load (pure, so a test runs them without Electron). On macOS the very first load
// can fail until the user allows "Local Network" for the app (V: user report 2026-11-10); the app must show a message
// and keep trying instead of leaving a blank window.
const FIRST_MS = 3000;
const MAX_MS = 30000;
const STALL_MS = 20000; // a page load that reports nothing for this long counts as failed

const backoffMs = (attempt) => Math.min(FIRST_MS * 2 ** Math.max(0, attempt), MAX_MS);

// Chromium error -3 is "aborted by a newer navigation": not a failure.
const isRealFailure = (code) => code !== -3;

function loadFailureMessage(platform, desc) {
  const base = `Waiting for the server (${desc || 'page not loaded'}).`;
  return platform === 'darwin'
    ? `${base} On macOS, allow Local Network access for d.idle watch when prompted (System Settings, Privacy & Security, Local Network); this page retries by itself.`
    : `${base} This page retries by itself.`;
}

module.exports = { backoffMs, isRealFailure, loadFailureMessage, FIRST_MS, MAX_MS, STALL_MS };
