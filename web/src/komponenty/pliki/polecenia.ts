// Polecenia strony Pliki w palecie Ctrl+K (stan/skroty.svelte.ts). Zwraca wyrejestrowanie wszystkich.
import { SZUKAJ, zarejestrujPolecenie } from '../../stan/skroty.svelte'

export interface DzialaniaPlikow {
  zapis: () => boolean
  nowyFolder: () => void
  wgraj: () => void
  szukaj: () => void
  widok: (w: 'lista' | 'kafelki') => void
  zaznaczWszystko: () => void
}

const GRUPA = 'Pliki'

export function zarejestrujPoleceniaPlikow(d: DzialaniaPlikow): () => void {
  const wyrejestruj = [
    zarejestrujPolecenie({
      id: 'pliki.nowy-folder', nazwa: 'Nowy folder', ikona: 'folder_plus', skrot: 'Shift+N', grupa: GRUPA,
      akcja: d.nowyFolder, widoczne: d.zapis,
    }),
    zarejestrujPolecenie({
      id: 'pliki.wgraj', nazwa: 'Wgraj pliki', ikona: 'wgraj', skrot: 'Shift+U', grupa: GRUPA, slowa: 'upload dodaj',
      akcja: d.wgraj, widoczne: d.zapis,
    }),
    zarejestrujPolecenie({
      id: SZUKAJ, nazwa: 'Szukaj plików', ikona: 'szukaj', skrot: '/', grupa: GRUPA, akcja: d.szukaj,
    }),
    zarejestrujPolecenie({
      id: 'pliki.widok-lista', nazwa: 'Widok: lista', ikona: 'lista', skrot: 'Shift+L', grupa: GRUPA,
      akcja: () => d.widok('lista'),
    }),
    zarejestrujPolecenie({
      id: 'pliki.widok-kafelki', nazwa: 'Widok: kafelki', ikona: 'kafelki', skrot: 'Shift+K', grupa: GRUPA,
      slowa: 'miniatury galeria', akcja: () => d.widok('kafelki'),
    }),
    zarejestrujPolecenie({
      id: 'pliki.zaznacz-wszystko', nazwa: 'Zaznacz wszystko', ikona: 'ok_kolo', grupa: GRUPA,
      slowa: 'Ctrl+A', akcja: d.zaznaczWszystko,
    }),
  ]
  return () => wyrejestruj.forEach((f) => f())
}
