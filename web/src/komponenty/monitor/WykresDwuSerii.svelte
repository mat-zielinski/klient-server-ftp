<script lang="ts">
  // Wykres liniowy dwóch serii na wspólnej osi (np. wysyłanie i odbiór, odczyt i zapis dysku) w stylu
  // WykresLiniowy: gładkie linie, wypełnienie gradientem, siatka, dymek z obiema wartościami, legenda z bieżącą
  // wartością. Matematyka i dymek ze wspólnych modułów wykresów (skala.ts, Dymek). Z `oknoMs` oś czasu przesuwa się
  // płynnie co klatkę; podziałki czasu dobierane do szerokości.
  import { Tween } from 'svelte/motion'
  import Dymek from '../wykresy/Dymek.svelte'
  import { WYKRES, NA_ZYWO } from '../wykresy/stale'
  import { kolorBarwy, liniowa, najblizszy, noweId, obserwujRozmiar, podzialki, sciezkaGladka, type Barwa, type RodzajOsi } from '../wykresy/skala'
  import type { PunktDwuSerii } from './stan.svelte'
  import { MS_MINUTA } from './stale'

  interface Props {
    punkty: PunktDwuSerii[]
    nazwaA: string
    nazwaB: string
    barwaA?: Barwa
    barwaB?: Barwa
    formatY: (v: number) => string
    oknoMs?: number
    os?: RodzajOsi
    /** Niższy wykres (np. ruch jednego interfejsu). */
    niski?: boolean
  }
  let { punkty, nazwaA, nazwaB, barwaA = 'niebieski', barwaB = 'zielony', formatY, oknoMs, os = 'bajty', niski = false }: Props = $props()

  /** Kroki podziałki czasu: od 1 min do 6 h, wybierany najmniejszy, który mieści opisy. */
  const KROKI_CZASU = [1, 2, 5, 10, 15, 30, 60, 120, 180, 360].map((m) => m * MS_MINUTA)
  const godzina = (t: number, sek = false) =>
    new Date(t).toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit', second: sek ? '2-digit' : undefined })

  const idA = noweId('gradient-a')
  const idB = noweId('gradient-b')
  const idPrzyciecia = noweId('przyciecie-dwoch')
  let szer = $state(0)
  let wys = $state(0)
  let kursorX = $state<number | null>(null)
  let teraz = $state(Date.now())

  $effect(() => {
    if (!oknoMs) return
    let klatka = requestAnimationFrame(function krok() {
      teraz = Date.now()
      klatka = requestAnimationFrame(krok)
    })
    return () => cancelAnimationFrame(klatka)
  })

  const m = WYKRES.margines
  const dziedzinaX = $derived.by((): [number, number] => {
    if (oknoMs) {
      const koniec = teraz - NA_ZYWO.opoznienieMs
      return [koniec - oknoMs, koniec]
    }
    if (punkty.length === 0) return [teraz - 1, teraz]
    return [punkty[0].t, punkty[punkty.length - 1].t]
  })

  const widoczne = $derived.by(() => {
    const [d0, d1] = dziedzinaX
    let pocz = 0
    let kon = punkty.length
    while (pocz < punkty.length - 1 && punkty[pocz + 1].t < d0) pocz++
    while (kon > pocz + 1 && punkty[kon - 2].t > d1) kon--
    return punkty.slice(pocz, kon)
  })

  const podzialkiY = $derived(podzialki(widoczne.reduce((x, p) => Math.max(x, p.a, p.b), 0), os))
  const maxY = new Tween(1, { duration: 300 })
  $effect(() => {
    maxY.target = podzialkiY[podzialkiY.length - 1]
  })

  const dol = $derived(wys - m.dol)
  const skalaX = $derived(liniowa(dziedzinaX[0], dziedzinaX[1], m.lewo, szer - m.prawo))
  const skalaY = $derived(liniowa(0, maxY.current || 1, dol, m.gora))
  const xs = $derived(widoczne.map((p) => skalaX(p.t)))
  const liniaA = $derived(sciezkaGladka(xs, widoczne.map((p) => skalaY(p.a))))
  const liniaB = $derived(sciezkaGladka(xs, widoczne.map((p) => skalaY(p.b))))
  const zamknij = (linia: string) => (xs.length > 1 ? `${linia} L${xs[xs.length - 1]},${dol} L${xs[0]},${dol} Z` : '')

  const podzialkiX = $derived.by(() => {
    const [d0, d1] = dziedzinaX
    const miejsc = Math.max(2, Math.floor((szer - m.lewo - m.prawo) / WYKRES.minOdstepOpisowCzasu))
    const krok = KROKI_CZASU.find((k) => (d1 - d0) / k <= miejsc) ?? KROKI_CZASU[KROKI_CZASU.length - 1]
    const wynik: number[] = []
    for (let t = Math.ceil(d0 / krok) * krok; t <= d1; t += krok) wynik.push(t)
    return wynik
  })

  const wskazany = $derived.by(() => {
    if (kursorX === null || xs.length === 0) return null
    const i = najblizszy(xs, kursorX)
    if (xs[i] < m.lewo || xs[i] > szer - m.prawo) return null
    return { x: xs[i], punkt: widoczne[i] }
  })
  const ostatni = $derived(punkty[punkty.length - 1] ?? null)

  function ruch(e: PointerEvent) {
    kursorX = e.clientX - (e.currentTarget as SVGSVGElement).getBoundingClientRect().left
  }
</script>

<div class="legenda">
  <span><span class="znak" style:background={kolorBarwy(barwaA)}></span>{nazwaA}<b>{ostatni ? formatY(ostatni.a) : '—'}</b></span>
  <span><span class="znak" style:background={kolorBarwy(barwaB)}></span>{nazwaB}<b>{ostatni ? formatY(ostatni.b) : '—'}</b></span>
</div>
<div class="wykres" class:niski use:obserwujRozmiar={(s, w) => { szer = s; wys = w }}>
  {#if szer > 0 && wys > 0}
    <svg width={szer} height={wys} viewBox="0 0 {szer} {wys}" onpointermove={ruch} onpointerleave={() => (kursorX = null)} role="img" aria-label="{nazwaA}, {nazwaB}">
      <defs>
        {#each [[idA, barwaA], [idB, barwaB]] as [id, b] (id)}
          <linearGradient {id} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" class="gradient-gora" style:stop-color={kolorBarwy(b as Barwa)} />
            <stop offset="1" class="gradient-dol" style:stop-color={kolorBarwy(b as Barwa)} />
          </linearGradient>
        {/each}
        <clipPath id={idPrzyciecia}>
          <rect x={m.lewo} y={0} width={Math.max(0, szer - m.lewo - m.prawo)} height={wys} />
        </clipPath>
      </defs>

      <g class="siatka">
        {#each podzialkiY as v (v)}
          {#if v <= maxY.current * 1.001}<line x1={m.lewo} x2={szer - m.prawo} y1={skalaY(v)} y2={skalaY(v)} />{/if}
        {/each}
      </g>
      <g class="opisy">
        {#each podzialkiY as v (v)}
          {#if v <= maxY.current * 1.001}
            <text x={m.lewo - WYKRES.odstepOpisu} y={skalaY(v)} text-anchor="end" dominant-baseline="middle">{formatY(v)}</text>
          {/if}
        {/each}
        {#each podzialkiX as t (t)}
          <text x={skalaX(t)} y={dol + WYKRES.odstepOpisu} text-anchor="middle" dominant-baseline="hanging">{godzina(t)}</text>
        {/each}
      </g>

      <g clip-path="url(#{idPrzyciecia})">
        {#if xs.length > 1}
          <path d={zamknij(liniaA)} fill="url(#{idA})" />
          <path d={zamknij(liniaB)} fill="url(#{idB})" />
        {/if}
        <path d={liniaA} class="linia" style:stroke={kolorBarwy(barwaA)} />
        <path d={liniaB} class="linia" style:stroke={kolorBarwy(barwaB)} />
      </g>

      {#if wskazany}
        <line class="kursor" x1={wskazany.x} x2={wskazany.x} y1={m.gora} y2={dol} />
        <circle class="kropka" cx={wskazany.x} cy={skalaY(wskazany.punkt.a)} r={WYKRES.promienKropki} style:fill={kolorBarwy(barwaA)} />
        <circle class="kropka" cx={wskazany.x} cy={skalaY(wskazany.punkt.b)} r={WYKRES.promienKropki} style:fill={kolorBarwy(barwaB)} />
      {/if}
    </svg>
    {#if wskazany}
      <Dymek x={wskazany.x / szer} y={skalaY(Math.max(wskazany.punkt.a, wskazany.punkt.b)) / wys}>
        <span class="dymek-tytul">{godzina(wskazany.punkt.t, !!oknoMs)}</span>
        <span class="dymek-wiersz">
          <span class="dymek-znacznik" style:background={kolorBarwy(barwaA)}></span>{nazwaA}
          <span class="dymek-wartosc">{formatY(wskazany.punkt.a)}</span>
        </span>
        <span class="dymek-wiersz">
          <span class="dymek-znacznik" style:background={kolorBarwy(barwaB)}></span>{nazwaB}
          <span class="dymek-wartosc">{formatY(wskazany.punkt.b)}</span>
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
  .wykres.niski {
    height: calc(var(--wysokosc-przycisk) * 4);
  }
  svg {
    display: block;
    overflow: visible;
    touch-action: pan-y;
  }
  .legenda {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-4);
    margin-bottom: var(--odstep-2);
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
  }
  .legenda > span {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-2);
  }
  .legenda b {
    font-family: var(--czcionka-mono);
    font-weight: var(--czcionka-polgruby);
    color: var(--kolor-tekst);
  }
  .znak {
    width: var(--odstep-3);
    height: var(--odstep-2);
    border-radius: var(--promien-kropka);
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
    stop-opacity: 0.25;
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
