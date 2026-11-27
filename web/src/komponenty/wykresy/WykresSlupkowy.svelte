<script lang="ts">
  // Wykres słupkowy z dwiema seriami i dwiema osiami: słupki (seria A, oś lewa, np. bajty)
  // i linia z kropkami (seria B, oś prawa, np. liczba pobrań). Siatka z osi lewej, dymek z obiema wartościami.
  import { Tween } from 'svelte/motion'
  import Dymek from './Dymek.svelte'
  import { WYKRES } from './stale'
  import { kolorBarwy, liniowa, obserwujRozmiar, podzialki, type PunktSlupka, type Seria } from './skala'

  interface Props { punkty: PunktSlupka[]; seriaA: Seria; seriaB: Seria }
  let { punkty, seriaA, seriaB }: Props = $props()

  let szer = $state(0)
  let wys = $state(0)
  let wskazanyIndeks = $state<number | null>(null)

  const m = $derived({ ...WYKRES.margines, prawo: WYKRES.marginesPrawaOs })
  const podzialkiA = $derived(podzialki(punkty.reduce((x, p) => Math.max(x, p.a), 0), seriaA.os))
  const podzialkiB = $derived(podzialki(punkty.reduce((x, p) => Math.max(x, p.b), 0), seriaB.os))
  const maxA = new Tween(1, { duration: 300 })
  const maxB = new Tween(1, { duration: 300 })
  $effect(() => {
    maxA.target = podzialkiA[podzialkiA.length - 1]
    maxB.target = podzialkiB[podzialkiB.length - 1]
  })

  const dol = $derived(wys - m.dol)
  const szerDanych = $derived(Math.max(0, szer - m.lewo - m.prawo))
  const pasmo = $derived(punkty.length ? szerDanych / punkty.length : 0)
  const szerSlupka = $derived(pasmo * WYKRES.wypelnienieSlupka)
  const skalaA = $derived(liniowa(0, maxA.current || 1, dol, m.gora))
  const skalaB = $derived(liniowa(0, maxB.current || 1, dol, m.gora))
  const srodek = (i: number) => m.lewo + pasmo * (i + 0.5)
  const co = $derived(pasmo > 0 ? Math.max(1, Math.ceil(WYKRES.minOdstepOpisowCzasu / pasmo)) : 1)
  const liniaB = $derived(punkty.map((p, i) => `${i ? 'L' : 'M'}${srodek(i)},${skalaB(p.b)}`).join(' '))

  function ruch(e: PointerEvent) {
    const prost = (e.currentTarget as SVGSVGElement).getBoundingClientRect()
    const x = e.clientX - prost.left - m.lewo
    const i = Math.floor(x / (pasmo || 1))
    wskazanyIndeks = i >= 0 && i < punkty.length ? i : null
  }
</script>

<div class="wykres" use:obserwujRozmiar={(s, w) => { szer = s; wys = w }}>
  {#if szer > 0 && wys > 0}
    <svg width={szer} height={wys} viewBox="0 0 {szer} {wys}" onpointermove={ruch} onpointerleave={() => (wskazanyIndeks = null)} role="img" aria-label="{seriaA.nazwa}, {seriaB.nazwa}">
      <g class="siatka">
        {#each podzialkiA as v (v)}
          {#if v <= maxA.current * 1.001}
            <line x1={m.lewo} x2={szer - m.prawo} y1={skalaA(v)} y2={skalaA(v)} />
          {/if}
        {/each}
      </g>

      {#if wskazanyIndeks !== null}
        <rect class="pasmo" x={m.lewo + pasmo * wskazanyIndeks} y={m.gora} width={pasmo} height={Math.max(0, dol - m.gora)} />
      {/if}

      <g class="opisy">
        {#each podzialkiA as v (v)}
          {#if v <= maxA.current * 1.001}
            <text x={m.lewo - WYKRES.odstepOpisu} y={skalaA(v)} text-anchor="end" dominant-baseline="middle">{seriaA.format(v)}</text>
          {/if}
        {/each}
        {#each podzialkiB as v (v)}
          {#if v <= maxB.current * 1.001}
            <text x={szer - m.prawo + WYKRES.odstepOpisu} y={skalaB(v)} text-anchor="start" dominant-baseline="middle">{seriaB.format(v)}</text>
          {/if}
        {/each}
        {#each punkty as p, i (i)}
          {#if i % co === 0}
            <text x={srodek(i)} y={dol + WYKRES.odstepOpisu} text-anchor="middle" dominant-baseline="hanging">{p.etykieta}</text>
          {/if}
        {/each}
      </g>

      <g style:fill={kolorBarwy(seriaA.barwa)}>
        {#each punkty as p, i (i)}
          {@const h = Math.max(0, dol - skalaA(p.a))}
          <rect class="slupek" class:przygaszony={wskazanyIndeks !== null && wskazanyIndeks !== i}
            x={srodek(i) - szerSlupka / 2} y={dol - h} width={szerSlupka} height={h}
            rx={Math.min(WYKRES.zaokraglenieSlupka, szerSlupka / 2)} />
        {/each}
      </g>

      {#if punkty.length > 1}
        <path class="linia" d={liniaB} style:stroke={kolorBarwy(seriaB.barwa)} />
      {/if}
      {#each punkty as p, i (i)}
        <circle class="kropka" cx={srodek(i)} cy={skalaB(p.b)} r={wskazanyIndeks === i ? WYKRES.promienKropki : WYKRES.promienKropki / 2}
          style:fill={kolorBarwy(seriaB.barwa)} />
      {/each}
    </svg>

    {#if wskazanyIndeks !== null}
      {@const p = punkty[wskazanyIndeks]}
      <Dymek x={srodek(wskazanyIndeks) / szer} y={skalaA(p.a) / wys}>
        <span class="dymek-tytul">{p.opis}</span>
        <span class="dymek-wiersz">
          <span class="dymek-znacznik" style:background={kolorBarwy(seriaA.barwa)}></span>
          {seriaA.nazwa}<span class="dymek-wartosc">{seriaA.format(p.a)}</span>
        </span>
        <span class="dymek-wiersz">
          <span class="dymek-znacznik" style:background={kolorBarwy(seriaB.barwa)}></span>
          {seriaB.nazwa}<span class="dymek-wartosc">{seriaB.format(p.b)}</span>
        </span>
      </Dymek>
    {/if}
  {/if}
</div>

<div class="legenda">
  <span><span class="znak" style:background={kolorBarwy(seriaA.barwa)}></span>{seriaA.nazwa} (oś lewa)</span>
  <span><span class="znak kolko" style:background={kolorBarwy(seriaB.barwa)}></span>{seriaB.nazwa} (oś prawa)</span>
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
  .pasmo {
    fill: var(--kolor-siatka);
    opacity: 0.5;
  }
  .slupek {
    transition: opacity var(--czas-szybki);
  }
  .slupek.przygaszony {
    opacity: 0.45;
  }
  .linia {
    fill: none;
    stroke-width: 2;
    stroke-linejoin: round;
  }
  .kropka {
    stroke: var(--kolor-panel);
    stroke-width: 1;
  }
  .legenda {
    display: flex;
    gap: var(--odstep-4);
    justify-content: center;
    margin-top: var(--odstep-2);
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
  }
  .legenda > span {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-2);
  }
  .znak {
    width: var(--odstep-3);
    height: var(--odstep-2);
    border-radius: var(--promien-kropka);
  }
  .znak.kolko {
    width: var(--odstep-2);
    border-radius: 50%;
  }
</style>
