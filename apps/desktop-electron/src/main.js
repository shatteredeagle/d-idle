'use strict';
require('./app').start().catch((e) => {
  process.stderr.write(`d2idle-watch: ${e && e.stack ? e.stack : e}\n`);
  require('electron').app.exit(1);
});
