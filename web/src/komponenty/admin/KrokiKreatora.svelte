<script lang="ts">
  // Pasek kroków kreatora: numer w kółku + podpis; zrobione = zielony znaczek, bieżący = akcent.
  import Ikona from '../wspolne/Ikona.svelte'

  interface Props {
    kroki: string[]
    /** Indeks bieżącego kroku (od 0). */
    biezacy: number
  }

  let { kroki, biezacy }: Props = $props()
</script>

<ol class="kroki">
  {#each kroki as nazwa, i (nazwa)}
    <li class:zrobiony={i < biezacy} class:biezacy={i === biezacy} aria-current={i === biezacy ? 'step' : undefined}>
      <span class="numer">
        {#if i < biezacy}<Ikona nazwa="ok" rozmiar="s" />{:else}{i + 1}{/if}
      </span>
      <span class="podpis skroc">{nazwa}</span>
    </li>
  {/each}
</ol>

<style>
  .kroki {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    gap: var(--odstep-m);
  }
  li {
    flex: 1 1 0;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: var(--odstep-s);
    padding-bottom: var(--odstep-m);
    border-bottom: var(--odstep-pozycje) solid var(--kolor-siatka);
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .numer {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-znaczek);
    height: var(--wysokosc-znaczek);
    border-radius: var(--promien-kolo);
    background: var(--kolor-przycisk);
    font-weight: var(--czcionka-gruby);
  }
  .biezacy {
    color: var(--kolor-tekst);
    border-bottom-color: var(--kolor-niebieski);
  }
  .biezacy .numer {
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .zrobiony {
    border-bottom-color: var(--kolor-zielony-tlo);
  }
  .zrobiony .numer {
    background: var(--kolor-tlo-ok);
    color: var(--kolor-zielony);
  }
</style>
