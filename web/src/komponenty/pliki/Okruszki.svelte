<script lang="ts">
  // Okruszki ścieżki: udział / folder / folder; klik przenosi do wybranego poziomu.
  import Ikona from '../wspolne/Ikona.svelte'
  import { czesciSciezki } from './narzedzia'

  let { udzial, sciezka, idz }: { udzial: string; sciezka: string; idz: (sciezka: string) => void } = $props()

  const poziomy = $derived.by(() => {
    const czesci = czesciSciezki(sciezka)
    return czesci.map((nazwa, i) => ({ nazwa, sciezka: '/' + czesci.slice(0, i + 1).join('/') }))
  })
</script>

<nav class="okruszki" aria-label="Ścieżka">
  <button class="poziom" class:biezacy={poziomy.length === 0} onclick={() => idz('/')}>
    <Ikona nazwa="dysk" rozmiar="s" />
    <span class="skroc">{udzial}</span>
  </button>
  {#each poziomy as p, i (p.sciezka)}
    <span class="sep" aria-hidden="true">/</span>
    <button class="poziom" class:biezacy={i === poziomy.length - 1} onclick={() => idz(p.sciezka)}>
      <span class="skroc">{p.nazwa}</span>
    </button>
  {/each}
</nav>

<style>
  .okruszki {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--odstep-pozycje);
    min-width: 0;
  }
  .poziom {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
    max-width: var(--kolumna-xl);
    height: var(--wysokosc-segment);
    padding: 0 var(--odstep-s);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    font-weight: var(--czcionka-polgruby);
  }
  .poziom:hover {
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
  }
  .poziom.biezacy {
    color: var(--kolor-tekst);
  }
  .sep {
    color: var(--kolor-opis);
  }
</style>
