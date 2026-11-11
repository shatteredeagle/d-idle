'use strict';
const path = require('node:path');
const { flipFuses, FuseVersion, FuseV1Options } = require('@electron/fuses');
const flips = require('./fuses');

function executable(context) {
  const { electronPlatformName: os, appOutDir } = context;
  const name = context.packager.appInfo.productFilename;
  if (os === 'darwin') return path.join(appOutDir, `${name}.app`);
  if (os === 'win32') return path.join(appOutDir, `${name}.exe`);
  return path.join(appOutDir, context.packager.executableName);
}

exports.default = async function afterPack(context) {
  const fuses = { version: FuseVersion.V1 };
  for (const [name, on] of Object.entries(flips)) {
    if (!(name in FuseV1Options)) throw new Error(`unknown fuse ${name}`);
    fuses[FuseV1Options[name]] = on;
  }
  await flipFuses(executable(context), fuses);
};
