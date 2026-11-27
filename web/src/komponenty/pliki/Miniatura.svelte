<script lang="ts">
  // Miniatura obrazu z GET /api/miniatura (s = kafelki, m = galeria), wczytywana dopiero w polu widzenia
  // (IntersectionObserver); przy błędzie lub dla innych rodzajów ikona rodzaju.
  import type { Wpis } from '../../api/typy'
  import Ikona from '../wspolne/Ikona.svelte'
  import { adresMiniatury } from './operacje'
  import { BARWY_RODZAJOW, ikonaWpisu } from './rodzaje'

  let { udzial, wpis, rozmiar = 's' }: { udzial: string; wpis: Wpis; rozmiar?: 's' | 'm' } = $props()

  let widoczna = $state(false)
  let blad = $state(false)
  let wczytana = $state(false)
  let el = $state<HTMLDivElement | null>(null)

  const obraz = $derived(!wpis.folder && wpis.rodzaj === 'obraz' && !blad)

  // nowa ścieżka = nowa próba
  $effect(() => {
    void wpis.sciezka
    blad = false
    wczytana = false
  })

  $effect(() => {
    if (!el || !obraz || widoczna) return
    const obserwator = new IntersectionObserver((w) => {
      if (w.some((x) => x.isIntersecting)) {
        widoczna = true
        obserwator.disconnect()
      }
    }, { rootMargin: '50%' })
    obserwator.observe(el)
    return () => obserwator.disconnect()
  })
</script>

<div class="miniatura" bind:this={el} style:color={BARWY_RODZAJOW[wpis.folder ? 'folder' : wpis.rodzaj]}>
  {#if obraz && widoczna}
    <img
      src={adresMiniatury(udzial, wpis.sciezka, rozmiar)}
      alt=""
      decoding="async"
      draggable="false"
      class:wczytana
      onload={() => (wczytana = true)}
      onerror={() => (blad = true)}
    />
  {/if}
  {#if !obraz || !wczytana}
    <span class="ikona"><Ikona nazwa={ikonaWpisu(wpis)} rozmiar="l" /></span>
  {/if}
</div>

<style>
  .miniatura {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
  .ikona {
    position: absolute;
    display: flex;
  }
  .ikona :global(svg) {
    width: var(--uklad-ikona-pusto);
    height: var(--uklad-ikona-pusto);
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0;
    transition: opacity var(--czas-sredni);
  }
  img.wczytana {
    opacity: 1;
  }
</style>
