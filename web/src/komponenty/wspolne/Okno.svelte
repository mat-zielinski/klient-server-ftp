<script lang="ts">
  // Okno modalne na <dialog>: tytuł, przycisk zamknięcia, treść i opcjonalna stopka (przyciski).
  // Zamyka: Esc, klik w zasłonę, krzyżyk; `otwarte` jest bindable.
  import type { Snippet } from 'svelte'
  import Ikona from './Ikona.svelte'

  interface Props {
    tytul: string
    otwarte: boolean
    children?: Snippet
    stopka?: Snippet
    /** Szerokie okno (podgląd plików, tabele). */
    szerokie?: boolean
    naZamknij?: () => void
  }

  let { tytul, otwarte = $bindable(), children, stopka, szerokie = false, naZamknij }: Props = $props()

  let dialog: HTMLDialogElement | undefined = $state()

  $effect(() => {
    if (!dialog) return
    if (otwarte && !dialog.open) dialog.showModal()
    else if (!otwarte && dialog.open) dialog.close()
  })

  function zamknij() {
    otwarte = false
  }

  function poZamknieciu() {
    if (otwarte) otwarte = false
    naZamknij?.()
  }

  function klikZaslony(e: MouseEvent) {
    if (e.target === dialog) zamknij()
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog bind:this={dialog} class="okno" class:szerokie onclose={poZamknieciu} onclick={klikZaslony} aria-label={tytul}>
  {#if otwarte}
    <div class="ramka">
      <header class="naglowek">
        <h3 class="skroc">{tytul}</h3>
        <button type="button" class="zamknij" onclick={zamknij} aria-label="Zamknij" title="Zamknij">
          <Ikona nazwa="zamknij" />
        </button>
      </header>
      <div class="tresc">{@render children?.()}</div>
      {#if stopka}<footer class="stopka">{@render stopka()}</footer>{/if}
    </div>
  {/if}
</dialog>

<style>
  .okno {
    width: min(var(--uklad-okno-w), calc(100vw - 2 * var(--odstep-sekcja)));
    max-height: var(--uklad-okno-maks-h);
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-karta);
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
    box-shadow: 0 var(--odstep-m) calc(var(--odstep-sekcja) * 2) var(--kolor-cien);
    overflow: hidden;
  }
  .szerokie {
    width: min(calc(var(--uklad-okno-w) * 2), calc(100vw - 2 * var(--odstep-sekcja)));
  }
  .okno::backdrop {
    background: var(--kolor-zaslona);
  }
  .ramka {
    display: flex;
    flex-direction: column;
    max-height: var(--uklad-okno-maks-h);
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    padding: var(--odstep-xl) var(--odstep-xxl);
    border-bottom: var(--ramka);
  }
  h3 {
    flex: 1;
  }
  .zamknij {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-przycisk-maly);
    height: var(--wysokosc-przycisk-maly);
    border: 0;
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-opis);
  }
  .zamknij:hover {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
    padding: var(--odstep-xxl);
    overflow: auto;
  }
  .stopka {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: var(--odstep-m);
    padding: var(--odstep-xl) var(--odstep-xxl);
    border-top: var(--ramka);
    background: var(--kolor-karta);
  }
</style>
