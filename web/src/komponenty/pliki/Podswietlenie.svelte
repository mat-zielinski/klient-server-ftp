<script lang="ts">
  // Tekst z podświetlonymi fragmentami pasującymi do zapytania (bez wielkości liter; słowa zapytania osobno).
  import { fragmentyPodswietlenia } from './narzedzia'

  let { tekst, zapytanie = '' }: { tekst: string; zapytanie?: string } = $props()

  const fragmenty = $derived(fragmentyPodswietlenia(tekst, zapytanie))
</script>

{#each fragmenty as f, i (i)}{#if f.trafienie}<mark>{f.tekst}</mark>{:else}{f.tekst}{/if}{/each}

<style>
  mark {
    background: var(--kolor-tlo-ostrz);
    color: var(--kolor-zolty);
    border-radius: var(--promien-kropka);
  }
</style>
