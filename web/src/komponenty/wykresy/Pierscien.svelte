<script lang="ts">
  // Pierścień podziału (np. protokoły): łuki z przerwami, środek pokazuje sumę albo wskazany wycinek,
  // legenda z udziałem procentowym. Skaluje się przez viewBox (kwadrat dopasowany do kontenera).
  import { WYKRES } from './stale'
  import { BARWY_SERII, kolorBarwy, type Pozycja } from './skala'

  interface Props { pozycje: Pozycja[]; format: (v: number) => string; podpisSumy?: string }
  let { pozycje, format, podpisSumy = 'Razem' }: Props = $props()

  const R = 50
  const r = R * (1 - WYKRES.gruboscPierscienia)
  const PELNY = Math.PI * 2
  let wskazany = $state<number | null>(null)

  const suma = $derived(pozycje.reduce((a, p) => a + Math.max(0, p.wartosc), 0))
  const procent = (v: number) => (suma > 0 ? Math.round((v / suma) * 1000) / 10 : 0)

  function punkt(promien: number, kat: number): string {
    return `${R + promien * Math.sin(kat)},${R - promien * Math.cos(kat)}`
  }
  function luk(od: number, doKata: number): string {
    const duzy = doKata - od > Math.PI ? 1 : 0
    return `M${punkt(R, od)} A${R},${R} 0 ${duzy} 1 ${punkt(R, doKata)} L${punkt(r, doKata)} A${r},${r} 0 ${duzy} 0 ${punkt(r, od)} Z`
  }

  const luki = $derived.by(() => {
    const widoczne = pozycje.filter((p) => p.wartosc > 0)
    const przerwa = widoczne.length > 1 ? WYKRES.przerwaPierscienia : 0
    let kat = 0
    return pozycje.map((p, i) => {
      const rozpietosc = suma > 0 ? (Math.max(0, p.wartosc) / suma) * PELNY : 0
      const od = kat
      kat += rozpietosc
      if (rozpietosc <= 0) return { d: '', barwa: BARWY_SERII[i % BARWY_SERII.length] }
      const koniec = Math.min(od + rozpietosc - przerwa, od + PELNY - 1e-4)
      return { d: luk(od, Math.max(od + 1e-4, koniec)), barwa: BARWY_SERII[i % BARWY_SERII.length] }
    })
  })
</script>

<div class="pierscien">
  <div class="kolo">
    <svg viewBox="0 0 {R * 2} {R * 2}" role="img" aria-label="Podział">
      {#if suma <= 0}
        <path class="pusty" d={luk(0, PELNY - 1e-4)} />
      {/if}
      {#each luki as l, i (i)}
        {#if l.d}
          <path d={l.d} style:fill={kolorBarwy(l.barwa)} class:przygaszony={wskazany !== null && wskazany !== i}
            role="presentation" onpointerenter={() => (wskazany = i)} onpointerleave={() => (wskazany = null)} />
        {/if}
      {/each}
    </svg>
    <div class="srodek">
      {#if wskazany !== null && pozycje[wskazany]}
        <span class="wartosc">{procent(pozycje[wskazany].wartosc)}%</span>
        <span class="podpis">{pozycje[wskazany].nazwa}</span>
      {:else}
        <span class="wartosc">{format(suma)}</span>
        <span class="podpis">{podpisSumy}</span>
      {/if}
    </div>
  </div>

  <ul class="legenda">
    {#each pozycje as p, i (p.nazwa)}
      <li class:wskazany={wskazany === i} onpointerenter={() => (wskazany = i)} onpointerleave={() => (wskazany = null)}>
        <span class="znak" style:background={kolorBarwy(BARWY_SERII[i % BARWY_SERII.length])}></span>
        <span class="nazwa">{p.nazwa}</span>
        <span class="liczba">{format(p.wartosc)}</span>
        <span class="proc">{procent(p.wartosc)}%</span>
      </li>
    {/each}
  </ul>
</div>

<style>
  .pierscien {
    display: flex;
    align-items: center;
    gap: var(--odstep-4);
    flex-wrap: wrap;
    justify-content: center;
  }
  .kolo {
    position: relative;
    width: var(--pierscien-rozmiar, var(--uklad-kafel-min-w));
    aspect-ratio: 1;
    flex: none;
  }
  svg {
    display: block;
    width: 100%;
    height: 100%;
  }
  path {
    transition: opacity var(--czas-szybki);
  }
  .przygaszony {
    opacity: 0.35;
  }
  .pusty {
    fill: var(--kolor-siatka);
  }
  .srodek {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    pointer-events: none;
    text-align: center;
  }
  .wartosc {
    font-family: var(--czcionka-mono);
    font-weight: var(--czcionka-gruby);
    font-size: var(--czcionka-kpi);
    color: var(--kolor-tekst);
  }
  .podpis {
    font-size: var(--czcionka-maly);
    color: var(--kolor-opis);
    text-transform: uppercase;
  }
  .legenda {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-1);
    min-width: 0;
    flex: 1;
  }
  li {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    align-items: center;
    gap: var(--odstep-2);
    padding: var(--odstep-1) var(--odstep-2);
    border-radius: var(--promien-kropka);
    font-size: var(--czcionka-maly);
  }
  li.wskazany {
    background: var(--kolor-aktywny);
  }
  .znak {
    width: var(--odstep-2);
    height: var(--odstep-2);
    border-radius: 50%;
  }
  .nazwa {
    color: var(--kolor-tekst);
    text-transform: uppercase;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .liczba {
    font-family: var(--czcionka-mono);
    color: var(--kolor-tekst);
  }
  .proc {
    font-family: var(--czcionka-mono);
    color: var(--kolor-opis);
    text-align: right;
    min-width: 5ch;
  }
</style>
