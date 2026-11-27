<script lang="ts">
  // Karta „Adresy serwera” (GET /api/siec): LAN, WireGuard, publiczny; adresy HTTPS i FTPS do skopiowania,
  // porty ze stanem otwarcia i odcisk certyfikatu.
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
        <li>
          <div class="naglowek">
            <Pigulka stan={STAN_RODZAJU[a.rodzaj]}>{NAZWY_RODZAJU[a.rodzaj]}</Pigulka>
            <span class="interfejs skroc">{a.interfejs}</span>
          </div>
          {#each [https, ftps] as adres (adres)}
            <div class="adres">
              <code class="skroc">{adres}</code>
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" tytul="Kopiuj {adres}" onclick={() => skopiuj(adres)} />
            </div>
          {/each}
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
  .naglowek {
    display: flex;
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
