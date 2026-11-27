<script lang="ts">
  // Ostatni krok kreatora: karta podsumowania (ikona, etykieta, wartość, „Zmień” wracające do kroku) i komunikat
  // o trybie sieci (ostrzeżenie dla trybu publicznego, wskazówka dla VPN).
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Komunikat from './Komunikat.svelte'
  import { TRYBY_SIECI } from './stale'
  import type { TrybSieci } from './typy'

  interface Props {
    login: string
    nazwa: string
    sciezka: string
    tylkoOdczyt: boolean
    tryb: TrybSieci
    /** Powrót do kroku (indeks od 0). */
    onzmien: (krok: number) => void
  }

  let { login, nazwa, sciezka, tylkoOdczyt, tryb, onzmien }: Props = $props()

  const wiersze = $derived<{ ikona: NazwaIkony; etykieta: string; wartosc: string; krok: number; mono?: boolean }[]>([
    { ikona: 'uzytkownik', etykieta: 'Administrator', wartosc: login, krok: 0 },
    { ikona: 'folder', etykieta: 'Udział', wartosc: nazwa, krok: 1 },
    { ikona: 'dysk', etykieta: 'Folder na dysku', wartosc: sciezka, krok: 1, mono: true },
    { ikona: tryb === 'vpn' ? 'tarcza' : 'globus', etykieta: 'Tryb sieci', wartosc: TRYBY_SIECI[tryb].nazwa, krok: 2 },
  ])
</script>

<ul class="podsumowanie">
  {#each wiersze as w (w.etykieta)}
    <li>
      <span class="ikona"><Ikona nazwa={w.ikona} /></span>
      <span class="tekst">
        <span class="etykieta-wiersza">{w.etykieta}</span>
        <span class="wartosc skroc" class:mono={w.mono} title={w.wartosc}>
          {w.wartosc}
          {#if w.etykieta === 'Udział'}
            <span class="obok"><Pigulka stan={tylkoOdczyt ? 'info' : 'ok'}>{tylkoOdczyt ? 'Tylko do odczytu' : 'Odczyt i zapis'}</Pigulka></span>
          {/if}
        </span>
      </span>
      <button type="button" class="zmien" onclick={() => onzmien(w.krok)}>Zmień</button>
    </li>
  {/each}
</ul>

{#if tryb === 'publiczny'}
  <Komunikat rodzaj="ostrz">{TRYBY_SIECI.publiczny.ostrzezenie}</Komunikat>
{:else}
  <Komunikat rodzaj="info">
    Klienci łączą się najpierw tunelem WireGuard, potem otwierają adres LAN tego komputera w przeglądarce.
  </Komunikat>
{/if}

<style>
  .podsumowanie {
    list-style: none;
    margin: 0;
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-kafel);
    background: var(--kolor-karta);
    overflow: hidden;
  }
  li {
    display: flex;
    align-items: center;
    gap: var(--odstep-xl);
    padding: var(--odstep-xl) var(--odstep-sekcja);
  }
  li + li {
    border-top: var(--ramka);
  }
  .ikona {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-punkt-marki);
    height: var(--wysokosc-punkt-marki);
    border-radius: var(--promien-kafel);
    background: var(--kolor-tlo-info);
    color: var(--kolor-niebieski);
  }
  .tekst {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
  }
  .etykieta-wiersza {
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
  }
  .wartosc {
    font-weight: var(--czcionka-polgruby);
  }
  .obok {
    margin-left: var(--odstep-m);
  }
  .mono {
    font-family: var(--czcionka-mono);
  }
  .zmien {
    flex: none;
    padding: 0;
    border: 0;
    background: none;
    color: var(--kolor-niebieski);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    cursor: pointer;
  }
  .zmien:hover {
    text-decoration: underline;
  }
</style>
