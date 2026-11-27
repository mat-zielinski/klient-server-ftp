<script lang="ts">
  // Lewa kolumna udziałów (jak pozycje paska bocznego stacji): ikona, nazwa, pigułka „Odczyt” przy braku zapisu.
  import type { UdzialInfo } from '../../api/typy'
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'

  let { udzialy, wybrany, wybierz }: { udzialy: UdzialInfo[]; wybrany: string; wybierz: (nazwa: string) => void } =
    $props()
</script>

<nav class="udzialy" aria-label="Udziały">
  <span class="etykieta naglowek">Udziały</span>
  {#each udzialy as u (u.nazwa)}
    <button class="pozycja" class:aktywna={u.nazwa === wybrany} aria-current={u.nazwa === wybrany ? 'page' : undefined}
      onclick={() => wybierz(u.nazwa)} title={u.nazwa}>
      <Ikona nazwa="dysk" rozmiar="s" />
      <span class="skroc nazwa">{u.nazwa}</span>
      {#if u.tylko_odczyt || u.prawo !== 'zapis'}
        <Pigulka stan="wyl">Odczyt</Pigulka>
      {/if}
    </button>
  {/each}
</nav>

<style>
  .udzialy {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    min-width: 0;
  }
  .naglowek {
    padding: var(--odstep-s) var(--odstep-m);
  }
  .pozycja {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    height: var(--wysokosc-pozycja-paska);
    padding: 0 var(--odstep-m);
    border: 0;
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-opis);
    text-align: left;
    position: relative;
  }
  .pozycja:hover {
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
  }
  .pozycja.aktywna {
    background: var(--kolor-aktywny);
    color: var(--kolor-tekst);
  }
  .pozycja.aktywna::before {
    content: '';
    position: absolute;
    left: 0;
    top: var(--odstep-m);
    bottom: var(--odstep-m);
    width: var(--odstep-pozycje);
    border-radius: var(--promien-kropka);
    background: var(--kolor-niebieski);
  }
  .pozycja.aktywna :global(svg) {
    color: var(--kolor-niebieski);
  }
  .nazwa {
    flex: 1;
    font-weight: var(--czcionka-polgruby);
  }
</style>
