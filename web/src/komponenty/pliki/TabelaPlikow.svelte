<script lang="ts">
  // Widok listy (wirtualizowany): ikona rodzaju, nazwa, rozmiar, zmieniono, akcje. Renderuje tylko widoczne
  // wiersze + bufor; odstępy u góry i u dołu trzymają wysokość przewijania. Zdarzenia obsługuje strona.
  import type { Wpis } from '../../api/typy'
  import Tabela, { type KolumnaTabeli } from '../wspolne/Tabela.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import { data, dataWzgledna, rozmiar } from '../../stan/format'
  import { BARWY_RODZAJOW, ikonaWpisu } from './rodzaje'
  import type { ZdarzeniaWpisow } from './zdarzenia'
  import Podswietlenie from './Podswietlenie.svelte'
  import { Wirtualizacja, dlugosc } from './wirtualizacja.svelte'

  let {
    wpisy,
    zaznaczone,
    kursor,
    sort,
    malejaco,
    wieleNaraz,
    zapytanie = '',
    z,
  }: {
    wpisy: Wpis[]
    zaznaczone: Set<string>
    kursor: number
    sort: string
    malejaco: boolean
    wieleNaraz: boolean
    /** Tryb wyników szukania: kolumna ścieżki i podświetlenie trafień. */
    zapytanie?: string
    z: ZdarzeniaWpisow
  } = $props()

  type Wiersz = { odstep: number } | { w: Wpis; i: number }

  const wirt = new Wirtualizacja(() => wpisy.length)
  let kontener = $state<HTMLDivElement | null>(null)

  $effect(() => (kontener ? wirt.podlacz(kontener) : undefined))
  $effect(() => {
    void wpisy.length
    wirt.zaplanuj()
  })

  /** Przewija do wpisu (strzałki na klawiaturze). */
  export function przewinDo(indeks: number) {
    wirt.przewinDo(indeks)
  }

  const wiersze = $derived.by((): Wiersz[] => {
    const lista: Wiersz[] = wpisy.slice(wirt.start, wirt.koniec).map((w, j) => ({ w, i: wirt.start + j }))
    return [{ odstep: wirt.gora }, ...lista, { odstep: wirt.dol }]
  })

  const kolumny = $derived<KolumnaTabeli[]>([
    ...(wieleNaraz ? [{ klucz: 'wybor', etykieta: '', szerokosc: 'xs', sortowalna: false }] : []),
    { klucz: 'ikona', etykieta: '', szerokosc: 'xs', sortowalna: false },
    { klucz: 'nazwa', etykieta: 'Nazwa' },
    { klucz: 'rozmiar', etykieta: 'Rozmiar', szerokosc: 's', wyrownanie: 'prawo', ukryjPonizej: 'telefon' },
    { klucz: 'zmieniono', etykieta: 'Zmieniono', szerokosc: 'l', ukryjPonizej: 'waski' },
    { klucz: 'akcje', etykieta: '', szerokosc: 's', wyrownanie: 'prawo', sortowalna: false },
  ])
</script>

<div class="kontener" bind:this={kontener}>
  <Tabela {kolumny} {wiersze} sortKlucz={sort} {malejaco} naSortuj={zapytanie ? undefined : (k, m) => z.sortuj(k, m)}>
    {#snippet wiersz(r)}
      {#if 'odstep' in r}
        {#if r.odstep > 0}
          <tr class="odstep" aria-hidden="true" style:height={dlugosc(r.odstep)}><td colspan={kolumny.length}></td></tr>
        {/if}
      {:else}
        {@const w = r.w}
        <tr
          class="wpis"
          class:zaznaczony={zaznaczone.has(w.sciezka)}
          class:kursor={r.i === kursor}
          data-indeks={r.i}
          aria-selected={zaznaczone.has(w.sciezka)}
          onclick={(e) => z.klik(w, r.i, e)}
          ondblclick={() => z.otworz(w)}
          oncontextmenu={(e) => z.menu(w, r.i, e)}
        >
          {#if wieleNaraz}
            <td class="srodek">
              <input
                type="checkbox"
                checked={zaznaczone.has(w.sciezka)}
                aria-label="Zaznacz {w.nazwa}"
                onclick={(e) => {
                  e.stopPropagation()
                  z.przelacz(w, r.i)
                }}
              />
            </td>
          {/if}
          <td class="srodek ikona" style:color={BARWY_RODZAJOW[w.folder ? 'folder' : w.rodzaj]}>
            <Ikona nazwa={ikonaWpisu(w)} rozmiar="s" />
          </td>
          <td class="nazwa" title={w.sciezka}>
            <span class="skroc"><Podswietlenie tekst={w.nazwa} {zapytanie} /></span>
            {#if zapytanie}<span class="opis skroc sciezka">{w.sciezka}</span>{/if}
          </td>
          <td class="prawo mono opis">{w.folder ? '' : rozmiar(w.rozmiar)}</td>
          <td class="opis" title={data(w.zmieniono)}>{dataWzgledna(w.zmieniono)}</td>
          <td class="prawo">
            <button
              class="wiecej"
              aria-label="Akcje: {w.nazwa}"
              title="Akcje"
              onclick={(e) => {
                e.stopPropagation()
                z.menu(w, r.i, e)
              }}
            >
              <Ikona nazwa="wiecej" rozmiar="s" />
            </button>
          </td>
        </tr>
      {/if}
    {/snippet}
  </Tabela>
</div>

<style>
  .kontener {
    min-width: 0;
  }
  .odstep,
  .odstep:hover {
    background: transparent !important;
    border: 0 !important;
  }
  .odstep td {
    padding: 0;
  }
  .wpis {
    cursor: default;
    user-select: none;
  }
  .wpis.kursor {
    box-shadow: inset var(--odstep-pozycje) 0 0 var(--kolor-niebieski);
  }
  .nazwa {
    font-size: var(--czcionka-tekst);
  }
  .nazwa .skroc {
    display: block;
  }
  .sciezka {
    font-size: var(--czcionka-drobny);
  }
  .ikona {
    line-height: 0;
  }
  .wiecej {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-segment);
    height: var(--wysokosc-segment);
    padding: 0;
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    opacity: 0;
  }
  .wpis:hover .wiecej,
  .wpis.zaznaczony .wiecej,
  .wiecej:focus-visible {
    opacity: 1;
  }
  .wiecej:hover {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  @media (hover: none) {
    .wiecej {
      opacity: 1;
    }
  }
</style>
