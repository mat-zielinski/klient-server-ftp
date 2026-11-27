<script lang="ts">
  // Mini wykres (sparkline) do wiersza transferu: ostatnie próbki prędkości, linia z wypełnieniem.
  // Rozciąga się na cały kontener (viewBox + preserveAspectRatio="none", grubość linii niezależna od skali).
  import { WYKRES } from './stale'
  import { kolorBarwy, noweId, sciezkaGladka, type Barwa } from './skala'

  interface Props { wartosci: number[]; barwa?: Barwa; opis?: string }
  let { wartosci, barwa = 'niebieski', opis = '' }: Props = $props()

  const W = 100
  const H = 100
  const idGradientu = noweId('gradient-mini')
  const kolor = $derived(kolorBarwy(barwa))

  const sciezki = $derived.by(() => {
    const n = wartosci.length
    if (n < 2) return { linia: '', obszar: '' }
    const max = Math.max(...wartosci, 0) || 1
    const m = WYKRES.marginesMini
    const xs = wartosci.map((_, i) => (i / (n - 1)) * W)
    const ys = wartosci.map((v) => H - m - (Math.max(0, v) / max) * (H - 2 * m))
    const linia = sciezkaGladka(xs, ys)
    return { linia, obszar: `${linia} L${W},${H} L0,${H} Z` }
  })
</script>

<svg class="mini" viewBox="0 0 {W} {H}" preserveAspectRatio="none" role="img" aria-label={opis}>
  {#if opis}<title>{opis}</title>{/if}
  <defs>
    <linearGradient id={idGradientu} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" class="gora" style:stop-color={kolor} />
      <stop offset="1" class="dol" style:stop-color={kolor} />
    </linearGradient>
  </defs>
  {#if sciezki.linia}
    <path d={sciezki.obszar} fill="url(#{idGradientu})" />
    <path d={sciezki.linia} class="linia" style:stroke={kolor} />
  {:else}
    <line x1="0" x2={W} y1={H - WYKRES.marginesMini} y2={H - WYKRES.marginesMini} class="pusta" />
  {/if}
</svg>

<style>
  .mini {
    display: block;
    width: 100%;
    height: var(--mini-wykres-wysokosc, var(--wysokosc-znaczek));
    min-width: var(--mini-wykres-szerokosc, var(--kolumna-s));
  }
  .linia {
    fill: none;
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
  }
  .pusta {
    stroke: var(--kolor-siatka);
    stroke-width: 1;
    vector-effect: non-scaling-stroke;
  }
  .gora {
    stop-opacity: 0.3;
  }
  .dol {
    stop-opacity: 0;
  }
</style>
