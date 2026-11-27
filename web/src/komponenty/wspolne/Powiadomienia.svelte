<script lang="ts">
  // Toasty powiadomień (stan/powiadomienia.svelte.ts) w prawym górnym rogu; klik zamyka.
  import { fly } from 'svelte/transition'
  import { powiadomienia, type RodzajPowiadomienia } from '../../stan/powiadomienia.svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  const IKONA: Record<RodzajPowiadomienia, NazwaIkony> = {
    ok: 'ok_kolo',
    info: 'info',
    blad: 'blad',
    ostrz: 'alert',
  }
</script>

<div class="powiadomienia" aria-live="polite">
  {#each powiadomienia.lista as p (p.id)}
    <button
      type="button"
      class="toast {p.rodzaj}"
      role={p.rodzaj === 'blad' ? 'alert' : 'status'}
      title="Zamknij"
      onclick={() => powiadomienia.zamknij(p.id)}
      transition:fly={{ x: 40, duration: 180 }}
    >
      <span class="ikona"><Ikona nazwa={IKONA[p.rodzaj]} /></span>
      <span class="tekst">{p.tekst}</span>
    </button>
  {/each}
</div>

<style>
  .powiadomienia {
    position: fixed;
    top: var(--odstep-sekcja);
    right: var(--odstep-sekcja);
    z-index: var(--warstwa-toast);
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--odstep-m);
    width: min(var(--uklad-toast-w), calc(100vw - 2 * var(--odstep-sekcja)));
    pointer-events: none;
  }
  .toast {
    pointer-events: auto;
    display: flex;
    align-items: flex-start;
    gap: var(--odstep-m);
    width: 100%;
    padding: var(--odstep-l) var(--odstep-xl);
    border: var(--ramka);
    border-left-width: calc(var(--odstep-ramka) * 3);
    border-radius: var(--promien-przycisk);
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
    box-shadow: 0 var(--odstep-s) var(--odstep-sekcja) var(--kolor-cien);
    font-size: var(--czcionka-maly);
    text-align: left;
  }
  .ikona {
    display: inline-flex;
  }
  .tekst {
    flex: 1;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .ok {
    border-left-color: var(--kolor-zielony);
  }
  .ok .ikona {
    color: var(--kolor-zielony);
  }
  .info {
    border-left-color: var(--kolor-niebieski);
  }
  .info .ikona {
    color: var(--kolor-niebieski);
  }
  .blad {
    border-left-color: var(--kolor-czerwony);
  }
  .blad .ikona {
    color: var(--kolor-czerwony);
  }
  .ostrz {
    border-left-color: var(--kolor-zolty);
  }
  .ostrz .ikona {
    color: var(--kolor-zolty);
  }
</style>
