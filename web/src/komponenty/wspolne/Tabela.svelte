<script lang="ts" module>
  import type { Prog } from '../../stan/uklad.svelte'

  export interface KolumnaTabeli {
    klucz: string
    etykieta: string
    /** 'xs' | 's' | 'm' | 'l' | 'xl' (tokeny --kolumna-*) albo wartość CSS bez px (np. '30%', 'auto'). */
    szerokosc?: string
    wyrownanie?: 'lewo' | 'srodek' | 'prawo'
    /** Kolumna chowa się, gdy okno węższe niż próg z tokenów (--prog-telefon / --prog-waski / --prog-sredni). */
    ukryjPonizej?: Prog
    /** Domyślnie true, gdy tabela ma naSortuj. */
    sortowalna?: boolean
  }
</script>

<script lang="ts" generics="T">
  // Zwarta tabela (jak listy stacji): jasny nagłówek WIELKIMI literami, wiersze z kreską, podświetlenie pod myszą.
  // Snippet `wiersz` renderuje CAŁY wiersz <tr> z komórkami <td> w kolejności kolumn (zdarzenia, zaznaczenie,
  // menu kontekstowe zostają po stronie strony). Ukrywanie kolumn działa po numerze kolumny.
  // Klasy pomocnicze dla komórek: .prawo, .srodek, .mono, .opis, .skroc; wiersz: .zaznaczony.
  import type { Snippet } from 'svelte'
  import { uklad } from '../../stan/uklad.svelte'
  import Ikona from './Ikona.svelte'

  interface Props {
    kolumny: KolumnaTabeli[]
    wiersze: T[]
    wiersz: Snippet<[T, number]>
    naSortuj?: (klucz: string, malejaco: boolean) => void
    sortKlucz?: string
    malejaco?: boolean
    /** Treść, gdy brak wierszy (np. <Pusto>). */
    pusto?: Snippet
    /** Nagłówek przyklejony przy przewijaniu. */
    przyklejony?: boolean
  }

  let {
    kolumny,
    wiersze,
    wiersz,
    naSortuj,
    sortKlucz = $bindable(),
    malejaco = $bindable(false),
    pusto,
    przyklejony = true,
  }: Props = $props()

  const SZEROKOSCI = ['xs', 's', 'm', 'l', 'xl']
  const id = $props.id()

  const ukryte = $derived(
    kolumny.map((k, i) => (k.ukryjPonizej && uklad.ponizej(k.ukryjPonizej) ? i + 1 : 0)).filter((n) => n > 0),
  )
  const stylUkrycia = $derived(
    ukryte.length
      ? `<style>${ukryte.map((n) => `[data-tabela="${id}"] tr > :nth-child(${n})`).join(',')}{display:none}</style>`
      : '',
  )

  function szerokosc(k: KolumnaTabeli): string | undefined {
    if (!k.szerokosc) return undefined
    return SZEROKOSCI.includes(k.szerokosc) ? `var(--kolumna-${k.szerokosc})` : k.szerokosc
  }

  function sortuj(k: KolumnaTabeli) {
    if (!naSortuj || k.sortowalna === false) return
    malejaco = sortKlucz === k.klucz ? !malejaco : false
    sortKlucz = k.klucz
    naSortuj(k.klucz, malejaco)
  }
</script>

{@html stylUkrycia}
<div class="tabela" data-tabela={id}>
  <table>
    <colgroup>
      {#each kolumny as k (k.klucz)}<col style:width={szerokosc(k)} />{/each}
    </colgroup>
    <thead class:przyklejony>
      <tr>
        {#each kolumny as k (k.klucz)}
          {@const sortowalna = !!naSortuj && k.sortowalna !== false}
          <th
            class={k.wyrownanie ?? 'lewo'}
            aria-sort={sortKlucz === k.klucz ? (malejaco ? 'descending' : 'ascending') : undefined}
          >
            {#if sortowalna}
              <button type="button" class="sort" class:aktywny={sortKlucz === k.klucz} onclick={() => sortuj(k)}>
                <span class="skroc">{k.etykieta}</span>
                {#if sortKlucz === k.klucz}
                  <Ikona nazwa={malejaco ? 'w_dol' : 'w_gore'} rozmiar="s" />
                {/if}
              </button>
            {:else}
              <span class="skroc">{k.etykieta}</span>
            {/if}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each wiersze as w, i (i)}
        {@render wiersz(w, i)}
      {/each}
    </tbody>
  </table>
  {#if !wiersze.length && pusto}<div class="pusto">{@render pusto()}</div>{/if}
</div>

<style>
  .tabela {
    width: 100%;
    min-width: 0;
    overflow: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
    font-size: var(--czcionka-maly);
  }
  thead th {
    height: var(--wysokosc-naglowek-karty);
    padding: 0 var(--odstep-l);
    border-bottom: var(--ramka);
    background: var(--kolor-panel);
    color: var(--kolor-opis);
    font-size: var(--czcionka-etykieta);
    font-weight: var(--czcionka-gruby);
    letter-spacing: var(--czcionka-rozstaw-etykiety);
    text-transform: uppercase;
    white-space: nowrap;
    overflow: hidden;
  }
  .przyklejony th {
    position: sticky;
    top: 0;
    z-index: 1;
  }
  th.lewo {
    text-align: left;
  }
  th.srodek {
    text-align: center;
  }
  th.prawo {
    text-align: right;
  }
  .sort {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
    max-width: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    letter-spacing: inherit;
    text-transform: inherit;
  }
  .sort:hover,
  .sort.aktywny {
    color: var(--kolor-tekst);
  }
  .tabela :global(tbody tr) {
    height: var(--wysokosc-wiersz-tabeli);
    border-bottom: var(--ramka);
    transition: background var(--czas-szybki);
  }
  .tabela :global(tbody tr:last-child) {
    border-bottom: 0;
  }
  .tabela :global(tbody tr:hover) {
    background: var(--kolor-karta);
  }
  .tabela :global(tbody tr.zaznaczony) {
    background: var(--kolor-aktywny);
  }
  .tabela :global(td) {
    padding: 0 var(--odstep-l);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .tabela :global(td.prawo) {
    text-align: right;
  }
  .tabela :global(td.srodek) {
    text-align: center;
  }
  .tabela :global(td.mono) {
    font-family: var(--czcionka-mono);
  }
  .tabela :global(td.opis) {
    color: var(--kolor-opis);
  }
  .pusto {
    padding: var(--odstep-sekcja);
  }
</style>
