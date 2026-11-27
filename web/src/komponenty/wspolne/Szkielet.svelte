<script lang="ts">
  // Szkielet ładowania: pulsujące paski w miejscu treści (zamiast pustego miejsca i przeskoku po wczytaniu).
  // `linie` = liczba pasków (ostatni krótszy); `wysokosc` = 'wiersz' | 'kafel' | 'wykres' (tokeny).
  interface Props {
    linie?: number
    wysokosc?: 'wiersz' | 'kafel' | 'wykres'
  }

  let { linie = 3, wysokosc = 'wiersz' }: Props = $props()
</script>

<div class="szkielet" aria-busy="true" aria-label="Wczytywanie">
  {#each { length: Math.max(1, linie) } as _, i (i)}
    <span class="pasek {wysokosc}" class:krotki={i === linie - 1 && linie > 1}></span>
  {/each}
</div>

<style>
  .szkielet {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-m);
    width: 100%;
  }
  .pasek {
    display: block;
    width: 100%;
    border-radius: var(--promien-segment);
    background: linear-gradient(
      90deg,
      var(--kolor-karta) 0%,
      var(--kolor-przycisk) 50%,
      var(--kolor-karta) 100%
    );
    background-size: 200% 100%;
    animation: fala 1.4s ease-in-out infinite;
  }
  .wiersz {
    height: var(--wysokosc-segment);
  }
  .kafel {
    height: calc(var(--czcionka-kpi) * 3);
    border-radius: var(--promien-karta);
  }
  .wykres {
    height: var(--wykres-wysokosc);
    border-radius: var(--promien-karta);
  }
  .krotki {
    width: 60%;
  }
  @keyframes fala {
    from {
      background-position: 100% 0;
    }
    to {
      background-position: -100% 0;
    }
  }
</style>
