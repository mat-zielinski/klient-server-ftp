<script lang="ts">
  // Półokrągły wskaźnik (jak w UniFi): tor łuku, łuk wartości w barwie zależnej od progów, liczba w środku,
  // etykieta i opis pod spodem. `wartosc` to ułamek 0..1 (null = brak danych, pusty tor i „—”).
  // Skaluje się przez viewBox do szerokości kontenera; wartość zmienia się płynnie (Tween).
  import { Tween } from 'svelte/motion'
  import { kolorBarwy, type Barwa } from './skala'

  interface Props {
    wartosc: number | null
    etykieta: string
    /** Tekst w środku zamiast procentu (np. „3,2 GB”). */
    tekst?: string
    opis?: string
    /** Progi [ostrzeżenie, alarm] jako ułamki; poniżej pierwszego barwa `barwa`. */
    progi?: [number, number]
    barwa?: Barwa
  }
  let { wartosc, etykieta, tekst, opis, progi = [0.7, 0.9], barwa = 'niebieski' }: Props = $props()

  // Geometria w jednostkach viewBox (nie piksele): łuk o promieniu R ze środkiem (SX, SY).
  const SX = 60
  const SY = 56
  const R = 46
  const GRUBOSC = 10
  const LUK = `M${SX - R},${SY} A${R},${R} 0 0 1 ${SX + R},${SY}`

  const ulamek = $derived(wartosc === null || !Number.isFinite(wartosc) ? 0 : Math.min(1, Math.max(0, wartosc)))
  const plynny = new Tween(0, { duration: 400 })
  $effect(() => {
    plynny.target = ulamek
  })

  const kolor = $derived(
    wartosc === null ? 'var(--kolor-siatka)'
    : ulamek >= progi[1] ? 'var(--kolor-czerwony)'
    : ulamek >= progi[0] ? 'var(--kolor-zolty)'
    : kolorBarwy(barwa),
  )
  const napis = $derived(tekst ?? (wartosc === null ? '—' : `${Math.round(ulamek * 100)}%`))
</script>

<div class="wskaznik" role="meter" aria-label={etykieta} aria-valuemin={0} aria-valuemax={100}
  aria-valuenow={wartosc === null ? undefined : Math.round(ulamek * 100)}>
  <div class="luk">
    <svg viewBox="0 0 {SX * 2} {SY + GRUBOSC / 2 + 1}" aria-hidden="true">
      <path d={LUK} class="tor" stroke-width={GRUBOSC} pathLength="100" />
      {#if plynny.current > 0.001}
        <path d={LUK} class="wartosc" stroke-width={GRUBOSC} pathLength="100"
          stroke-dasharray="{plynny.current * 100} 100" style:stroke={kolor} />
      {/if}
    </svg>
    <span class="liczba" style:color={wartosc === null ? 'var(--kolor-opis)' : 'var(--kolor-tekst)'}>{napis}</span>
  </div>
  <span class="etykieta">{etykieta}</span>
  {#if opis}<span class="opis" title={opis}>{opis}</span>{/if}
</div>

<style>
  .wskaznik {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--odstep-1);
    min-width: 0;
  }
  .luk {
    position: relative;
    width: 100%;
    max-width: var(--pierscien-rozmiar, var(--uklad-kafel-min-w));
  }
  svg {
    display: block;
    width: 100%;
  }
  path {
    fill: none;
    stroke-linecap: round;
  }
  .tor {
    stroke: var(--kolor-siatka);
  }
  .wartosc {
    transition: stroke var(--czas-sredni);
  }
  .liczba {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    text-align: center;
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-kpi);
    font-weight: var(--czcionka-gruby);
    line-height: 1;
  }
  .etykieta {
    font-size: var(--czcionka-etykieta);
    font-weight: var(--czcionka-polgruby);
    letter-spacing: var(--czcionka-rozstaw-etykiety);
    text-transform: uppercase;
    color: var(--kolor-opis);
  }
  .opis {
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
    font-family: var(--czcionka-mono);
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
