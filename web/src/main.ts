// Wejście aplikacji: style (tokeny przed globalnymi), progi układu z tokenów, stan sesji, PWA, montaż App.
import './styl/tokeny.css'
import './styl/globalne.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { uklad } from './stan/uklad.svelte'
import { sesja } from './stan/sesja.svelte'

const SW = '/sw.js'

/** Odnośniki PWA w <head> (manifest, ikony), gdy index.html ich nie ma. */
function dodajOdnosnik(rel: string, href: string, typ?: string): void {
  if (document.head.querySelector(`link[rel="${rel}"]`)) return
  const l = document.createElement('link')
  l.rel = rel
  l.href = href
  if (typ) l.type = typ
  document.head.append(l)
}

dodajOdnosnik('manifest', '/manifest.webmanifest')
dodajOdnosnik('icon', '/ikona-32.png', 'image/png')
dodajOdnosnik('apple-touch-icon', '/apple-touch-icon.png')

// service worker tylko w wersji zbudowanej i w bezpiecznym kontekście (HTTPS albo localhost)
if (import.meta.env.PROD && 'serviceWorker' in navigator && window.isSecureContext) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register(SW).catch(() => {
      // celowo: brak service workera = aplikacja działa bez instalacji i pamięci podręcznej
    })
  })
}

uklad.odczytajProgi()
void sesja.odswiez()

const cel = document.getElementById('aplikacja')
if (!cel) throw new Error('Brak elementu #aplikacja w index.html.')

export default mount(App, { target: cel })
