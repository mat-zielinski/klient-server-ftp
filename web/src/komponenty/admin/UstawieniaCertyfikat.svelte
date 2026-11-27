<script lang="ts">
  // Ustawienia → Certyfikat HTTPS i FTPS: własny (samopodpisany, tworzy serwer), z pliku (ścieżki certyfikatu
  // i klucza PEM) albo Let's Encrypt (domena). Odcisk bieżącego certyfikatu z GET /api/siec do porównania
  // przez klienta (przeglądarka, FileZilla).
  import type { Konfiguracja } from '../../api/typy'
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Komunikat from './Komunikat.svelte'
  import { kopiuj, poleWymagaRestartu, zrodloCertyfikatu } from './narzedzia'
  import type { ZrodloCertyfikatu } from './typy'
  import { powiadom } from '../../stan/powiadomienia.svelte'

  interface Props {
    konfig: Konfiguracja
    zapisana: Konfiguracja
    odcisk: string
  }

  let { konfig = $bindable(), zapisana, odcisk }: Props = $props()

  const ZRODLA: { id: ZrodloCertyfikatu; nazwa: string; opis: string; ikona: NazwaIkony }[] = [
    { id: 'wlasny', nazwa: 'Własny', opis: 'Samopodpisany, tworzy go serwer. Przeglądarka pokaże ostrzeżenie.', ikona: 'tarcza_pusta' },
    { id: 'plik', nazwa: 'Z pliku', opis: 'Certyfikat i klucz PEM, np. z firmowego urzędu.', ikona: 'plik' },
    { id: 'letsencrypt', nazwa: 'Let\u2019s Encrypt', opis: 'Darmowy, zaufany. Wymaga domeny i portu 80 lub 443 z internetu.', ikona: 'globus' },
  ]

  let zrodlo = $state<ZrodloCertyfikatu>(zrodloCertyfikatu(konfig))
  const restart = $derived(
    poleWymagaRestartu(zapisana, konfig, 'certyfikat') ||
      poleWymagaRestartu(zapisana, konfig, 'klucz') ||
      poleWymagaRestartu(zapisana, konfig, 'domena_lets_encrypt'),
  )

  function wybierz(z: ZrodloCertyfikatu) {
    zrodlo = z
    if (z !== 'letsencrypt') konfig.domena_lets_encrypt = ''
    if (z !== 'plik') {
      konfig.certyfikat = ''
      konfig.klucz = ''
    }
  }

  async function kopiujOdcisk() {
    const ok = await kopiuj(odcisk)
    powiadom(ok ? 'Skopiowano odcisk certyfikatu.' : 'Nie udało się skopiować.', ok ? 'ok' : 'blad')
  }
</script>

<div class="certyfikat">
  <div class="naglowek">
    <h4>Źródło certyfikatu</h4>
    {#if restart}<Pigulka stan="ostrz">Wymaga restartu</Pigulka>{/if}
  </div>
  <div class="zrodla" role="radiogroup" aria-label="Źródło certyfikatu">
    {#each ZRODLA as z (z.id)}
      <button type="button" role="radio" aria-checked={zrodlo === z.id} class="zrodlo" class:wybrane={zrodlo === z.id}
        onclick={() => wybierz(z.id)}>
        <span class="tytul"><Ikona nazwa={z.ikona} rozmiar="s" />{z.nazwa}</span>
        <span class="opis">{z.opis}</span>
      </button>
    {/each}
  </div>

  {#if zrodlo === 'plik'}
    <Pole etykieta="Plik certyfikatu (PEM)" bind:value={konfig.certyfikat} ikona="plik" mono
      podpowiedz="C:\certyfikaty\serwer.crt" opis="Pełna ścieżka na tym komputerze, z łańcuchem pośrednim." />
    <Pole etykieta="Plik klucza prywatnego (PEM)" bind:value={konfig.klucz} ikona="klucz" mono
      podpowiedz="C:\certyfikaty\serwer.key" opis="Klucz bez hasła, pełna ścieżka na tym komputerze." />
  {:else if zrodlo === 'letsencrypt'}
    <Pole etykieta="Domena" bind:value={konfig.domena_lets_encrypt} ikona="globus" mono
      podpowiedz="pliki.example.com" opis="Domena musi wskazywać na publiczny adres routera." />
    {#if konfig.tryb_sieci === 'vpn'}
      <Komunikat rodzaj="ostrz">
        W trybie VPN serwer nie jest widoczny z internetu, więc Let&rsquo;s Encrypt nie potwierdzi domeny.
      </Komunikat>
    {/if}
  {:else if konfig.tryb_sieci === 'publiczny'}
    <Komunikat rodzaj="ostrz">
      W trybie publicznym klienci zobaczą ostrzeżenie przeglądarki. Lepiej użyć Let&rsquo;s Encrypt.
    </Komunikat>
  {/if}

  <div class="odcisk">
    <span class="etykieta">Odcisk bieżącego certyfikatu</span>
    {#if odcisk}
      <div class="wiersz">
        <code class="mono skroc" title={odcisk}>{odcisk}</code>
        <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" tytul="Kopiuj odcisk" onclick={kopiujOdcisk} />
      </div>
      <p class="opis">Klient porównuje go z odciskiem, który pokaże przeglądarka albo FileZilla przy pierwszym połączeniu.</p>
    {:else}
      <p class="opis">Brak (serwer jeszcze nie utworzył certyfikatu albo nie wczytano danych sieci).</p>
    {/if}
  </div>
</div>

<style>
  .certyfikat {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
  }
  .zrodla {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(var(--kolumna-l), 1fr));
    gap: var(--odstep-m);
  }
  .zrodlo {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--odstep-xs);
    padding: var(--odstep-l) var(--odstep-xl);
    text-align: left;
    border: var(--ramka);
    border-radius: var(--promien-kafel);
    background: var(--kolor-karta);
    transition: background var(--czas-szybki), border-color var(--czas-szybki);
  }
  .zrodlo:hover {
    background: var(--kolor-przycisk);
  }
  .zrodlo.wybrane {
    border-color: var(--kolor-niebieski);
    background: var(--kolor-aktywny);
  }
  .tytul {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-s);
    font-weight: var(--czcionka-polgruby);
  }
  .wybrane .tytul {
    color: var(--kolor-niebieski);
  }
  .odcisk {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
    padding-top: var(--odstep-xl);
    border-top: var(--ramka);
  }
  code {
    padding: var(--odstep-xs) var(--odstep-m);
    background: var(--kolor-mono);
    border: var(--ramka);
    border-radius: var(--promien-segment);
    font-size: var(--czcionka-maly);
  }
</style>
