<script lang="ts">
  // Ramka komunikatu w treści (błąd logowania, blokada, ostrzeżenie trybu publicznego): ikona + tekst na tle stanu.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from '../wspolne/Ikona.svelte'

  interface Props {
    rodzaj: 'ok' | 'info' | 'blad' | 'ostrz'
    children: Snippet
  }

  let { rodzaj, children }: Props = $props()

  const IKONA: Record<Props['rodzaj'], NazwaIkony> = { ok: 'ok_kolo', info: 'info', blad: 'blad', ostrz: 'alert' }
</script>

<div class="komunikat {rodzaj}" role={rodzaj === 'blad' ? 'alert' : 'status'}>
  <span class="ikona"><Ikona nazwa={IKONA[rodzaj]} rozmiar="s" /></span>
  <div class="tresc">{@render children()}</div>
</div>

<style>
  .komunikat {
    display: flex;
    align-items: flex-start;
    gap: var(--odstep-m);
    padding: var(--odstep-m) var(--odstep-l);
    border-radius: var(--promien-przycisk);
    font-size: var(--czcionka-maly);
  }
  .ikona {
    display: inline-flex;
    padding-top: var(--odstep-ramka);
  }
  .tresc {
    min-width: 0;
    color: var(--kolor-tekst);
  }
  .ok {
    background: var(--kolor-tlo-ok);
    color: var(--kolor-zielony);
  }
  .info {
    background: var(--kolor-tlo-info);
    color: var(--kolor-niebieski);
  }
  .blad {
    background: var(--kolor-tlo-blad);
    color: var(--kolor-czerwony);
  }
  .ostrz {
    background: var(--kolor-tlo-ostrz);
    color: var(--kolor-zolty);
  }
</style>
