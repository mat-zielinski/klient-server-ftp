<script lang="ts">
  // Ranking (top pliki, top klienci): nazwa, wartość i pasek proporcjonalny do największej pozycji.
  import { kolorBarwy, type Barwa, type Pozycja } from './skala'

  interface Props {
    pozycje: Pozycja[]
    format: (v: number) => string
    barwa?: Barwa
    pusto?: string
  }
  let { pozycje, format, barwa = 'niebieski', pusto = 'Brak danych w tym zakresie.' }: Props = $props()

  const max = $derived(pozycje.reduce((a, p) => Math.max(a, p.wartosc), 0) || 1)
</script>

{#if pozycje.length === 0}
  <p class="pusto">{pusto}</p>
{:else}
  <ol class="ranking">
    {#each pozycje as p, i (i)}
      <li title={p.opis ?? p.nazwa}>
        <span class="miejsce">{i + 1}</span>
        <span class="nazwa">{p.nazwa}</span>
        <span class="wartosc">{format(p.wartosc)}</span>
        <span class="tor"><span class="pasek" style:width="{(p.wartosc / max) * 100}%" style:background={kolorBarwy(barwa)}></span></span>
        {#if p.opis}<span class="opis">{p.opis}</span>{/if}
      </li>
    {/each}
  </ol>
{/if}

<style>
  .ranking {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-3);
  }
  li {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    column-gap: var(--odstep-2);
    row-gap: var(--odstep-1);
    align-items: center;
    font-size: var(--czcionka-maly);
  }
  .miejsce {
    font-family: var(--czcionka-mono);
    color: var(--kolor-opis);
    min-width: 2ch;
    text-align: right;
  }
  .nazwa {
    color: var(--kolor-tekst);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .wartosc {
    font-family: var(--czcionka-mono);
    color: var(--kolor-tekst);
  }
  .tor {
    grid-column: 2 / 4;
    height: var(--odstep-1);
    background: var(--kolor-siatka);
    border-radius: var(--promien-kropka);
    overflow: hidden;
  }
  .pasek {
    display: block;
    height: 100%;
    border-radius: var(--promien-kropka);
    transition: width var(--czas-szybki);
  }
  .opis {
    grid-column: 2 / 4;
    color: var(--kolor-opis);
  }
  .pusto {
    margin: 0;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
</style>
