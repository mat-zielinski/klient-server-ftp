<script lang="ts" module>
  export interface OpcjaWyboru {
    wartosc: string
    etykieta: string
    wylaczona?: boolean
  }
</script>

<script lang="ts">
  // Lista rozwijana (QComboBox stacji): natywny <select> w ramce pola, własna strzałka z ikon.
  import Ikona from './Ikona.svelte'

  interface Props {
    etykieta: string
    value: string
    opcje: (OpcjaWyboru | string)[]
    opis?: string
    wylaczony?: boolean
    ukryjEtykiete?: boolean
    onchange?: (wartosc: string) => void
  }

  let {
    etykieta,
    value = $bindable(),
    opcje,
    opis,
    wylaczony = false,
    ukryjEtykiete = false,
    onchange,
  }: Props = $props()

  const id = $props.id()
  const lista = $derived(opcje.map((o) => (typeof o === 'string' ? { wartosc: o, etykieta: o } : o)))
</script>

<label class="wybor" for={id}>
  <span class="etykieta-pola" class:tylko-czytnik={ukryjEtykiete}>{etykieta}</span>
  <span class="ramka" class:wylaczony>
    <select {id} bind:value disabled={wylaczony} onchange={() => onchange?.(value)}>
      {#each lista as o (o.wartosc)}
        <option value={o.wartosc} disabled={o.wylaczona}>{o.etykieta}</option>
      {/each}
    </select>
    <span class="strzalka"><Ikona nazwa="w_dol" rozmiar="s" /></span>
  </span>
  {#if opis}<span class="opis">{opis}</span>{/if}
</label>

<style>
  .wybor {
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
    position: relative;
    display: flex;
    align-items: center;
    height: var(--wysokosc-pole);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .ramka:focus-within {
    border-color: var(--kolor-niebieski);
  }
  .wylaczony {
    opacity: var(--uklad-suwak-przygaszenie);
  }
  select {
    appearance: none;
    width: 100%;
    height: 100%;
    padding: 0 calc(var(--uklad-ikona) + var(--odstep-l)) 0 var(--odstep-przycisk-maly);
    border: 0;
    outline: 0;
    background: transparent;
    font-size: var(--czcionka-maly);
    cursor: pointer;
  }
  option {
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
  }
  .strzalka {
    position: absolute;
    right: var(--odstep-m);
    display: inline-flex;
    color: var(--kolor-opis);
    pointer-events: none;
  }
</style>
