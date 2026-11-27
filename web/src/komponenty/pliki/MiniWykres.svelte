<script lang="ts">
  // Mini wykres prędkości (linia + wypełnienie) z ostatnich próbek; skala od zera do maksimum historii.
  import { DLUGOSC_HISTORII } from './stale'

  let { probki }: { probki: number[] } = $props()

  const SZER = DLUGOSC_HISTORII
  const WYS = 20

  const punkty = $derived.by(() => {
    const ostatnie = probki.slice(-SZER)
    const maks = Math.max(1, ...ostatnie)
    const przes = SZER - ostatnie.length
    return ostatnie.map((v, i) => [przes + i, WYS - (v / maks) * (WYS - 1)] as const)
  })
  const linia = $derived(punkty.map(([x, y]) => `${x},${y.toFixed(1)}`).join(' '))
  const pole = $derived(
    punkty.length ? `${punkty[0][0]},${WYS} ${linia} ${punkty[punkty.length - 1][0]},${WYS}` : '',
  )
</script>

<svg class="wykres" viewBox="0 0 {SZER} {WYS}" preserveAspectRatio="none" aria-hidden="true">
  {#if punkty.length > 1}
    <polygon class="pole" points={pole} />
    <polyline class="linia" points={linia} vector-effect="non-scaling-stroke" />
  {/if}
</svg>

<style>
  .wykres {
    display: block;
    width: 100%;
    height: var(--wysokosc-segment);
    background: var(--kolor-mono);
    border-radius: var(--promien-kropka);
  }
  .linia {
    fill: none;
    stroke: var(--kolor-niebieski);
    stroke-width: var(--uklad-kreska-ikony);
  }
  .pole {
    fill: var(--kolor-tlo-info);
  }
</style>
