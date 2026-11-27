<script lang="ts">
  // Karta „Adresy serwera” (GET /api/siec): LAN, WireGuard, publiczny; adresy HTTPS i FTPS do skopiowania,
  // porty ze stanem otwarcia i odcisk certyfikatu. W trybie VPN adres publiczny jest wyszarzony (serwer nie słucha
  // na nim), a adres WireGuard (bez niego LAN) wyróżniony jako zalecany dla klientów.
  import type { Adres } from '../../api/typy'
  import Karta from '../wspolne/Karta.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import { kopiuj, type InfoSieci, type StanPigulki } from './typy'

  interface Props { siec: InfoSieci | null; blad?: string }
  let { siec, blad = '' }: Props = $props()

  const NAZWY_RODZAJU: Record<Adres['rodzaj'], string> = { lan: 'LAN', wireguard: 'WireGuard', publiczny: 'Publiczny', petla: 'Pętla' }
  const STAN_RODZAJU: Record<Adres['rodzaj'], StanPigulki> = { lan: 'ok', wireguard: 'info', publiczny: 'ostrz', petla: 'wyl' }
  const KOLEJNOSC: Adres['rodzaj'][] = ['lan', 'wireguard', 'publiczny']

  const adresy = $derived.by((): Adres[] => {
    if (!siec) return []
    const lista = (siec.adresy ?? []).filter((a) => a.rodzaj !== 'petla')
    if (siec.publiczny && !lista.some((a) => a.ip === siec.publiczny)) {
      lista.push({ interfejs: 'Internet', ip: siec.publiczny, rodzaj: 'publiczny' })
    }
    return lista.sort((a, b) => KOLEJNOSC.indexOf(a.rodzaj) - KOLEJNOSC.indexOf(b.rodzaj))
  })

  const trybVpn = $derived(siec?.tryb !== 'publiczny')
  /** Rodzaj adresu zalecanego w trybie VPN: WireGuard, a gdy go brak, LAN. */
  const zalecany = $derived<Adres['rodzaj'] | null>(
    !trybVpn ? null : adresy.some((a) => a.rodzaj === 'wireguard') ? 'wireguard' : adresy.some((a) => a.rodzaj === 'lan') ? 'lan' : null,
  )

  const porty = $derived(
    siec
      ? [
          { klucz: 'https', nazwa: 'HTTPS', port: siec.porty.https },
          { klucz: 'ftps', nazwa: 'FTPS', port: siec.porty.ftps },
          { klucz: 'http', nazwa: 'HTTP', port: siec.porty.http },
        ].filter((p) => p.port > 0)
      : [],
  )

  function host(ip: string): string {
    return ip.includes(':') ? `[${ip}]` : ip
  }

  async function skopiuj(tekst: string) {
    if (await kopiuj(tekst)) powiadom('Skopiowano do schowka.', 'ok')
    else powiadom('Nie udało się skopiować. Zaznacz adres i skopiuj ręcznie.', 'ostrz')
  }
</script>

<Karta tytul="Adresy serwera" ikona="siec">
  {#if blad}
    <Pusto ikona="alert" tytul="Brak informacji o sieci" opis={blad} />
  {:else if !siec}
    <p class="opis">Wczytywanie…</p>
  {:else}
    <ul class="adresy">
      {#each adresy as a (a.ip)}
        {@const https = `https://${host(a.ip)}:${siec.porty.https}`}
        {@const ftps = `ftps://${host(a.ip)}:${siec.porty.ftps}`}
        {@const niedostepny = trybVpn && a.rodzaj === 'publiczny'}
        <li class:niedostepny class:zalecany={a.rodzaj === zalecany}>
          <div class="naglowek">
            <Pigulka stan={niedostepny ? 'wyl' : STAN_RODZAJU[a.rodzaj]}>{NAZWY_RODZAJU[a.rodzaj]}</Pigulka>
            <span class="interfejs skroc">{a.interfejs}</span>
            {#if niedostepny}
              <Pigulka stan="wyl">Niedostępny w trybie VPN</Pigulka>
            {:else if a.rodzaj === zalecany}
              <Pigulka stan="ok" kropka>Zalecany dla klientów</Pigulka>
            {/if}
          </div>
          {#if niedostepny}
            <div class="adres"><code class="skroc">{a.ip}</code></div>
            <p class="opis-adresu">Klienci łączą się przez WireGuard adresem LAN.</p>
          {:else}
            {#each [https, ftps] as adres (adres)}
              <div class="adres">
                <code class="skroc">{adres}</code>
                <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" tytul="Kopiuj {adres}" onclick={() => skopiuj(adres)} />
              </div>
            {/each}
          {/if}
        </li>
      {:else}
        <li class="opis">Brak adresów sieciowych.</li>
      {/each}
    </ul>

    <div class="porty">
      {#each porty as p (p.klucz)}
        {@const otwarty = siec.porty_otwarte?.[p.klucz]}
        <span class="port">
          <span class="etykieta">{p.nazwa}</span>
          <span class="mono">{p.port}</span>
          {#if otwarty !== undefined}
            <Pigulka stan={otwarty ? 'ok' : 'wyl'}>{otwarty ? 'Otwarty' : 'Zamknięty'}</Pigulka>
          {/if}
        </span>
      {/each}
      <span class="port">
        <span class="etykieta">Tryb</span>
        <span>{siec.tryb === 'publiczny' ? 'Publiczny' : 'Tylko VPN'}</span>
      </span>
    </div>

    {#if siec.odcisk_certyfikatu}
      <div class="odcisk">
        <span class="etykieta">Odcisk certyfikatu (SHA-256)</span>
        <div class="adres">
          <code class="skroc">{siec.odcisk_certyfikatu}</code>
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" tytul="Kopiuj odcisk certyfikatu" onclick={() => skopiuj(siec.odcisk_certyfikatu)} />
        </div>
      </div>
    {/if}
  {/if}
</Karta>

<style>
  .adresy {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-4);
  }
  .zalecany {
    background: var(--kolor-tlo-ok);
    border-radius: var(--promien-segment);
    padding: var(--odstep-3);
  }
  .niedostepny code {
    color: var(--kolor-wylaczony-napis);
    background: var(--kolor-wylaczony-tlo);
  }
  .opis-adresu {
    margin: var(--odstep-1) 0 0;
    color: var(--kolor-wylaczony-napis);
    font-size: var(--czcionka-maly);
  }
  .naglowek {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--odstep-3);
    margin-bottom: var(--odstep-1);
  }
  .interfejs {
    color: var(--kolor-opis);
    font-size: var(--czcionka-maly);
  }
  .adres {
    display: flex;
    align-items: center;
    gap: var(--odstep-2);
    min-width: 0;
  }
  code {
    flex: 1;
    min-width: 0;
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-maly);
    background: var(--kolor-mono);
    border: var(--ramka);
    border-radius: var(--promien-segment);
    padding: var(--odstep-1) var(--odstep-2);
  }
  .porty {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-3) var(--odstep-5);
    margin-top: var(--odstep-5);
    padding-top: var(--odstep-4);
    border-top: var(--ramka);
  }
  .port {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-2);
    font-size: var(--czcionka-maly);
  }
  .odcisk {
    margin-top: var(--odstep-4);
    display: flex;
    flex-direction: column;
    gap: var(--odstep-1);
  }
</style>
