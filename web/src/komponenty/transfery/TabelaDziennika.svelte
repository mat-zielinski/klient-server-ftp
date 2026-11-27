<script lang="ts">
  // Tabela wpisów dziennika (GET /api/dziennik) z filtrami rodzaju, użytkownika (admin) i szukaniem,
  // pigułkami protokołu i wyniku oraz eksportem CSV (GET /api/dziennik/eksport) w nagłówku karty.
  import { adresApi, api, trescBledu } from '../../api/klient'
  import type { WpisDziennika } from '../../api/typy'
  import Karta from '../wspolne/Karta.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import Tabela, { type KolumnaTabeli } from '../wspolne/Tabela.svelte'
  import Wybor from '../wspolne/Wybor.svelte'
  import { czasTrwania, data, liczbaCala, rozmiar } from '../../stan/format'
  import { DZIENNIK } from './stale'
  import { RODZAJE_WPISU, nazwaPliku, nazwaRodzaju, nazwaWyniku, stanProtokolu, stanWyniku } from './typy'

  interface Props { od: string; doCzasu: string; admin: boolean }
  let { od, doCzasu, admin }: Props = $props()

  let rodzaj = $state('')
  let kto = $state('')
  let szukaj = $state('')
  let wpisy = $state.raw<WpisDziennika[]>([])
  let blad = $state('')
  let wczytywanie = $state(false)

  const kolumny = $derived<KolumnaTabeli[]>([
    { klucz: 'czas', etykieta: 'Czas', szerokosc: 'l' },
    { klucz: 'rodzaj', etykieta: 'Rodzaj', szerokosc: 's', ukryjPonizej: 'telefon' },
    ...(admin ? [{ klucz: 'kto', etykieta: 'Kto', szerokosc: 'm' } as KolumnaTabeli] : []),
    { klucz: 'ip', etykieta: 'IP', szerokosc: 'm', ukryjPonizej: 'waski' },
    { klucz: 'protokol', etykieta: 'Protokół', szerokosc: 's', ukryjPonizej: 'telefon' },
    { klucz: 'plik', etykieta: 'Plik' },
    { klucz: 'bajty', etykieta: 'Rozmiar', szerokosc: 's', wyrownanie: 'prawo' },
    { klucz: 'czas_ms', etykieta: 'Trwało', szerokosc: 's', wyrownanie: 'prawo', ukryjPonizej: 'sredni' },
    { klucz: 'wynik', etykieta: 'Wynik', szerokosc: 's' },
  ])

  const opcjeRodzaju = [{ wartosc: '', etykieta: 'Wszystkie' }, ...RODZAJE_WPISU]

  async function wczytaj(parametry: Record<string, string | number>, czyAktualne: () => boolean) {
    wczytywanie = true
    try {
      const lista = await api.get<WpisDziennika[]>('/dziennik', parametry)
      if (!czyAktualne()) return
      wpisy = Array.isArray(lista) ? lista : []
      blad = ''
    } catch (b) {
      if (czyAktualne()) blad = trescBledu(b)
    } finally {
      if (czyAktualne()) wczytywanie = false
    }
  }

  // Każda zmiana zakresu lub filtra: zapytanie po krótkiej zwłoce (pisanie w polu nie wysyła co znak).
  $effect(() => {
    const parametry: Record<string, string | number> = { od, do: doCzasu, limit: DZIENNIK.limitWpisow }
    if (rodzaj) parametry.rodzaj = rodzaj
    if (admin && kto.trim()) parametry.kto = kto.trim()
    if (szukaj.trim()) parametry.szukaj = szukaj.trim()
    let aktualne = true
    const zegar = setTimeout(() => wczytaj(parametry, () => aktualne), DZIENNIK.zwlokaSzukaniaMs)
    return () => {
      aktualne = false
      clearTimeout(zegar)
    }
  })

  function eksportuj() {
    const a = document.createElement('a')
    a.href = adresApi('/dziennik/eksport', { od, do: doCzasu })
    a.download = ''
    document.body.appendChild(a)
    a.click()
    a.remove()
  }
</script>

<Karta tytul="Wpisy dziennika" ikona="dziennik" bezMarginesu>
  {#snippet akcje()}
    <span class="licznik">{liczbaCala(wpisy.length)}{wpisy.length >= DZIENNIK.limitWpisow ? '+' : ''}</span>
    <Przycisk rozmiar="maly" ikona="pobierz" onclick={eksportuj}>Eksport CSV</Przycisk>
  {/snippet}

  <div class="filtry">
    <div class="filtr"><Wybor etykieta="Rodzaj" opcje={opcjeRodzaju} bind:value={rodzaj} /></div>
    {#if admin}
      <div class="filtr"><Pole etykieta="Użytkownik" podpowiedz="Login" ikona="uzytkownik" bind:value={kto} /></div>
    {/if}
    <div class="filtr szeroki"><Pole etykieta="Szukaj" typ="search" podpowiedz="Plik, IP, opis" ikona="szukaj" bind:value={szukaj} /></div>
    {#if wczytywanie}<span class="opis">Wczytywanie…</span>{/if}
  </div>

  <Tabela {kolumny} wiersze={wpisy}>
    {#snippet wiersz(w: WpisDziennika)}
      <tr>
        <td class="mono opis">{data(w.czas)}</td>
        <td class="skroc">{nazwaRodzaju(w.rodzaj)}</td>
        {#if admin}<td class="skroc" title={w.kto}>{w.kto || 'Gość'}</td>{/if}
        <td class="mono opis">{w.ip}</td>
        <td>{#if w.protokol}<Pigulka stan={stanProtokolu(w.protokol)}>{w.protokol.toUpperCase()}</Pigulka>{/if}</td>
        <td title="{w.udzial}{w.sciezka}{w.opis ? `\n${w.opis}` : ''}">
          <span class="plik">
            <span class="skroc">{w.sciezka ? nazwaPliku(w.sciezka) : w.opis}</span>
            {#if w.udzial}<span class="opis skroc">{w.udzial}</span>{/if}
          </span>
        </td>
        <td class="prawo mono">{w.bajty > 0 ? rozmiar(w.bajty) : ''}</td>
        <td class="prawo mono opis">{w.czas_ms > 0 ? czasTrwania(w.czas_ms / 1000) : ''}</td>
        <td><Pigulka stan={stanWyniku(w.wynik)}>{nazwaWyniku(w.wynik)}</Pigulka></td>
      </tr>
    {/snippet}
    {#snippet pusto()}
      <Pusto ikona={blad ? 'alert' : 'dziennik'} tytul={blad ? 'Nie udało się wczytać dziennika' : 'Brak wpisów'}
        opis={blad || 'W tym zakresie i z tymi filtrami nic nie znaleziono.'} />
    {/snippet}
  </Tabela>
</Karta>

<style>
  .filtry {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: var(--odstep-4);
    padding: var(--odstep-5) var(--odstep-l);
    border-bottom: var(--ramka);
  }
  .filtr {
    min-width: var(--kolumna-l);
  }
  .filtr.szeroki {
    flex: 1;
    max-width: var(--uklad-pole-w);
  }
  .plik {
    display: flex;
    align-items: center;
    gap: var(--odstep-2);
    min-width: 0;
  }
  .plik .opis {
    flex: none;
    max-width: 40%;
  }
</style>
