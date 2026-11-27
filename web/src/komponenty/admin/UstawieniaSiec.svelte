<script lang="ts">
  // Ustawienia → Sieć: tryb (VPN / Publiczny), adres nasłuchu, porty HTTP / HTTPS / FTPS, porty pasywne FTP,
  // FTPS wł./wył., adresy serwera z GET /api/siec, test portów i reguła Zapory Windows (POST /api/siec/zapora,
  // po potwierdzeniu). Pola, których zmiana działa po restarcie, mają pigułkę „Wymaga restartu”.
  import { api, trescBledu } from '../../api/klient'
  import type { Konfiguracja } from '../../api/typy'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przelacznik from '../wspolne/Przelacznik.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Wybor, { type OpcjaWyboru } from '../wspolne/Wybor.svelte'
  import Komunikat from './Komunikat.svelte'
  import OknoPotwierdzenia from './OknoPotwierdzenia.svelte'
  import WyborTrybu from './WyborTrybu.svelte'
  import { bladPortu, poleWymagaRestartu } from './narzedzia'
  import type { InfoSieci } from './typy'
  import { powiadom } from '../../stan/powiadomienia.svelte'

  interface Props {
    konfig: Konfiguracja
    zapisana: Konfiguracja
    siec: InfoSieci | null
    bladSieci: string
    /** Ponowne pobranie GET /api/siec (test portów). */
    naTestuj: () => Promise<void>
  }

  let { konfig = $bindable(), zapisana, siec, bladSieci, naTestuj }: Props = $props()

  const WSZYSTKIE = '0.0.0.0'
  const RODZAJ_ADRESU: Record<string, string> = { lan: 'LAN', wireguard: 'WireGuard', publiczny: 'Publiczny', petla: 'Pętla' }

  let testuje = $state(false)
  let oknoZapory = $state(false)

  const opcjeAdresu = $derived.by((): OpcjaWyboru[] => {
    const lista: OpcjaWyboru[] = [{ wartosc: WSZYSTKIE, etykieta: 'Wszystkie interfejsy (0.0.0.0)' }]
    for (const a of siec?.adresy ?? []) {
      lista.push({ wartosc: a.ip, etykieta: `${a.ip} (${a.interfejs}, ${RODZAJ_ADRESU[a.rodzaj] ?? a.rodzaj})` })
    }
    if (konfig.adres_nasluchu && !lista.some((o) => o.wartosc === konfig.adres_nasluchu)) {
      lista.push({ wartosc: konfig.adres_nasluchu, etykieta: `${konfig.adres_nasluchu} (z konfiguracji)` })
    }
    return lista
  })

  const restart = (pole: keyof Konfiguracja) => poleWymagaRestartu(zapisana, konfig, pole)

  async function testuj() {
    testuje = true
    try {
      await naTestuj()
    } finally {
      testuje = false
    }
  }

  async function dodajRegule() {
    try {
      await api.post('/siec/zapora')
      powiadom('Dodano regułę Zapory Windows dla portów serwera.', 'ok')
      await naTestuj()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
      throw b
    }
  }
</script>

{#snippet znacznik(pole: keyof Konfiguracja)}
  {#if restart(pole)}<Pigulka stan="ostrz">Wymaga restartu</Pigulka>{/if}
{/snippet}

<div class="siec">
  <section class="sekcja">
    <div class="naglowek"><h4>Tryb sieci</h4>{@render znacznik('tryb_sieci')}</div>
    <WyborTrybu bind:value={konfig.tryb_sieci} />
  </section>

  <section class="sekcja">
    <div class="naglowek"><h4>Nasłuch i porty</h4>
      {@render znacznik('adres_nasluchu')}
      {#if restart('port_http') || restart('port_https') || restart('port_ftps') || restart('porty_pasywne_ftp')}
        <Pigulka stan="ostrz">Porty: wymaga restartu</Pigulka>
      {/if}
    </div>
    <Wybor etykieta="Adres nasłuchu" bind:value={konfig.adres_nasluchu} opcje={opcjeAdresu}
      opis={konfig.tryb_sieci === 'vpn'
        ? 'W trybie VPN wybierz adres LAN albo tunelu, żeby serwer nie słuchał na innych interfejsach.'
        : 'Adres, na który router przekierowuje port.'} />
    <div class="porty">
      <Pole etykieta="Port HTTP" typ="number" bind:value={konfig.port_http} mono min={1} max={65535}
        blad={bladPortu(Number(konfig.port_http)) || undefined}
        opis={konfig.tryb_sieci === 'publiczny' ? 'W trybie publicznym tylko przekierowanie na HTTPS.' : undefined} />
      <Pole etykieta="Port HTTPS" typ="number" bind:value={konfig.port_https} mono min={1} max={65535}
        blad={bladPortu(Number(konfig.port_https)) || undefined} />
      <Pole etykieta="Port FTPS" typ="number" bind:value={konfig.port_ftps} mono min={1} max={65535}
        wylaczone={!konfig.ftps_wlaczone} blad={bladPortu(Number(konfig.port_ftps)) || undefined} />
    </div>
    <div class="porty">
      <Pole etykieta="Porty pasywne FTP od" typ="number" bind:value={konfig.porty_pasywne_ftp[0]} mono min={1} max={65535}
        wylaczone={!konfig.ftps_wlaczone} blad={bladPortu(Number(konfig.porty_pasywne_ftp[0])) || undefined} />
      <Pole etykieta="do" typ="number" bind:value={konfig.porty_pasywne_ftp[1]} mono min={1} max={65535}
        wylaczone={!konfig.ftps_wlaczone}
        blad={bladPortu(Number(konfig.porty_pasywne_ftp[1])) ||
          (Number(konfig.porty_pasywne_ftp[1]) < Number(konfig.porty_pasywne_ftp[0]) ? 'Koniec zakresu przed początkiem.' : undefined)} />
    </div>
    <div class="wiersz">
      <Przelacznik etykieta="Serwer FTPS" bind:value={konfig.ftps_wlaczone}
        opis="Dostęp programami FTP (FileZilla, WinSCP, Total Commander), zawsze szyfrowany TLS." />
      {@render znacznik('ftps_wlaczone')}
    </div>
  </section>

  <section class="sekcja">
    <div class="naglowek">
      <h4>Adresy serwera</h4>
      <span class="akcje">
        <Przycisk rozmiar="maly" ikona="odswiez" wylaczony={testuje} onclick={testuj}>
          {testuje ? 'Sprawdzanie…' : 'Test portów'}
        </Przycisk>
        <Przycisk rozmiar="maly" ikona="tarcza" onclick={() => (oknoZapory = true)}>Dodaj regułę Zapory Windows</Przycisk>
      </span>
    </div>
    {#if bladSieci}
      <Komunikat rodzaj="blad">{bladSieci}</Komunikat>
    {:else if siec}
      <ul class="adresy">
        {#each siec.adresy as a (a.ip + a.interfejs)}
          <li>
            <span class="mono">{a.ip}</span>
            <span class="opis skroc">{a.interfejs}</span>
            <Pigulka stan={a.rodzaj === 'wireguard' ? 'ok' : a.rodzaj === 'publiczny' ? 'ostrz' : a.rodzaj === 'lan' ? 'info' : 'wyl'}>
              {RODZAJ_ADRESU[a.rodzaj] ?? a.rodzaj}
            </Pigulka>
          </li>
        {/each}
        {#if siec.publiczny}
          <li>
            <span class="mono">{siec.publiczny}</span>
            <span class="opis skroc">Adres zewnętrzny routera</span>
            <Pigulka stan="ostrz">Internet</Pigulka>
          </li>
        {/if}
      </ul>
      {#if Object.keys(siec.porty_otwarte ?? {}).length}
        <div class="test">
          <span class="etykieta">Wynik testu portów</span>
          <span class="pigulki">
            {#each Object.entries(siec.porty_otwarte) as [port, otwarty] (port)}
              <Pigulka stan={otwarty ? 'ok' : 'blad'} kropka>{port.toUpperCase()}: {otwarty ? 'otwarty' : 'zamknięty'}</Pigulka>
            {/each}
          </span>
        </div>
      {/if}
    {:else}
      <p class="opis">Wczytywanie adresów…</p>
    {/if}
  </section>
</div>

<OknoPotwierdzenia tytul="Dodać regułę Zapory Windows?" bind:otwarte={oknoZapory} napis="Dodaj regułę" ikona="tarcza"
  niebezpieczne={false} naPotwierdz={dodajRegule}>
  <p>Serwer doda regułę przychodzącą Zapory Windows dla swoich portów (HTTP, HTTPS, FTPS i pasywnych FTP).</p>
  <p class="opis">Windows może poprosić o zgodę administratora. Regułę usuniesz w Zaporze Windows.</p>
</OknoPotwierdzenia>

<style>
  .siec {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
  }
  .sekcja {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .sekcja + .sekcja {
    padding-top: var(--odstep-7);
    border-top: var(--ramka);
  }
  .naglowek {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--odstep-m);
  }
  .naglowek .akcje {
    margin-left: auto;
    display: inline-flex;
    flex-wrap: wrap;
    gap: var(--odstep-s);
  }
  .porty {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(calc(var(--uklad-pole-w) * 0.6), 1fr));
    gap: var(--odstep-xl);
  }
  .adresy {
    list-style: none;
    margin: 0;
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
  }
  .adresy li {
    display: grid;
    grid-template-columns: var(--kolumna-l) minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--odstep-m);
    padding: var(--odstep-s) var(--odstep-l);
    min-height: var(--wysokosc-wiersz-tabeli);
  }
  .adresy li + li {
    border-top: var(--ramka);
  }
  .test {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
  }
  .pigulki {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-s);
  }
</style>
