// Minimal service worker. It exists so browsers offer to install ytclipper and
// register its share target (manifest.webmanifest). It caches nothing: the app
// needs the server anyway, so there is no offline mode and no fetch handler.
self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));
