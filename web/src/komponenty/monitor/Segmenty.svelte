<script lang="ts">
  // Przełącznik segmentowy (jak segmenty w stacji i UniFi): kilka przycisków w ramce, zaznaczony wyróżniony.
  interface Props {
    opcje: { id: string; etykieta: string }[]
    value: string
    etykieta: string
  }
  let { opcje, value = $bindable(), etykieta }: Props = $props()
</script>

<div class="segmenty" role="radiogroup" aria-label={etykieta}>
  {#each opcje as o (o.id)}
    <button type="button" role="radio" aria-checked={value === o.id} class:wybrany={value === o.id}
      onclick={() => (value = o.id)}>{o.etykieta}</button>
  {/each}
</div>

<style>
  .segmenty {
    display: inline-flex;
    gap: var(--odstep-pozycje);
    padding: var(--odstep-pozycje);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  button {
    height: var(--wysokosc-segment);
    padding: 0 var(--odstep-4);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    font: inherit;
    font-size: var(--czcionka-maly);
    cursor: pointer;
    transition: background var(--czas-szybki), color var(--czas-szybki);
  }
  button:hover {
    color: var(--kolor-tekst);
    background: var(--kolor-przycisk-hover);
  }
  button.wybrany {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
    font-weight: var(--czcionka-polgruby);
  }
  button:focus-visible {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
  }
</style>
