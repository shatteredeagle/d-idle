'use strict';
// Electron fuses flipped at package time, by FuseV1Options name (afterPack maps them; plain data so a test needs no
// install). enableCookieEncryption stays off: it would tie the profile to the OS keyring and the login cookie is
// all it protects (docs/desktop-electron.md, decision 2026-11-10).
module.exports = {
  RunAsNode: false,
  EnableNodeCliInspectArguments: false,
  EnableNodeOptionsEnvironmentVariable: false,
  OnlyLoadAppFromAsar: true,
  EnableEmbeddedAsarIntegrityValidation: true,
  GrantFileProtocolExtraPrivileges: false,
};
