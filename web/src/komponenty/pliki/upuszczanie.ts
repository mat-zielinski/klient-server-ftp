// Zbieranie plików z przeciągnij i upuść (także całych folderów) i z pola wyboru plików.

export interface PlikDoWgrania {
  plik: File
  /** Ścieżka względem folderu docelowego, np. „zdjecia/a.jpg”. */
  wzgledna: string
}

type Wejscie = FileSystemEntry

function czytajWszystkie(czytnik: FileSystemDirectoryReader): Promise<Wejscie[]> {
  // readEntries zwraca porcje (Chrome po 100), czytamy do pustej
  return new Promise((ok, zle) => {
    const wynik: Wejscie[] = []
    const krok = () =>
      czytnik.readEntries((porcja) => {
        if (!porcja.length) return ok(wynik)
        wynik.push(...porcja)
        krok()
      }, zle)
    krok()
  })
}

async function zbierz(w: Wejscie, przedrostek: string, cel: PlikDoWgrania[]): Promise<void> {
  if (w.isFile) {
    const plik = await new Promise<File>((ok, zle) => (w as FileSystemFileEntry).file(ok, zle))
    cel.push({ plik, wzgledna: przedrostek + w.name })
  } else if (w.isDirectory) {
    const dzieci = await czytajWszystkie((w as FileSystemDirectoryEntry).createReader())
    for (const d of dzieci) await zbierz(d, przedrostek + w.name + '/', cel)
  }
}

/** Pliki z upuszczenia; foldery rozwinięte rekurencyjnie (gdy przeglądarka to wspiera). */
export async function zUpuszczenia(dt: DataTransfer): Promise<PlikDoWgrania[]> {
  const wejscia = [...dt.items]
    .filter((i) => i.kind === 'file')
    .map((i) => i.webkitGetAsEntry?.() ?? null)
  if (wejscia.length && wejscia.every((w) => w !== null)) {
    const wynik: PlikDoWgrania[] = []
    for (const w of wejscia) await zbierz(w as Wejscie, '', wynik)
    return wynik
  }
  return zListy(dt.files)
}

/** Pliki z <input type="file">. */
export function zListy(pliki: FileList | null): PlikDoWgrania[] {
  return [...(pliki ?? [])].map((plik) => ({ plik, wzgledna: plik.webkitRelativePath || plik.name }))
}

/** Czy przeciągane są pliki z dysku (a nie np. tekst). */
export function toPliki(e: DragEvent): boolean {
  return !!e.dataTransfer && [...e.dataTransfer.types].includes('Files')
}
