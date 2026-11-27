<script lang="ts">
  // Wskaźnik kroków kreatora: kółka z numerem i podpisem połączone paskiem postępu (wypełnienie animowane).
  // Ukończone = ptaszek i klikalne (powrót do kroku), bieżący = akcent z poświatą, przyszłe = szare.
  import Ikona from '../wspolne/Ikona.svelte'

  interface Props {
    kroki: string[]
    /** Indeks bieżącego kroku (od 0). */
    biezacy: number
    /** Klik w ukończony krok (powrót). Bez tej funkcji kroki nie są klikalne. */
    onwybierz?: (i: number) => void
    /** Podpisy schowane (wąski ekran): zostaje podpis bieżącego kroku pod paskiem. */
    zwarte?: boolean
  }

  let { kroki, biezacy, onwybierz, zwarte = false }: Props = $props()

  const postep = $derived(kroki.length > 1 ? (biezacy / (kroki.length - 1)) * 100 : 0)
</script>

<nav class="kroki" aria-label="Kroki kreatora" style:--kolumny={kroki.length}>
  <div class="tor" aria-hidden="true">
    <div class="wypelnienie" style:width="{postep}%"></div>
  </div>
  <ol>
    {#each kroki as nazwa, i (nazwa)}
      {@const zrobiony = i < biezacy}
      <li class:zrobiony class:biezacy={i === biezacy}>
        <button
          type="button"
          class="krok"
          disabled={!zrobiony || !onwybierz}
          aria-current={i === biezacy ? 'step' : undefined}
          title={zrobiony ? `Wróć do kroku: ${nazwa}` : nazwa}
          onclick={() => onwybierz?.(i)}
        >
          <span class="numer">
            {#if zrobiony}<Ikona nazwa="ok" rozmiar="s" />{:else}{i + 1}{/if}
          </span>
          <span class="podpis skroc" class:tylko-czytnik={zwarte}>{nazwa}</span>
        </button>
      </li>
    {/each}
  </ol>
  {#if zwarte}
    <p class="biezacy-podpis">Krok {biezacy + 1} z {kroki.length}: <strong>{kroki[biezacy]}</strong></p>
  {/if}
</nav>

<style>
  .kroki {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-m);
  }
  /* tor biegnie przez środki pierwszego i ostatniego kółka */
  .tor {
    position: absolute;
    top: calc(var(--wysokosc-znaczek) / 2 - var(--wysokosc-postep-krokow) / 2);
    left: calc(50% / var(--kolumny));
    right: calc(50% / var(--kolumny));
    height: var(--wysokosc-postep-krokow);
    border-radius: var(--promien-kropka);
    background: var(--kolor-siatka);
    overflow: hidden;
  }
  .wypelnienie {
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--kolor-zielony), var(--kolor-niebieski));
    transition: width var(--czas-postep) var(--krzywa-wejscia);
  }
  ol {
    position: relative;
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(var(--kolumny), minmax(0, 1fr));
  }
  .krok {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--odstep-s);
    padding: 0;
    border: 0;
    background: none;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
    cursor: default;
  }
  .krok:not(:disabled) {
    cursor: pointer;
  }
  .krok:disabled {
    opacity: 1;
  }
  .krok:focus-visible .numer {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
    outline-offset: var(--odstep-pozycje);
  }
  .numer {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-znaczek);
    height: var(--wysokosc-znaczek);
    border-radius: var(--promien-kolo);
    background: var(--kolor-panel);
    border: var(--odstep-ramka) solid var(--kolor-siatka);
    font-weight: var(--czcionka-gruby);
    transition:
      background var(--czas-sredni),
      color var(--czas-sredni),
      box-shadow var(--czas-sredni),
      transform var(--czas-sredni);
  }
  .podpis {
    max-width: 100%;
    padding: 0 var(--odstep-xs);
  }
  .biezacy .krok {
    color: var(--kolor-tekst);
    font-weight: var(--czcionka-polgruby);
  }
  .biezacy .numer {
    background: var(--kolor-niebieski-tlo);
    border-color: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
    box-shadow: 0 0 0 var(--odstep-xs) var(--kolor-fokus-pola);
  }
  .zrobiony .numer {
    background: var(--kolor-zielony-tlo);
    border-color: var(--kolor-zielony-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .zrobiony .krok:hover .numer {
    transform: scale(1.08);
  }
  .zrobiony .krok:hover .podpis {
    color: var(--kolor-tekst);
    text-decoration: underline;
  }
  .biezacy-podpis {
    margin: 0;
    text-align: center;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .biezacy-podpis strong {
    color: var(--kolor-tekst);
  }
  @media (prefers-reduced-motion: reduce) {
    .wypelnienie,
    .numer {
      transition: none;
    }
  }
</style>
