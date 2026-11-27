<script lang="ts">
  // Wybór zakresu Dziennika jak segmenty stacji: Dziś / 7 dni / 30 dni / Własny (pola dat), oraz krok wykresu.
  import Pole from '../wspolne/Pole.svelte'
  import Wybor from '../wspolne/Wybor.svelte'
  import { KROKI, ZAKRESY, krokDomyslny, type Krok, type RodzajZakresu, type Zakres } from './zakres'

  interface Props { zakres: Zakres }
  let { zakres = $bindable() }: Props = $props()

  const opcjeKroku = KROKI.map((k) => ({ wartosc: k.id, etykieta: k.etykieta }))

  function wybierz(rodzaj: RodzajZakresu) {
    zakres = { ...zakres, rodzaj, krok: krokDomyslny(rodzaj) }
  }
</script>

<div class="wybor-zakresu">
  <div class="segmenty" role="radiogroup" aria-label="Zakres">
    {#each ZAKRESY as z (z.id)}
      <button type="button" role="radio" aria-checked={zakres.rodzaj === z.id} class:wybrany={zakres.rodzaj === z.id}
        onclick={() => wybierz(z.id)}>{z.etykieta}</button>
    {/each}
  </div>

  {#if zakres.rodzaj === 'wlasny'}
    <div class="daty">
      <Pole etykieta="Od" typ="date" ukryjEtykiete bind:value={zakres.od} />
      <span class="opis">do</span>
      <Pole etykieta="Do" typ="date" ukryjEtykiete bind:value={zakres.do} />
    </div>
  {/if}

  <div class="krok">
    <Wybor etykieta="Krok" ukryjEtykiete opcje={opcjeKroku} value={zakres.krok}
      onchange={(w) => (zakres = { ...zakres, krok: w as Krok })} />
  </div>
</div>

<style>
  .wybor-zakresu {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--odstep-3);
  }
  .segmenty {
    display: inline-flex;
    padding: var(--odstep-wnetrze-segmentu);
    gap: var(--odstep-wnetrze-segmentu);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .segmenty button {
    height: var(--wysokosc-segment);
    padding: 0 var(--odstep-4);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    cursor: pointer;
    transition: background var(--czas-szybki), color var(--czas-szybki);
  }
  .segmenty button:hover {
    color: var(--kolor-tekst);
    background: var(--kolor-przycisk-hover);
  }
  .segmenty button.wybrany {
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .daty {
    display: flex;
    align-items: center;
    gap: var(--odstep-2);
  }
  .krok {
    min-width: var(--kolumna-m);
  }
</style>
