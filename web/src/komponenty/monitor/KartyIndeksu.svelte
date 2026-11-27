<script lang="ts">
  // Analityka indeksu (GET /api/indeks/statystyki): zajętość wg rodzaju plików (pierścień, wybór udziału),
  // ranking udziałów wg bajtów i stan indeksu każdego udziału. Dwie karty obok siebie w siatce rodzica.
  // Ładowanie = szkielety, brak adresu na serwerze = stan pusty z opisem.
  import { onMount } from 'svelte'
  import { BladApi, trescBledu } from '../../api/klient'
  import { pobierzStatystykiIndeksu, type StatystykiIndeksu, type StatystykiUdzialu } from '../../api/monitor'
  import Karta from '../wspolne/Karta.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import Wybor from '../wspolne/Wybor.svelte'
  import ListaRankingu from '../wykresy/ListaRankingu.svelte'
  import Pierscien from '../wykresy/Pierscien.svelte'
  import type { Pozycja } from '../wykresy/skala'
  import { dataWzgledna, liczbaCala, rozmiar } from '../../stan/format'
  import Szkielet from './Szkielet.svelte'
  import { MONITOR, NAZWY_RODZAJOW } from './stale'

  const WSZYSTKIE = ''

  let dane = $state.raw<StatystykiIndeksu | null>(null)
  let ladowanie = $state(true)
  let blad = $state('')
  let brakEndpointu = $state(false)
  let wybrany = $state(WSZYSTKIE)

  async function wczytaj() {
    try {
      dane = await pobierzStatystykiIndeksu()
      blad = ''
      brakEndpointu = false
    } catch (b) {
      brakEndpointu = b instanceof BladApi && b.kod === 404
      blad = trescBledu(b)
    } finally {
      ladowanie = false
    }
  }

  onMount(() => {
    void wczytaj()
    const zegar = setInterval(wczytaj, MONITOR.odswiezanieIndeksuMs)
    return () => clearInterval(zegar)
  })

  const udzialy = $derived<StatystykiUdzialu[]>(dane?.udzialy ?? [])
  const opcje = $derived([{ wartosc: WSZYSTKIE, etykieta: 'Wszystkie udziały' }, ...udzialy.map((u) => ({ wartosc: u.nazwa, etykieta: u.nazwa }))])
  const rodzaje = $derived.by((): Pozycja[] => {
    const suma = new Map<string, { pliki: number; bajty: number }>()
    for (const u of udzialy) {
      if (wybrany !== WSZYSTKIE && u.nazwa !== wybrany) continue
      for (const [r, l] of Object.entries(u.wg_rodzaju ?? {})) {
        const s = suma.get(r) ?? { pliki: 0, bajty: 0 }
        suma.set(r, { pliki: s.pliki + l.pliki, bajty: s.bajty + l.bajty })
      }
    }
    return [...suma.entries()]
      .filter(([, l]) => l.bajty > 0)
      .sort((a, b) => b[1].bajty - a[1].bajty)
      .map(([r, l]) => ({ nazwa: NAZWY_RODZAJOW[r] ?? r, wartosc: l.bajty, opis: `${liczbaCala(l.pliki)} plików` }))
  })
  const ranking = $derived<Pozycja[]>(
    [...udzialy]
      .sort((a, b) => b.bajty - a.bajty)
      .slice(0, MONITOR.pozycjiRankingu)
      .map((u) => ({ nazwa: u.nazwa, wartosc: u.bajty, opis: `${liczbaCala(u.pliki)} plików, ${liczbaCala(u.foldery)} folderów` })),
  )
  const sumaPlikow = $derived(udzialy.reduce((a, u) => a + u.pliki, 0))
</script>

{#snippet stanPusty()}
  {#if brakEndpointu}
    <Pusto ikona="baza" tytul="Indeks plików jeszcze niedostępny"
      opis="Serwer nie udostępnia jeszcze adresu /api/indeks/statystyki. Statystyki pojawią się po aktualizacji serwera." />
  {:else}
    <Pusto ikona="blad" tytul="Nie udało się wczytać statystyk indeksu" opis={blad} />
  {/if}
{/snippet}

<Karta tytul="Zajętość wg rodzaju plików" ikona="dysk">
  {#snippet akcje()}
    {#if udzialy.length > 1}
      <Wybor etykieta="Udział" ukryjEtykiete opcje={opcje} bind:value={wybrany} />
    {/if}
  {/snippet}
  {#if ladowanie && !dane}
    <Szkielet rodzaj="wskaznik" />
  {:else if blad && !dane}
    {@render stanPusty()}
  {:else}
    <div class="rodzaje">
      <Pierscien pozycje={rodzaje} format={rozmiar} podpisSumy="Zajęte" />
      <div class="ranking">
        <h3>Udziały wg rozmiaru</h3>
        <ListaRankingu pozycje={ranking} format={rozmiar} barwa="zielony" pusto="Brak udziałów w indeksie." />
      </div>
    </div>
  {/if}
</Karta>

<Karta tytul="Stan indeksu" ikona="szukaj">
  {#snippet akcje()}
    {#if dane}<span class="suma">{liczbaCala(sumaPlikow)} plików</span>{/if}
  {/snippet}
  {#if ladowanie && !dane}
    <Szkielet rodzaj="linia" ile={3} />
  {:else if blad && !dane}
    {@render stanPusty()}
  {:else if udzialy.length === 0}
    <Pusto ikona="szukaj" tytul="Indeks jest pusty" opis="Brak udziałów do zindeksowania." />
  {:else}
    <ul class="indeks">
      {#each udzialy as u (u.nazwa)}
        <li>
          <span class="nazwa" title={u.nazwa}>{u.nazwa}</span>
          {#if u.zindeksowano}
            <Pigulka stan="ok" kropka>Zindeksowano</Pigulka>
          {:else}
            <Pigulka stan="ostrz" kropka>W toku</Pigulka>
          {/if}
          <span class="liczby">{liczbaCala(u.pliki)} plików, {liczbaCala(u.foldery)} folderów, {rozmiar(u.bajty)}</span>
          <span class="kiedy">{u.zindeksowano ? dataWzgledna(u.zindeksowano) : 'Pierwsze indeksowanie trwa'}</span>
        </li>
      {/each}
    </ul>
  {/if}
</Karta>

<style>
  .rodzaje {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, var(--kolumna-xl)), 1fr));
    gap: var(--odstep-7);
    align-items: start;
  }
  .ranking {
    min-width: 0;
  }
  h3 {
    margin: 0 0 var(--odstep-3);
    font-size: var(--czcionka-etykieta);
    font-weight: var(--czcionka-polgruby);
    letter-spacing: var(--czcionka-rozstaw-etykiety);
    text-transform: uppercase;
    color: var(--kolor-opis);
  }
  .suma {
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
  }
  .indeks {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
  }
  li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: var(--odstep-1) var(--odstep-3);
    align-items: center;
    padding: var(--odstep-3) 0;
    border-bottom: var(--ramka);
  }
  li:last-child {
    border-bottom: 0;
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .liczby,
  .kiedy {
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
  }
  .liczby {
    font-family: var(--czcionka-mono);
  }
  .kiedy {
    text-align: right;
  }
</style>
