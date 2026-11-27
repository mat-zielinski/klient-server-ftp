<script lang="ts">
  // Zasłona nad listą podczas przeciągania plików: dokąd trafią albo informacja o braku prawa zapisu.
  import Ikona from '../wspolne/Ikona.svelte'

  let { zapis, sciezka }: { zapis: boolean; sciezka: string } = $props()
</script>

<div class="zaslona" class:zablokowane={!zapis}>
  <Ikona nazwa={zapis ? 'wgraj' : 'klodka'} rozmiar="l" />
  <span>{zapis ? `Upuść, aby wgrać do ${sciezka}` : 'Brak prawa zapisu w tym udziale'}</span>
</div>

<style>
  .zaslona {
    position: absolute;
    inset: var(--odstep-m);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--odstep-m);
    background: var(--kolor-tlo-info);
    color: var(--kolor-niebieski);
    border: var(--odstep-pozycje) dashed var(--kolor-niebieski);
    border-radius: var(--promien-karta);
    font-weight: var(--czcionka-polgruby);
    pointer-events: none;
  }
  .zaslona.zablokowane {
    background: var(--kolor-tlo-ostrz);
    color: var(--kolor-zolty);
    border-color: var(--kolor-zolty);
  }
  .zaslona :global(svg) {
    width: var(--uklad-ikona-pusto);
    height: var(--uklad-ikona-pusto);
  }
</style>
