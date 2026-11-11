'use strict';
// The shell's connection state machine, free of Electron so it can be tested. Statuses: setup (no address),
// checking, down, warming, up. The owner calls tick() every POLL_MS while the helper is shown.
const { probe } = require('./probe');

const POLL_MS = 3000;

class Shell {
  constructor({ fetchFn, onChange }) {
    this.fetchFn = fetchFn;
    this.onChange = onChange || (() => {});
    this.state = { status: 'setup', address: '', error: '' };
  }

  setAddress(address) {
    this.state = { status: address ? 'checking' : 'setup', address, error: '' };
    this.onChange(this.state);
  }

  setError(error) {
    this.state = { ...this.state, error };
    this.onChange(this.state);
  }

  async tick() {
    if (!this.state.address || this.state.status === 'setup') return this.state; // setup: the user is typing
    const address = this.state.address;
    const status = await probe(this.fetchFn, address);
    if (address !== this.state.address) return this.state; // changed while probing
    if (status !== this.state.status || this.state.error) {
      this.state = { status, address, error: '' };
      this.onChange(this.state);
    }
    return this.state;
  }
}

module.exports = { Shell, POLL_MS };
