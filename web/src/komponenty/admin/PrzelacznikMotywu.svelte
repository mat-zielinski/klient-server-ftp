<script lang="ts">
  // Przełącznik motywu na ekranach wejścia: segmenty Auto / Jasny / Ciemny z ikonami (grupa radiowa).
  // Działa od razu (stan/motyw: wybór zapamiętany w przeglądarce), także przed zalogowaniem.
  import type { NazwaIkony } from '../../ikony/ikony'
  import { motyw, NAZWY_MOTYWOW, type WyborMotywu } from '../../stan/motyw.svelte'
  import Ikona from '../wspolne/Ikona.svelte'

  interface Props {
    /** Same ikony (telefon); podpis zostaje w dymku i dla czytnika ekranu. */
    zwarty?: boolean
  }

  let { zwarty = false }: Props = $props()

  const OPCJE: { wartosc: WyborMotywu; etykieta: string; ikona: NazwaIkony }[] = [
    { wartosc: 'auto', etykieta: 'Auto', ikona: 'motyw_auto' },
    { wartosc: 'jasny', etykieta: 'Jasny', ikona: 'slonce' },
    { wartosc: 'ciemny', etykieta: 'Ciemny', ikona: 'ksiezyc' },
  ]

  function strzalki(e: KeyboardEvent) {
    const krok = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? -1 : 0
    if (!krok) return
    e.preventDefault()
    const i = OPCJE.findIndex((o) => o.wartosc === motyw.wybor)
    const nowa = OPCJE[(i + krok + OPCJE.length) % OPCJE.length]
    motyw.ustaw(nowa.wartosc)
    const grupa = e.currentTarget as HTMLElement
    grupa.querySelector<HTMLButtonElement>(`[data-wartosc="${nowa.wartosc}"]`)?.focus()
  }
</script>

<div class="motywy" role="radiogroup" aria-label="Motyw" tabindex="-1" onkeydown={strzalki}>
  {#each OPCJE as o (o.wartosc)}
    <button
      type="button"
      role="radio"
      aria-checked={motyw.wybor === o.wartosc}
      tabindex={motyw.wybor === o.wartosc ? 0 : -1}
      data-wartosc={o.wartosc}
      class:wybrany={motyw.wybor === o.wartosc}
      title={NAZWY_MOTYWOW[o.wartosc]}
      onclick={() => motyw.ustaw(o.wartosc)}
    >
      <Ikona nazwa={o.ikona} rozmiar="s" />
      <span class="podpis" class:tylko-czytnik={zwarty}>{o.etykieta}</span>
    </button>
  {/each}
</div>

<style>
  .motywy {
    display: inline-flex;
    gap: var(--odstep-pozycje);
    padding: var(--odstep-wnetrze-segmentu);
    border: var(--odstep-ramka) solid var(--kolor-ramka-wejscia);
    border-radius: var(--promien-przycisk);
    background: var(--kolor-karta-wejscia);
    backdrop-filter: blur(var(--odstep-m));
    box-shadow: var(--cien-karta-wejscia);
  }
  button {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
    height: var(--wysokosc-przycisk-maly);
    padding: 0 var(--odstep-m);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    cursor: pointer;
    transition: background var(--czas-szybki), color var(--czas-szybki);
  }
  button:hover {
    color: var(--kolor-tekst);
  }
  button.wybrany {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  button:focus-visible {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
  }
</style>
