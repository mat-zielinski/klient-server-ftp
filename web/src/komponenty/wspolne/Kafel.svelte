<script lang="ts">
  // Kafel liczby jak w stacji (kafel_liczby.py): etykieta WIELKIMI nad liczbą mono, opcjonalny opis i ikona.
  // `stan` barwi liczbę (ok = zielony, info = niebieski, blad = czerwony, ostrz = żółty, wyl = opis).
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  interface Props {
    etykieta: string
    wartosc: string | number
    opis?: string
    ikona?: NazwaIkony
    stan?: 'ok' | 'info' | 'blad' | 'ostrz' | 'wyl'
  }

  let { etykieta, wartosc, opis, ikona, stan }: Props = $props()
</script>

<div class="kafel">
  <div class="gora">
    <span class="etykieta skroc">{etykieta}</span>
    {#if ikona}<span class="ikona"><Ikona nazwa={ikona} rozmiar="s" /></span>{/if}
  </div>
  <div class="wartosc skroc {stan ?? ''}" title={String(wartosc)}>{wartosc}</div>
  {#if opis}<div class="opis skroc" title={opis}>{opis}</div>{/if}
</div>

<style>
  .kafel {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    min-width: 0;
    padding: var(--odstep-l) var(--odstep-xl);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
  }
  .gora {
    display: flex;
    align-items: center;
    gap: var(--odstep-s);
  }
  .ikona {
    margin-left: auto;
    display: inline-flex;
    color: var(--kolor-opis);
  }
  .wartosc {
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-kpi);
    font-weight: var(--czcionka-gruby);
    line-height: 1.15;
  }
  .ok {
    color: var(--kolor-zielony);
  }
  .info {
    color: var(--kolor-niebieski);
  }
  .blad {
    color: var(--kolor-czerwony);
  }
  .ostrz {
    color: var(--kolor-zolty);
  }
  .wyl {
    color: var(--kolor-opis);
  }
</style>
