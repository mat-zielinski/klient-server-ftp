<script lang="ts">
  // Wykres liniowy serii czasowej (jak w UniFi): gładka linia, wypełnienie gradientem, siatka, osie, dymek.
  // Z `oknoMs` oś czasu to ostatnie `oknoMs` i przesuwa się płynnie co klatkę (wykres na żywo).
  import { Tween } from 'svelte/motion'
  import Dymek from './Dymek.svelte'
  import { WYKRES, NA_ZYWO } from './stale'
  import { kolorBarwy, liniowa, najblizszy, noweId, obserwujRozmiar, podzialki, sciezkaGladka, type Barwa, type PunktCzasu, type RodzajOsi } from './skala'

  interface Props {
    punkty: PunktCzasu[]
    formatY: (v: number) => string
    formatX?: (t: number) => string
    formatDymka?: (t: number) => string
    oknoMs?: number
    barwa?: Barwa
    os?: RodzajOsi
    nazwa?: string
  }
  let { punkty, formatY, formatX = godzinaMinuta, formatDymka = godzinaSekunda, oknoMs, barwa = 'niebieski', nazwa = '', os = 'zwykla' }: Props = $props()

  function godzinaMinuta(t: number): string {
    return new Date(t).toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit' })
  }
  function godzinaSekunda(t: number): string {
    return new Date(t).toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  }

  const idGradientu = noweId('gradient-linii')
  const idPrzyciecia = noweId('przyciecie-linii')
  let szer = $state(0)
  let wys = $state(0)
  let kursorX = $state<number | null>(null)
  let teraz = $state(Date.now())

  // Zegar klatek tylko na żywo: płynne przesuwanie osi między zdarzeniami co 1 s.
  $effect(() => {
    if (!oknoMs) return
    let klatka = requestAnimationFrame(function krok() {
      teraz = Date.now()
      klatka = requestAnimationFrame(krok)
    })
    return () => cancelAnimationFrame(klatka)
  })

  const m = WYKRES.margines
  const kolor = $derived(kolorBarwy(barwa))
  const dziedzinaX = $derived.by((): [number, number] => {
    if (oknoMs) {
      const koniec = teraz - NA_ZYWO.opoznienieMs
      return [koniec - oknoMs, koniec]
    }
    if (punkty.length === 0) return [teraz - 1, teraz]
    return [punkty[0].t, punkty[punkty.length - 1].t]
  })

  // Punkty widoczne plus po jednym z każdej strony, żeby linia wchodziła i wychodziła poza krawędź.
  const widoczne = $derived.by(() => {
    const [d0, d1] = dziedzinaX
    let pocz = 0
    let kon = punkty.length
    while (pocz < punkty.length - 1 && punkty[pocz + 1].t < d0) pocz++
    while (kon > pocz + 1 && punkty[kon - 2].t > d1) kon--
    return punkty.slice(pocz, kon)
  })

  const maxDanych = $derived(widoczne.reduce((a, p) => Math.max(a, p.v), 0))
  const podzialkiY = $derived(podzialki(maxDanych, os))
  const maxY = new Tween(1, { duration: 300 })
  $effect(() => {
    maxY.target = podzialkiY[podzialkiY.length - 1]
  })

  const skalaX = $derived(liniowa(dziedzinaX[0], dziedzinaX[1], m.lewo, szer - m.prawo))
  const skalaY = $derived(liniowa(0, maxY.current || 1, wys - m.dol, m.gora))
  const xs = $derived(widoczne.map((p) => skalaX(p.t)))
  const ys = $derived(widoczne.map((p) => skalaY(p.v)))
  const linia = $derived(sciezkaGladka(xs, ys))
  const obszar = $derived(
    xs.length > 1 ? `${linia} L${xs[xs.length - 1]},${wys - m.dol} L${xs[0]},${wys - m.dol} Z` : '',
  )

  const podzialkiX = $derived.by(() => {
    const [d0, d1] = dziedzinaX
    if (oknoMs) {
      const krok = NA_ZYWO.podzialkaCzasuMs
      const wynik: number[] = []
      for (let t = Math.ceil(d0 / krok) * krok; t <= d1; t += krok) wynik.push(t)
      return wynik
    }
    const ile = Math.max(2, Math.floor((szer - m.lewo - m.prawo) / WYKRES.minOdstepOpisowCzasu))
    return Array.from({ length: ile }, (_, i) => d0 + ((d1 - d0) * i) / (ile - 1))
  })

  const wskazany = $derived.by(() => {
    if (kursorX === null || xs.length === 0) return null
    const i = najblizszy(xs, kursorX)
    if (xs[i] < m.lewo || xs[i] > szer - m.prawo) return null
    return { x: xs[i], y: ys[i], punkt: widoczne[i] }
  })

  function ruch(e: PointerEvent) {
    const prost = (e.currentTarget as SVGSVGElement).getBoundingClientRect()
    kursorX = e.clientX - prost.left
  }
</script>

<div class="wykres" use:obserwujRozmiar={(s, w) => { szer = s; wys = w }}>
  {#if szer > 0 && wys > 0}
    <svg width={szer} height={wys} viewBox="0 0 {szer} {wys}" onpointermove={ruch} onpointerleave={() => (kursorX = null)} role="img" aria-label={nazwa}>
      <defs>
        <linearGradient id={idGradientu} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" class="gradient-gora" style:stop-color={kolor} />
          <stop offset="1" class="gradient-dol" style:stop-color={kolor} />
        </linearGradient>
        <clipPath id={idPrzyciecia}>
          <rect x={m.lewo} y={0} width={Math.max(0, szer - m.lewo - m.prawo)} height={wys} />
        </clipPath>
      </defs>

      <g class="siatka">
        {#each podzialkiY as v (v)}
          {#if v <= maxY.current * 1.001}
            <line x1={m.lewo} x2={szer - m.prawo} y1={skalaY(v)} y2={skalaY(v)} />
          {/if}
        {/each}
      </g>
      <g class="opisy">
        {#each podzialkiY as v (v)}
          {#if v <= maxY.current * 1.001}
            <text x={m.lewo - WYKRES.odstepOpisu} y={skalaY(v)} text-anchor="end" dominant-baseline="middle">{formatY(v)}</text>
          {/if}
        {/each}
        {#each podzialkiX as t (t)}
          <text x={skalaX(t)} y={wys - m.dol + WYKRES.odstepOpisu} text-anchor="middle" dominant-baseline="hanging">{formatX(t)}</text>
        {/each}
      </g>

      <g clip-path="url(#{idPrzyciecia})">
        {#if obszar}<path d={obszar} fill="url(#{idGradientu})" />{/if}
        <path d={linia} class="linia" style:stroke={kolor} />
      </g>

      {#if wskazany}
        <line class="kursor" x1={wskazany.x} x2={wskazany.x} y1={m.gora} y2={wys - m.dol} />
        <circle class="kropka" cx={wskazany.x} cy={wskazany.y} r={WYKRES.promienKropki} style:fill={kolor} />
      {/if}
    </svg>
    {#if wskazany}
      <Dymek x={wskazany.x / szer} y={wskazany.y / wys}>
        <span class="dymek-tytul">{formatDymka(wskazany.punkt.t)}</span>
        <span class="dymek-wiersz">
          <span class="dymek-znacznik" style:background={kolor}></span>
          {nazwa}
          <span class="dymek-wartosc">{formatY(wskazany.punkt.v)}</span>
        </span>
      </Dymek>
    {/if}
  {/if}
</div>

<style>
  .wykres {
    position: relative;
    width: 100%;
    height: var(--wykres-wysokosc, calc(var(--wysokosc-przycisk) * 8));
    min-width: 0;
  }
  svg {
    display: block;
    overflow: visible;
    touch-action: pan-y;
  }
  .siatka line {
    stroke: var(--kolor-siatka);
    stroke-width: 1;
    shape-rendering: crispEdges;
  }
  .opisy text {
    fill: var(--kolor-opis);
    font-size: var(--czcionka-drobny);
    font-family: var(--czcionka-mono);
  }
  .linia {
    fill: none;
    stroke-width: 2;
    stroke-linejoin: round;
    stroke-linecap: round;
  }
  .gradient-gora {
    stop-opacity: 0.35;
  }
  .gradient-dol {
    stop-opacity: 0;
  }
  .kursor {
    stroke: var(--kolor-opis);
    stroke-width: 1;
    stroke-dasharray: 3 3;
    shape-rendering: crispEdges;
  }
  .kropka {
    stroke: var(--kolor-panel);
    stroke-width: 2;
  }
</style>
