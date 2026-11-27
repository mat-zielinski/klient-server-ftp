<script lang="ts">
  // Widok kafelków (wirtualizowany po wierszach siatki): miniatura obrazu z /api/miniatura (rozmiar s, leniwie),
  // ikona rodzaju dla reszty. Zdarzenia obsługuje strona.
  import type { Wpis } from '../../api/typy'
  import { rozmiar } from '../../stan/format'
  import Miniatura from './Miniatura.svelte'
  import Podswietlenie from './Podswietlenie.svelte'
  import type { ZdarzeniaWpisow } from './zdarzenia'
  import { Wirtualizacja, dlugosc } from './wirtualizacja.svelte'

  let {
    wpisy,
    udzial,
    zaznaczone,
    kursor,
    wieleNaraz,
    zapytanie = '',
    z,
  }: {
    wpisy: Wpis[]
    udzial: string
    zaznaczone: Set<string>
    kursor: number
    wieleNaraz: boolean
    zapytanie?: string
    z: ZdarzeniaWpisow
  } = $props()

  let siatka = $state<HTMLUListElement | null>(null)
  let kolumn = $state(1)

  const wirt = new Wirtualizacja(() => wpisy.length, () => kolumn)

  function policzKolumny() {
    if (!siatka) return
    const n = getComputedStyle(siatka).gridTemplateColumns.split(' ').filter(Boolean).length
    if (n > 0 && n !== kolumn) kolumn = n
  }

  $effect(() => {
    if (!siatka) return
    const el = siatka
    const obserwator = new ResizeObserver(() => {
      policzKolumny()
      wirt.zaplanuj()
    })
    obserwator.observe(el)
    const odlacz = wirt.podlacz(el)
    return () => {
      obserwator.disconnect()
      odlacz()
    }
  })
  $effect(() => {
    void wpisy.length
    wirt.zaplanuj()
  })

  export function przewinDo(indeks: number) {
    wirt.przewinDo(indeks)
  }

  /** Kafelków w wierszu (strzałki góra / dół). */
  export function wWierszu(): number {
    return kolumn
  }

  const widoczne = $derived(wpisy.slice(wirt.start, wirt.koniec))
</script>

<ul class="kafelki" bind:this={siatka} style:padding-top={dlugosc(wirt.gora)} style:padding-bottom={dlugosc(wirt.dol)}>
  {#each widoczne as w, j (w.sciezka)}
    {@const i = wirt.start + j}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
    <li
      class="kafel"
      class:zaznaczony={zaznaczone.has(w.sciezka)}
      class:kursor={i === kursor}
      data-indeks={i}
      title={w.nazwa}
      onclick={(e) => z.klik(w, i, e)}
      ondblclick={() => z.otworz(w)}
      oncontextmenu={(e) => z.menu(w, i, e)}
    >
      {#if wieleNaraz}
        <input
          class="wybor"
          type="checkbox"
          checked={zaznaczone.has(w.sciezka)}
          aria-label="Zaznacz {w.nazwa}"
          onclick={(e) => {
            e.stopPropagation()
            z.przelacz(w, i)
          }}
        />
      {/if}
      <div class="ramka-miniatury">
        <Miniatura {udzial} wpis={w} rozmiar="s" />
      </div>
      <span class="nazwa"><Podswietlenie tekst={w.nazwa} {zapytanie} /></span>
      <span class="opis">{w.folder ? 'Folder' : rozmiar(w.rozmiar)}</span>
    </li>
  {/each}
</ul>

<style>
  .kafelki {
    list-style: none;
    margin: 0;
    padding-left: var(--odstep-l);
    padding-right: var(--odstep-l);
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(var(--uklad-kafel-min-w), 1fr));
    gap: var(--odstep-l);
    border-top: var(--odstep-l) solid transparent;
    border-bottom: var(--odstep-l) solid transparent;
  }
  .kafel {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    padding: var(--odstep-m);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-kafel);
    cursor: default;
    user-select: none;
    min-width: 0;
    transition: background var(--czas-szybki), border-color var(--czas-szybki);
  }
  .kafel:hover {
    background: var(--kolor-panel);
  }
  .kafel.zaznaczony {
    background: var(--kolor-aktywny);
    border-color: var(--kolor-niebieski);
  }
  .kafel.kursor {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
  }
  .ramka-miniatury {
    aspect-ratio: 4 / 3;
    background: var(--kolor-mono);
    border-radius: var(--promien-segment);
    overflow: hidden;
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    word-break: break-word;
    min-height: calc(var(--czcionka-tekst) * 2.8);
  }
  .wybor {
    position: absolute;
    top: var(--odstep-m);
    left: var(--odstep-m);
    z-index: 1;
  }
</style>
