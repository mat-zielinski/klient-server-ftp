<script lang="ts">
  // Przełącznik „switch” stacji (UKLAD.suwak_*): tor z gałką, etykieta obok, opcjonalny opis pod spodem.
  interface Props {
    etykieta: string
    value: boolean
    opis?: string
    wylaczony?: boolean
    onchange?: (wartosc: boolean) => void
  }

  let { etykieta, value = $bindable(), opis, wylaczony = false, onchange }: Props = $props()

  function przelacz() {
    if (wylaczony) return
    value = !value
    onchange?.(value)
  }
</script>

<div class="przelacznik" class:wylaczony>
  <button
    type="button"
    role="switch"
    class="tor"
    class:wl={value}
    aria-checked={value}
    aria-label={etykieta}
    disabled={wylaczony}
    onclick={przelacz}
  >
    <span class="galka"></span>
  </button>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <span class="napisy" onclick={przelacz}>
    <span class="etykieta-p">{etykieta}</span>
    {#if opis}<span class="opis">{opis}</span>{/if}
  </span>
</div>

<style>
  .przelacznik {
    display: flex;
    align-items: flex-start;
    gap: var(--odstep-m);
  }
  .wylaczony {
    opacity: var(--uklad-suwak-przygaszenie);
  }
  .tor {
    position: relative;
    flex: none;
    width: var(--uklad-suwak-w);
    height: var(--uklad-suwak-h);
    margin-top: var(--odstep-ramka);
    padding: 0;
    border: 0;
    border-radius: var(--uklad-suwak-h);
    background: var(--kolor-przycisk);
    transition: background var(--czas-szybki);
  }
  .tor.wl {
    background: var(--kolor-niebieski-tlo);
  }
  .galka {
    position: absolute;
    top: var(--uklad-suwak-margines);
    left: var(--uklad-suwak-margines);
    width: calc(var(--uklad-suwak-h) - 2 * var(--uklad-suwak-margines));
    height: calc(var(--uklad-suwak-h) - 2 * var(--uklad-suwak-margines));
    border-radius: var(--promien-kolo);
    background: var(--kolor-na-ciemnym);
    transition: transform var(--czas-szybki);
  }
  .wl .galka {
    transform: translateX(calc(var(--uklad-suwak-w) - var(--uklad-suwak-h)));
  }
  .napisy {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    cursor: pointer;
    min-width: 0;
  }
  .wylaczony .napisy {
    cursor: default;
  }
  .etykieta-p {
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
  }
</style>
