<script lang="ts">
  // Dymek wykresu: pozycja jako ułamek rozmiaru kontenera (0..1), sam odbija się od prawej i górnej krawędzi.
  import type { Snippet } from 'svelte'

  interface Props { x: number; y: number; children: Snippet }
  let { x, y, children }: Props = $props()

  const PROG_ODBICIA = 0.6
  const naLewo = $derived(x > PROG_ODBICIA)
  const wDol = $derived(y < 1 - PROG_ODBICIA)
</script>

<div
  class="dymek"
  class:na-lewo={naLewo}
  class:w-dol={wDol}
  style:left="{x * 100}%"
  style:top="{y * 100}%"
  role="tooltip"
>
  {@render children()}
</div>

<style>
  .dymek {
    position: absolute;
    pointer-events: none;
    z-index: 2;
    transform: translate(var(--odstep-3), calc(-100% - var(--odstep-2)));
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    box-shadow: 0 var(--odstep-1) var(--odstep-5) var(--kolor-cien);
    padding: var(--odstep-2) var(--odstep-3);
    font-size: var(--czcionka-maly);
    color: var(--kolor-tekst);
    white-space: nowrap;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-1);
  }
  .dymek.na-lewo {
    transform: translate(calc(-100% - var(--odstep-3)), calc(-100% - var(--odstep-2)));
  }
  .dymek.w-dol {
    transform: translate(var(--odstep-3), var(--odstep-2));
  }
  .dymek.na-lewo.w-dol {
    transform: translate(calc(-100% - var(--odstep-3)), var(--odstep-2));
  }
  /* Treść dymka (wspólna dla wszystkich wykresów): tytuł, wiersze „znacznik nazwa wartość”. */
  .dymek :global(.dymek-tytul) {
    color: var(--kolor-opis);
  }
  .dymek :global(.dymek-wiersz) {
    display: flex;
    align-items: center;
    gap: var(--odstep-2);
  }
  .dymek :global(.dymek-znacznik) {
    width: var(--odstep-2);
    height: var(--odstep-2);
    border-radius: 50%;
    flex: none;
  }
  .dymek :global(.dymek-wartosc) {
    font-family: var(--czcionka-mono);
    margin-left: auto;
    padding-left: var(--odstep-3);
  }
</style>
