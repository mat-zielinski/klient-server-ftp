<script lang="ts">
  // Nagłówek strony jak w stacji (naglowek_strony.py, klasa .ph): karta w jednym wierszu
  // [tytuł / podtytuł] [środek: kafle] [prawo: pigułki, przyciski]. Przy zwężaniu najpierw chowa się środek,
  // potem prawy obszar łamie się pod tytuł (bez przeskakiwania wysokości przy zwykłych szerokościach).
  // Na telefonie po lewej przycisk otwierający szufladę paska bocznego.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../ikony/ikony'
  import { pasek } from '../stan/pasek.svelte'
  import { uklad } from '../stan/uklad.svelte'
  import Ikona from './wspolne/Ikona.svelte'

  interface Props {
    tytul: string
    podtytul?: string
    ikona?: NazwaIkony
    /** Środek wiersza (np. kafle liczb); chowany na wąskim oknie. */
    srodek?: Snippet
    /** Prawy obszar (pigułki stanu, przyciski). */
    akcje?: Snippet
  }

  let { tytul, podtytul, ikona, srodek, akcje }: Props = $props()
</script>

<header class="naglowek-strony">
  {#if uklad.telefon}
    <button type="button" class="menu" onclick={() => pasek.otworzSzuflade()} aria-label="Menu" title="Menu">
      <Ikona nazwa="menu" />
    </button>
  {/if}
  <div class="tytuly">
    <h2 class="tytul skroc" title={tytul}>{tytul}</h2>
    {#if podtytul}
      <div class="podtytul">
        {#if ikona}<Ikona nazwa={ikona} rozmiar="s" />{/if}
        <span class="skroc" title={podtytul}>{podtytul}</span>
      </div>
    {/if}
  </div>
  {#if srodek && !uklad.waski}<div class="srodek">{@render srodek()}</div>{/if}
  {#if akcje}<div class="akcje">{@render akcje()}</div>{/if}
</header>

<style>
  .naglowek-strony {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--odstep-s) var(--odstep-l);
    min-height: var(--wysokosc-naglowek-strony);
    padding: var(--odstep-s) var(--odstep-xxl);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
  }
  .menu {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-przycisk);
    height: var(--wysokosc-przycisk);
    margin-left: calc(-1 * var(--odstep-s));
    border: 0;
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-tekst);
  }
  .menu:hover {
    background: var(--kolor-przycisk);
  }
  .tytuly {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-ramka);
    flex: 1 1 var(--uklad-strona-min-w);
    min-width: var(--uklad-tytul-karty-min-w);
  }
  .tytul {
    font-size: var(--czcionka-tytul-strony);
    font-weight: var(--czcionka-gruby);
  }
  .podtytul {
    display: flex;
    align-items: center;
    gap: var(--odstep-s);
    min-width: 0;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .srodek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    flex: none;
  }
  .akcje {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: var(--odstep-l);
    margin-left: auto;
  }
</style>
