<script lang="ts">
  // Edytor praw konta do udziałów: wiersz na udział (nazwa, znaczek „tylko do odczytu”) i segmenty
  // Brak / Odczyt / Zapis. Udział tylko do odczytu nie pozwala wybrać zapisu.
  import type { Prawo, Udzial } from '../../api/typy'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import { OPCJE_PRAWA } from './stale'

  interface Props {
    udzialy: Udzial[]
    value: Record<string, Prawo>
  }

  let { udzialy, value = $bindable() }: Props = $props()

  function ustaw(udzial: string, p: Prawo) {
    value = { ...value, [udzial]: p }
  }
</script>

{#if udzialy.length === 0}
  <Pusto ikona="folder" tytul="Brak udziałów" opis="Dodaj udział w Ustawieniach, potem nadaj do niego prawa." />
{:else}
  <div class="prawa">
    {#each udzialy as u (u.nazwa)}
      {@const biezace = value[u.nazwa] ?? 'brak'}
      <div class="wiersz-prawa">
        <span class="nazwa skroc" title={u.sciezka}>{u.nazwa}</span>
        {#if u.tylko_odczyt}<Pigulka stan="wyl">Tylko odczyt</Pigulka>{/if}
        <div class="segmenty" role="radiogroup" aria-label="Prawo do udziału {u.nazwa}">
          {#each OPCJE_PRAWA as o (o.wartosc)}
            {@const zablokowane = o.wartosc === 'zapis' && u.tylko_odczyt}
            <button
              type="button"
              role="radio"
              aria-checked={biezace === o.wartosc}
              class:wybrany={biezace === o.wartosc}
              disabled={zablokowane}
              title={zablokowane ? 'Udział jest tylko do odczytu' : undefined}
              onclick={() => ustaw(u.nazwa, o.wartosc)}
            >
              {o.etykieta}
            </button>
          {/each}
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .prawa {
    display: flex;
    flex-direction: column;
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .wiersz-prawa {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    padding: var(--odstep-s) var(--odstep-l);
    min-height: var(--wysokosc-wiersz-tabeli);
  }
  .wiersz-prawa + .wiersz-prawa {
    border-top: var(--ramka);
  }
  .nazwa {
    flex: 1 1 auto;
    font-weight: var(--czcionka-polgruby);
  }
  .segmenty {
    flex: none;
    display: inline-flex;
    padding: var(--odstep-wnetrze-segmentu);
    gap: var(--odstep-pozycje);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  button {
    height: var(--wysokosc-segment);
    padding: 0 var(--odstep-l);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  button:hover:not(:disabled) {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  button.wybrany {
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  button:disabled {
    opacity: var(--uklad-suwak-przygaszenie);
  }
</style>
