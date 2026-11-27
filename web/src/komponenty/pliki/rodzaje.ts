// Rodzaje wpisów: ikony, nazwy, które pliki mają podgląd.
import type { Rodzaj, Wpis } from '../../api/typy'
import { IKONA_RODZAJU, type NazwaIkony } from '../../ikony/ikony'

export const NAZWY_RODZAJOW: Record<Rodzaj, string> = {
  folder: 'Folder',
  obraz: 'Obraz',
  wideo: 'Wideo',
  dzwiek: 'Dźwięk',
  pdf: 'Dokument PDF',
  tekst: 'Tekst',
  archiwum: 'Archiwum',
  inny: 'Plik',
}

/** Barwa ikony rodzaju (zmienna CSS z tokenów). */
export const BARWY_RODZAJOW: Record<Rodzaj, string> = {
  folder: 'var(--kolor-zolty)',
  obraz: 'var(--kolor-zielony)',
  wideo: 'var(--kolor-fioletowy)',
  dzwiek: 'var(--kolor-rozowy)',
  pdf: 'var(--kolor-czerwony)',
  tekst: 'var(--kolor-niebieski)',
  archiwum: 'var(--kolor-pomaranczowy)',
  inny: 'var(--kolor-opis)',
}

export function ikonaWpisu(w: Wpis): NazwaIkony {
  return IKONA_RODZAJU[w.folder ? 'folder' : w.rodzaj] ?? 'plik'
}

const Z_PODGLADEM: Rodzaj[] = ['obraz', 'wideo', 'dzwiek', 'pdf', 'tekst']

export function maPodglad(w: Wpis): boolean {
  return !w.folder && Z_PODGLADEM.includes(w.rodzaj)
}
