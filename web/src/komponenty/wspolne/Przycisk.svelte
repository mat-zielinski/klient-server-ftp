<script lang="ts">
  // Przycisk jak w stacji (arkusz.py: _przyciski): zwykły (szary), główny (niebieski), niebezpieczny (czerwony),
  // duch (przezroczysty z ramką). Nieaktywny zachowuje kształt, napis przygaszony.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  interface Props {
    rodzaj?: 'zwykly' | 'glowny' | 'niebezpieczny' | 'duch'
    ikona?: NazwaIkony
    wylaczony?: boolean
    onclick?: (e: MouseEvent) => void
    children?: Snippet
    /** Mały (wysokość 26) lub duży (38). */
    rozmiar?: 'maly' | 'duzy'
    typ?: 'button' | 'submit'
    /** Dymek i etykieta dostępności (ważne dla przycisku z samą ikoną). */
    tytul?: string
  }

  let {
    rodzaj = 'zwykly',
    ikona,
    wylaczony = false,
    onclick,
    children,
    rozmiar,
    typ = 'button',
    tytul,
  }: Props = $props()
</script>

<button
  type={typ}
  class="przycisk {rodzaj} {rozmiar ?? ''}"
  class:sama-ikona={!children}
  disabled={wylaczony}
  title={tytul}
  aria-label={!children ? tytul : undefined}
  {onclick}
>
  {#if ikona}<Ikona nazwa={ikona} rozmiar={rozmiar === 'maly' ? 's' : 'm'} />{/if}
  {#if children}<span class="napis">{@render children()}</span>{/if}
</button>

<style>
  .przycisk {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--odstep-s);
    height: var(--wysokosc-przycisk);
    padding: 0 var(--odstep-xl);
    border: 0;
    border-radius: var(--promien-przycisk);
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    white-space: nowrap;
    transition: background var(--czas-szybki);
    flex: none;
  }
  .napis {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .zwykly:hover:not(:disabled) {
    background: var(--kolor-przycisk-hover);
  }
  .glowny {
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .glowny:hover:not(:disabled) {
    filter: brightness(1.1);
  }
  .niebezpieczny {
    background: var(--kolor-czerwony-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .niebezpieczny:hover:not(:disabled) {
    filter: brightness(1.1);
  }
  .duch {
    background: transparent;
    border: var(--ramka);
  }
  .duch:hover:not(:disabled) {
    background: var(--kolor-przycisk);
  }
  .maly {
    height: var(--wysokosc-przycisk-maly);
    padding: 0 var(--odstep-przycisk-maly);
    font-size: var(--czcionka-drobny);
  }
  .duzy {
    height: var(--wysokosc-przycisk-duzy);
    font-size: var(--czcionka-tekst);
  }
  .sama-ikona {
    width: var(--wysokosc-przycisk);
    padding: 0;
  }
  .sama-ikona.maly {
    width: var(--wysokosc-przycisk-maly);
  }
  .sama-ikona.duzy {
    width: var(--wysokosc-przycisk-duzy);
  }
  /* nieaktywne (motyw.py: WYLACZONY) */
  .zwykly:disabled {
    background: var(--kolor-wylaczony-tlo);
    color: var(--kolor-wylaczony-napis);
  }
  .duch:disabled {
    border-color: var(--kolor-przycisk);
    color: var(--kolor-wylaczony-napis);
  }
  .glowny:disabled {
    background: var(--kolor-wylaczony-glowny);
    color: var(--kolor-wylaczony-napis-glowny);
  }
  .niebezpieczny:disabled {
    background: var(--kolor-wylaczony-blad);
    color: var(--kolor-wylaczony-napis-blad);
  }
</style>
