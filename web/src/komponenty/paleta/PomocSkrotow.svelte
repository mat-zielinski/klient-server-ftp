<script lang="ts">
  // Okno pomocy „?”: skróty wbudowane i skróty wszystkich aktywnych poleceń, pogrupowane.
  import Okno from '../wspolne/Okno.svelte'
  import Klawisze from './Klawisze.svelte'
  import {
    skroty,
    GRUPA_DOMYSLNA,
    SKROT_PALETY,
    SKROT_POMOCY,
    SKROT_SZUKANIA,
  } from '../../stan/skroty.svelte'

  const WBUDOWANE = [
    { nazwa: 'Paleta poleceń', skrot: SKROT_PALETY },
    { nazwa: 'Szukaj', skrot: SKROT_SZUKANIA },
    { nazwa: 'Ta pomoc', skrot: SKROT_POMOCY },
    { nazwa: 'Zamknij okno lub menu', skrot: 'Esc' },
  ]

  const grupy = $derived.by(() => {
    const mapa = new Map<string, { nazwa: string; skrot: string }[]>([['Ogólne', WBUDOWANE]])
    for (const p of skroty.aktywne) {
      if (!p.skrot) continue
      const g = p.grupa ?? GRUPA_DOMYSLNA
      mapa.set(g, [...(mapa.get(g) ?? []), { nazwa: p.nazwa, skrot: p.skrot }])
    }
    return [...mapa]
  })
</script>

<Okno tytul="Skróty klawiszowe" bind:otwarte={skroty.pomoc}>
  <div class="grupy">
    {#each grupy as [grupa, lista] (grupa)}
      <section>
        <h4 class="etykieta">{grupa}</h4>
        <dl>
          {#each lista as s (s.nazwa + s.skrot)}
            <div class="wiersz-skrotu">
              <dt>{s.nazwa}</dt>
              <dd><Klawisze skrot={s.skrot} /></dd>
            </div>
          {/each}
        </dl>
      </section>
    {/each}
  </div>
</Okno>

<style>
  .grupy {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-sekcja);
  }
  h4 {
    margin-bottom: var(--odstep-s);
  }
  dl {
    margin: 0;
  }
  .wiersz-skrotu {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-l);
    min-height: var(--wysokosc-przycisk-maly);
    border-bottom: var(--ramka);
  }
  .wiersz-skrotu:last-child {
    border-bottom: 0;
  }
  dt {
    font-size: var(--czcionka-maly);
  }
  dd {
    margin: 0;
  }
</style>
