<script lang="ts">
  // Okno „Klient FTP (FTPS)”: adres, port, login do skopiowania, instrukcja dla klientów i odcisk certyfikatu.
  import type { Adres, OpcjaProtokolu } from '../../api/typy'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import KodQR from '../wspolne/KodQR.svelte'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import { kopiuj } from './narzedzia'

  let {
    otwarte = $bindable(false),
    opcja,
    adresy,
    odcisk,
  }: { otwarte: boolean; opcja: OpcjaProtokolu | null; adresy: Adres[]; odcisk: string } = $props()

  // ftps://login@host:port/sciezka
  const czesci = $derived.by(() => {
    try {
      const u = new URL(opcja?.adres ?? '')
      return {
        host: u.hostname,
        port: u.port || String(opcja?.port ?? ''),
        login: decodeURIComponent(u.username),
        sciezka: decodeURIComponent(u.pathname),
      }
    } catch {
      return { host: '', port: String(opcja?.port ?? ''), login: '', sciezka: '' }
    }
  })

  const RODZAJE: Record<Adres['rodzaj'], string> = { lan: 'LAN', wireguard: 'WireGuard', publiczny: 'Publiczny', petla: 'Lokalny' }

  async function kopiujTekst(tekst: string, co: string) {
    if (await kopiuj(tekst)) powiadom(`Skopiowano ${co}.`, 'ok')
    else powiadom('Nie udało się skopiować. Zaznacz tekst ręcznie.', 'blad')
  }
</script>

<Okno tytul="Pobierz przez klienta FTP (FTPS)" bind:otwarte>
  <div class="tresc">
    <div class="adres">
      <code class="mono skroc" title={opcja?.adres}>{opcja?.adres}</code>
      <Przycisk rodzaj="glowny" ikona="kopiuj" onclick={() => kopiujTekst(opcja?.adres ?? '', 'adres')}>Kopiuj</Przycisk>
    </div>

    <dl class="dane">
      <dt class="etykieta">Protokół</dt>
      <dd>FTP z jawnym TLS (FTPES)</dd>
      <dt class="etykieta">Serwer</dt>
      <dd class="mono">{czesci.host}</dd>
      <dt class="etykieta">Port</dt>
      <dd class="mono">{czesci.port}</dd>
      <dt class="etykieta">Login</dt>
      <dd class="mono">{czesci.login}</dd>
      <dt class="etykieta">Ścieżka</dt>
      <dd class="mono skroc">{czesci.sciezka}</dd>
    </dl>

    <div class="sekcja">
      <span class="etykieta">Jak połączyć</span>
      <ul class="instrukcja">
        <li><b>FileZilla</b>: Plik, Menedżer stron, Nowa strona. Protokół FTP, szyfrowanie „Wymagaj jawnego FTP przez TLS”,
          wpisz serwer, port i login, tryb logowania „Pytaj o hasło”.</li>
        <li><b>Total Commander</b>: Sieć, Połącz z serwerem FTP, Nowe połączenie. Serwer w postaci <code>host:port</code>,
          zaznacz „SSL/TLS” i „Użyj trybu pasywnego”.</li>
        <li><b>WinSCP</b>: Nowa sesja, protokół FTP, szyfrowanie „Jawne szyfrowanie TLS/SSL”, wpisz serwer, port i login.</li>
        <li>Przy pierwszym połączeniu klient pokaże certyfikat. Porównaj odcisk z poniższym i zaakceptuj go.</li>
      </ul>
    </div>

    {#if opcja?.adres}
      <div class="qr">
        <KodQR tekst={opcja.adres} rozmiar="s" etykieta="Kod QR adresu FTPS" />
        <span class="opis">Adres dla aplikacji FTP na telefonie (np. Solid Explorer, FE File Explorer).</span>
      </div>
    {/if}

    {#if odcisk}
      <div class="sekcja">
        <span class="etykieta">Odcisk certyfikatu (SHA-256)</span>
        <div class="adres">
          <code class="mono odcisk">{odcisk}</code>
          <Przycisk rodzaj="duch" ikona="kopiuj" onclick={() => kopiujTekst(odcisk, 'odcisk')}>Kopiuj</Przycisk>
        </div>
      </div>
    {/if}

    {#if adresy.length > 1}
      <div class="sekcja">
        <span class="etykieta">Inne adresy serwera</span>
        <ul class="adresy">
          {#each adresy as a (a.interfejs + a.ip)}
            <li>
              <Pigulka stan={a.rodzaj === 'publiczny' ? 'ostrz' : 'info'}>{RODZAJE[a.rodzaj]}</Pigulka>
              <code class="mono">{a.ip}</code>
              <span class="opis skroc">{a.interfejs}</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>
</Okno>

<style>
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .adres {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
  }
  .adres code {
    flex: 1;
    padding: var(--odstep-s) var(--odstep-m);
    background: var(--kolor-mono);
    border: var(--ramka);
    border-radius: var(--promien-segment);
  }
  .odcisk {
    word-break: break-all;
    white-space: normal;
    font-size: var(--czcionka-drobny);
  }
  .dane {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: var(--odstep-xs) var(--odstep-l);
    margin: 0;
    align-items: baseline;
  }
  .dane dd {
    margin: 0;
  }
  .sekcja {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
  }
  .qr {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
  }
  .instrukcja {
    margin: 0;
    padding-left: var(--odstep-sekcja);
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .instrukcja b {
    color: var(--kolor-tekst);
  }
  .adresy {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
  }
  .adresy li {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
  }
</style>
