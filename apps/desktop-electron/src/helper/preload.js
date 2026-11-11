'use strict';
// The only preload in the app (helper view only). Four invoke channels and one push channel; the main process checks
// the sender of every call. The game view has no preload.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('d2idle', {
  getState: () => ipcRenderer.invoke('d2idle:state'),
  saveAddress: (address) => ipcRenderer.invoke('d2idle:saveAddress', String(address)),
  retry: () => ipcRenderer.invoke('d2idle:retry'),
  openPage: (name) => ipcRenderer.invoke('d2idle:openPage', String(name)),
  onState: (cb) => {
    const l = (_e, s) => cb(s);
    ipcRenderer.on('d2idle:state', l);
    return () => ipcRenderer.removeListener('d2idle:state', l);
  },
});
