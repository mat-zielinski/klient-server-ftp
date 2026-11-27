<script lang="ts">
  // Skrót klawiszowy jako klawisze <kbd> („Ctrl” „K”).
  import { klawiszeSkrotu } from '../../stan/skroty.svelte'

  let { skrot }: { skrot: string } = $props()
</script>

<span class="klawisze">
  {#each klawiszeSkrotu(skrot) as k, i (i)}<kbd>{k}</kbd>{/each}
</span>

<style>
  .klawisze {
    display: inline-flex;
    gap: var(--odstep-pozycje);
    flex: none;
  }
  kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: var(--wysokosc-kropka);
    height: calc(var(--wysokosc-znaczek) - var(--odstep-xs));
    padding: 0 var(--odstep-s);
    border: var(--ramka);
    border-bottom-width: calc(var(--odstep-ramka) * 2);
    border-radius: var(--promien-kropka);
    background: var(--kolor-karta);
    color: var(--kolor-opis);
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-drobny);
  }
</style>
