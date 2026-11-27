<script lang="ts">
  // Pigułka stanu (.st stacji, KOLOR_STANU): tekst w kolorze stanu na tle stanu, opcjonalna kropka.
  import type { Snippet } from 'svelte'

  interface Props {
    stan: 'ok' | 'info' | 'blad' | 'ostrz' | 'wyl'
    children?: Snippet
    kropka?: boolean
  }

  let { stan, children, kropka = false }: Props = $props()
</script>

<span class="pigulka {stan}">
  {#if kropka}<span class="kropka"></span>{/if}
  {@render children?.()}
</span>

<style>
  .pigulka {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-s);
    height: var(--wysokosc-znaczek);
    padding: 0 var(--odstep-przycisk-maly);
    border-radius: var(--promien-pigulka);
    font-size: var(--czcionka-drobny);
    font-weight: var(--czcionka-gruby);
    white-space: nowrap;
    flex: none;
  }
  .kropka {
    width: var(--wysokosc-kropka);
    height: var(--wysokosc-kropka);
    border-radius: var(--promien-kolo);
    background: currentColor;
  }
  .ok {
    color: var(--kolor-zielony);
    background: var(--kolor-tlo-ok);
  }
  .info {
    color: var(--kolor-niebieski);
    background: var(--kolor-tlo-info);
  }
  .blad {
    color: var(--kolor-czerwony);
    background: var(--kolor-tlo-blad);
  }
  .ostrz {
    color: var(--kolor-zolty);
    background: var(--kolor-tlo-ostrz);
  }
  .wyl {
    color: var(--kolor-opis);
    background: var(--kolor-karta);
  }
</style>
