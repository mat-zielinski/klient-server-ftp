<script lang="ts">
  // Stan pusty: duża przygaszona ikona, tytuł i opis, opcjonalne akcje pod spodem.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  interface Props {
    ikona: NazwaIkony
    tytul: string
    opis?: string
    children?: Snippet
  }

  let { ikona, tytul, opis, children }: Props = $props()
</script>

<div class="pusto">
  <span class="ikona"><Ikona nazwa={ikona} rozmiar="l" /></span>
  <h3>{tytul}</h3>
  {#if opis}<p class="opis">{opis}</p>{/if}
  {#if children}<div class="akcje">{@render children()}</div>{/if}
</div>

<style>
  .pusto {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--odstep-m);
    padding: var(--odstep-sekcja);
    text-align: center;
    color: var(--kolor-tekst);
    min-height: calc(var(--uklad-ikona-pusto) * 4);
  }
  .ikona {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--uklad-ikona-pusto);
    height: var(--uklad-ikona-pusto);
    border-radius: var(--promien-karta);
    background: var(--kolor-karta);
    border: var(--ramka);
    color: var(--kolor-opis);
  }
  .opis {
    max-width: var(--uklad-okno-w);
  }
  .akcje {
    display: flex;
    gap: var(--odstep-m);
    margin-top: var(--odstep-s);
  }
</style>
