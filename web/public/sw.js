// Service worker Skrytki: pamięć podręczna TYLKO powłoki aplikacji (index.html, pliki /assets/*.js i *.css, manifest,
// ikony), żeby aplikacja otwierała się szybko i dała się zainstalować (PWA).
// NIGDY nie przechwytuje /api/ (dane, pobrania plików, SSE) ani innych adresów: przechodzą prosto do sieci.
// Strona (nawigacja): najpierw sieć, bez sieci zapisany index.html. Pliki /assets/ mają hash w nazwie: najpierw pamięć.

const PAMIEC = 'skrytka-powloka-v1'
const POWLOKA = ['./', './index.html', './manifest.webmanifest', './ikona-192.png', './ikona-512.png']
const ZASOB_POWLOKI = /\/assets\/[^/]+\.(js|css)$/
const STATYCZNE = /\/(manifest\.webmanifest|ikona-[\w-]+\.png|apple-touch-icon\.png)$/

self.addEventListener('install', (e) => {
  e.waitUntil(
    caches
      .open(PAMIEC)
      .then((c) => c.addAll(POWLOKA))
      .then(() => self.skipWaiting()),
  )
})

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches
      .keys()
      .then((klucze) => Promise.all(klucze.filter((k) => k !== PAMIEC).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

function zapisz(zadanie, odp) {
  if (odp.ok && odp.type === 'basic') {
    const kopia = odp.clone()
    caches.open(PAMIEC).then((c) => c.put(zadanie, kopia))
  }
  return odp
}

self.addEventListener('fetch', (e) => {
  const z = e.request
  if (z.method !== 'GET') return
  const adres = new URL(z.url)
  if (adres.origin !== self.location.origin) return
  if (adres.pathname.startsWith('/api/')) return // dane i pliki: zawsze sieć, nigdy pamięć

  if (z.mode === 'navigate') {
    e.respondWith(fetch(z).catch(() => caches.match('./index.html')))
    return
  }
  if (ZASOB_POWLOKI.test(adres.pathname) || STATYCZNE.test(adres.pathname)) {
    e.respondWith(caches.match(z).then((w) => w || fetch(z).then((odp) => zapisz(z, odp))))
  }
})
