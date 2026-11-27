<script lang="ts">
  // Pole tekstowe (QLineEdit stacji): etykieta nad polem, opcjonalna ikona, podpowiedź lub błąd pod polem.
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  interface Props {
    etykieta: string
    value: string | number
    typ?: 'text' | 'password' | 'number' | 'email' | 'search' | 'url' | 'datetime-local' | 'date'
    podpowiedz?: string
    blad?: string
    opis?: string
    ikona?: NazwaIkony
    wylaczone?: boolean
    wymagane?: boolean
    mono?: boolean
    min?: number
    max?: number
    autocomplete?: HTMLInputElement['autocomplete']
    /** Etykieta tylko dla czytnika ekranu (np. pole szukania w pasku). */
    ukryjEtykiete?: boolean
    autofocus?: boolean
    oninput?: (e: Event) => void
    onkeydown?: (e: KeyboardEvent) => void
  }

  let {
    etykieta,
    value = $bindable(),
    typ = 'text',
    podpowiedz,
    blad,
    opis,
    ikona,
    wylaczone = false,
    wymagane = false,
    mono = false,
    min,
    max,
    autocomplete,
    ukryjEtykiete = false,
    autofocus = false,
    oninput,
    onkeydown,
  }: Props = $props()

  const id = $props.id()

  function ustawFokus(el: HTMLInputElement) {
    if (autofocus) el.focus()
  }
</script>

<label class="pole" class:z-bledem={!!blad} for={id}>
  <span class="etykieta-pola" class:tylko-czytnik={ukryjEtykiete}>{etykieta}</span>
  <span class="ramka" class:wylaczone>
    {#if ikona}<span class="ikona"><Ikona nazwa={ikona} rozmiar="s" /></span>{/if}
    <input
      {id}
      bind:value
      type={typ}
      placeholder={podpowiedz}
      disabled={wylaczone}
      required={wymagane}
      class:mono
      {min}
      {max}
      {autocomplete}
      aria-invalid={blad ? 'true' : undefined}
      {oninput}
      {onkeydown}
      use:ustawFokus
    />
  </span>
  {#if blad}
    <span class="blad">{blad}</span>
  {:else if opis}
    <span class="opis">{opis}</span>
  {/if}
</label>

<style>
  .pole {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    min-width: 0;
  }
  .etykieta-pola {
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    color: var(--kolor-opis);
  }
  .ramka {
    display: flex;
    align-items: center;
    gap: var(--odstep-s);
    height: var(--wysokosc-pole);
    padding: 0 var(--odstep-przycisk-maly);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    transition: border-color var(--czas-szybki);
  }
  .ramka:focus-within {
    border-color: var(--kolor-niebieski);
  }
  .z-bledem .ramka {
    border-color: var(--kolor-czerwony);
  }
  .wylaczone {
    opacity: var(--uklad-suwak-przygaszenie);
  }
  .ikona {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    outline: 0;
    background: transparent;
    font-size: var(--czcionka-maly);
  }
  input::placeholder {
    color: var(--kolor-opis);
  }
  .mono {
    font-family: var(--czcionka-mono);
  }
  .blad {
    font-size: var(--czcionka-drobny);
    color: var(--kolor-czerwony);
  }
</style>
