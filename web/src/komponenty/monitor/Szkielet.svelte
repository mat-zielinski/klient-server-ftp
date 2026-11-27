<script lang="ts">
  // Szkielet ładowania: przygaszony prostokąt z pulsowaniem w miejscu treści, która jeszcze nie przyszła.
  interface Props {
    rodzaj?: 'linia' | 'kafel' | 'wykres' | 'wskaznik'
    /** Ile szkieletów obok siebie (siatka zawija się jak kafle). */
    ile?: number
  }
  let { rodzaj = 'linia', ile = 1 }: Props = $props()
</script>

<div class="szkielety {rodzaj}" aria-busy="true" aria-label="Wczytywanie">
  {#each Array.from({ length: ile }, (_, i) => i) as i (i)}
    <span class="szkielet"></span>
  {/each}
</div>

<style>
  .szkielety {
    display: grid;
    gap: var(--odstep-l);
    grid-template-columns: repeat(auto-fill, minmax(var(--uklad-kafel-min-w), 1fr));
  }
  .szkielet {
    display: block;
    border-radius: var(--promien-kafel);
    background: var(--kolor-siatka);
    animation: puls calc(var(--czas-sredni) * 7) ease-in-out infinite;
  }
  .linia .szkielet {
    height: var(--wysokosc-segment);
  }
  .kafel .szkielet {
    height: calc(var(--wysokosc-przycisk-duzy) * 2);
  }
  .wskaznik .szkielet {
    height: var(--uklad-kafel-min-w);
  }
  .wykres {
    grid-template-columns: minmax(0, 1fr);
  }
  .wykres .szkielet {
    height: var(--wykres-wysokosc, calc(var(--wysokosc-przycisk) * 8));
  }
  @keyframes puls {
    0%,
    100% {
      opacity: 0.35;
    }
    50% {
      opacity: 0.8;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .szkielet {
      animation: none;
      opacity: 0.5;
    }
  }
</style>
