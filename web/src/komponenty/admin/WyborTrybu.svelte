<script lang="ts">
  // Wybór trybu sieci (Kreator krok 3, Ustawienia → Sieć): dwie duże karty-opcje VPN i Publiczny z opisem,
  // VPN ze znaczkiem „Zalecany”, Publiczny z ostrzeżeniem po wybraniu. Działa jak grupa radiowa (strzałki, Tab).
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Komunikat from './Komunikat.svelte'
  import { TRYBY_SIECI } from './stale'
  import type { TrybSieci } from './typy'

  interface Props {
    value: TrybSieci
    onchange?: (t: TrybSieci) => void
  }

  let { value = $bindable(), onchange }: Props = $props()
  const nazwaGrupy = $props.id()

  function wybierz(t: TrybSieci) {
    value = t
    onchange?.(t)
  }
</script>

<div class="tryby" role="radiogroup" aria-label="Tryb sieci">
  <label class="opcja" class:wybrana={value === 'vpn'}>
    <input type="radio" name={nazwaGrupy} value="vpn" checked={value === 'vpn'} onchange={() => wybierz('vpn')} />
    <span class="ikona"><Ikona nazwa="tarcza" rozmiar="l" /></span>
    <span class="tekst">
      <span class="tytul">{TRYBY_SIECI.vpn.nazwa} <Pigulka stan="ok">{TRYBY_SIECI.vpn.znaczek}</Pigulka></span>
      <span class="opis">{TRYBY_SIECI.vpn.opis}</span>
    </span>
    <span class="znacznik" aria-hidden="true"></span>
  </label>
  <label class="opcja" class:wybrana={value === 'publiczny'}>
    <input
      type="radio"
      name={nazwaGrupy}
      value="publiczny"
      checked={value === 'publiczny'}
      onchange={() => wybierz('publiczny')}
    />
    <span class="ikona"><Ikona nazwa="globus" rozmiar="l" /></span>
    <span class="tekst">
      <span class="tytul">{TRYBY_SIECI.publiczny.nazwa} <Pigulka stan="ostrz">Wymuszone HTTPS</Pigulka></span>
      <span class="opis">{TRYBY_SIECI.publiczny.opis}</span>
    </span>
    <span class="znacznik" aria-hidden="true"></span>
  </label>
  {#if value === 'publiczny'}
    <Komunikat rodzaj="ostrz">{TRYBY_SIECI.publiczny.ostrzezenie}</Komunikat>
  {/if}
</div>

<style>
  .tryby {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-m);
  }
  .opcja {
    display: flex;
    align-items: flex-start;
    gap: var(--odstep-l);
    padding: var(--odstep-xl);
    border: var(--ramka);
    border-radius: var(--promien-kafel);
    background: var(--kolor-karta);
    cursor: pointer;
    transition: border-color var(--czas-szybki), background var(--czas-szybki);
  }
  .opcja:hover {
    background: var(--kolor-przycisk);
  }
  .opcja.wybrana {
    border-color: var(--kolor-niebieski);
    background: var(--kolor-aktywny);
  }
  .opcja:focus-within {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
  }
  input {
    position: absolute;
    opacity: 0;
    pointer-events: none;
  }
  .ikona {
    color: var(--kolor-niebieski);
    display: inline-flex;
    padding-top: var(--odstep-pozycje);
  }
  .tekst {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    min-width: 0;
  }
  .tytul {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--odstep-m);
    font-weight: var(--czcionka-polgruby);
  }
  .tekst {
    flex: 1;
  }
  /* kółko wyboru po prawej (jak radio): wybrane = pełne obramowanie akcentem z kropką */
  .znacznik {
    flex: none;
    width: var(--uklad-ikona-l);
    height: var(--uklad-ikona-l);
    margin-top: var(--odstep-pozycje);
    border-radius: var(--promien-kolo);
    border: var(--odstep-pozycje) solid var(--kolor-siatka);
    transition: border var(--czas-szybki), box-shadow var(--czas-szybki);
  }
  .wybrana .znacznik {
    border: var(--odstep-kafelki) solid var(--kolor-niebieski);
    box-shadow: 0 0 0 var(--odstep-xs) var(--kolor-fokus-pola);
  }
</style>
