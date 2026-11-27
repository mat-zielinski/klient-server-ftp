<script lang="ts">
  // Ustawienia (admin), układ jak w stacji: kategorie po lewej (Udziały, Sieć, Certyfikat, Wygląd), karta wybranej
  // kategorii po prawej z [Zapisz] na dole. Zapis PUT /api/ustawienia całej konfiguracji; zmiana trybu, adresu,
  // portów, FTPS albo certyfikatu = pigułka „Wymaga restartu” (przy polu i w nagłówku po zapisie).
  // Wąskie okno: kategorie jako rząd nad kartą.
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import type { Konfiguracja } from '../api/typy'
  import type { NazwaIkony } from '../ikony/ikony'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import UstawieniaCertyfikat from '../komponenty/admin/UstawieniaCertyfikat.svelte'
  import UstawieniaSiec from '../komponenty/admin/UstawieniaSiec.svelte'
  import UstawieniaUdzialy from '../komponenty/admin/UstawieniaUdzialy.svelte'
  import UstawieniaWyglad from '../komponenty/admin/UstawieniaWyglad.svelte'
  import { bladPortu, kopiaKonfiguracji, wymagaRestartu } from '../komponenty/admin/narzedzia'
  import type { InfoSieci, WynikZapisu } from '../komponenty/admin/typy'
  import NaglowekStrony from '../komponenty/NaglowekStrony.svelte'
  import Ikona from '../komponenty/wspolne/Ikona.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Pigulka from '../komponenty/wspolne/Pigulka.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import { nawigacja, idz } from '../stan/nawigacja.svelte'
  import { motyw } from '../stan/motyw.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'
  import { uklad } from '../stan/uklad.svelte'

  type Kategoria = 'udzialy' | 'siec' | 'certyfikat' | 'wyglad'
  const KATEGORIE: { id: Kategoria; nazwa: string; ikona: NazwaIkony; opis: string }[] = [
    { id: 'udzialy', nazwa: 'Udziały', ikona: 'folder', opis: 'Foldery udostępniane klientom' },
    { id: 'siec', nazwa: 'Sieć', ikona: 'siec', opis: 'Tryb, adres, porty, FTPS, zapora' },
    { id: 'certyfikat', nazwa: 'Certyfikat', ikona: 'klodka', opis: 'HTTPS i FTPS' },
    { id: 'wyglad', nazwa: 'Wygląd', ikona: 'slonce', opis: 'Motyw ciemny lub jasny' },
  ]

  let konfig = $state<Konfiguracja | null>(null)
  let zapisana = $state<Konfiguracja | null>(null)
  let siec = $state<InfoSieci | null>(null)
  let bladSieci = $state('')
  let blad = $state('')
  let zapisuje = $state(false)
  let restartPoZapisie = $state(false)

  const kategoria = $derived<Kategoria>(
    KATEGORIE.some((k) => k.id === nawigacja.parametry.kategoria) ? (nawigacja.parametry.kategoria as Kategoria) : 'udzialy',
  )
  const biezaca = $derived(KATEGORIE.find((k) => k.id === kategoria) ?? KATEGORIE[0])
  const zmienione = $derived(!!konfig && !!zapisana && JSON.stringify(normalizuj(konfig)) !== JSON.stringify(zapisana))
  const restartZmian = $derived(!!konfig && !!zapisana && wymagaRestartu(zapisana, normalizuj(konfig)))
  const bladFormularza = $derived(konfig ? sprawdz(konfig) : '')

  function normalizuj(k: Konfiguracja): Konfiguracja {
    const n = kopiaKonfiguracji(k)
    n.port_http = Number(k.port_http)
    n.port_https = Number(k.port_https)
    n.port_ftps = Number(k.port_ftps)
    n.porty_pasywne_ftp = [Number(k.porty_pasywne_ftp[0]), Number(k.porty_pasywne_ftp[1])]
    n.adres_nasluchu = k.adres_nasluchu.trim()
    n.domena_lets_encrypt = k.domena_lets_encrypt.trim()
    n.certyfikat = k.certyfikat.trim()
    n.klucz = k.klucz.trim()
    return n
  }

  function sprawdz(k: Konfiguracja): string {
    const n = normalizuj(k)
    for (const [nazwa, port] of [
      ['HTTP', n.port_http],
      ['HTTPS', n.port_https],
      ['FTPS', n.port_ftps],
      ['pasywny FTP (od)', n.porty_pasywne_ftp[0]],
      ['pasywny FTP (do)', n.porty_pasywne_ftp[1]],
    ] as const) {
      const b = bladPortu(port)
      if (b) return `Port ${nazwa}: ${b}`
    }
    if (n.porty_pasywne_ftp[1] < n.porty_pasywne_ftp[0]) return 'Zakres portów pasywnych FTP jest odwrócony.'
    const glowne = [n.port_http, n.port_https, ...(n.ftps_wlaczone ? [n.port_ftps] : [])]
    if (new Set(glowne).size !== glowne.length) return 'Porty HTTP, HTTPS i FTPS muszą być różne.'
    if (n.certyfikat && !n.klucz) return 'Podaj plik klucza do certyfikatu.'
    return ''
  }

  async function wczytaj() {
    try {
      const k = await api.get<Konfiguracja>('/ustawienia')
      konfig = kopiaKonfiguracji(k)
      zapisana = kopiaKonfiguracji(k)
      blad = ''
    } catch (b) {
      blad = trescBledu(b)
    }
    await wczytajSiec()
  }

  async function wczytajSiec() {
    try {
      siec = await api.get<InfoSieci>('/siec')
      bladSieci = ''
    } catch (b) {
      bladSieci = trescBledu(b)
    }
  }

  async function zapisz() {
    if (!konfig || !zapisana) return
    if (bladFormularza) {
      powiadom(bladFormularza, 'blad')
      return
    }
    zapisuje = true
    const nowa = normalizuj(konfig)
    const restart = wymagaRestartu(zapisana, nowa)
    try {
      const wynik = await api.put<WynikZapisu | undefined>('/ustawienia', nowa)
      zapisana = kopiaKonfiguracji(nowa)
      konfig = kopiaKonfiguracji(nowa)
      restartPoZapisie = restartPoZapisie || restart || !!wynik?.restart
      powiadom(
        wynik?.komunikat || (restart ? 'Zapisano. Zmiany sieci zadziałają po restarcie serwera.' : 'Zapisano ustawienia.'),
        restart ? 'ostrz' : 'ok',
      )
      void wczytajSiec()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
    } finally {
      zapisuje = false
    }
  }

  function cofnij() {
    if (!zapisana) return
    konfig = kopiaKonfiguracji(zapisana)
    if (zapisana.motyw === 'jasny' || zapisana.motyw === 'ciemny') motyw.ustaw(zapisana.motyw)
  }

  onMount(() => {
    if (sesja.admin) void wczytaj()
  })
</script>

{#if !sesja.admin}
  <Pusto ikona="klodka" tytul="Ustawienia są dostępne dla administratora" />
{:else if blad}
  <Pusto ikona="blad" tytul="Nie udało się wczytać ustawień" opis={blad}>
    <Przycisk ikona="odswiez" onclick={wczytaj}>Spróbuj ponownie</Przycisk>
  </Pusto>
{:else if !konfig || !zapisana}
  <Pusto ikona="klepsydra" tytul="Wczytywanie ustawień…" />
{:else}
  <NaglowekStrony tytul="Ustawienia" podtytul="Udziały, sieć, certyfikat i wygląd serwera" ikona="ustawienia">
    {#snippet akcje()}
      {#if restartPoZapisie}<Pigulka stan="ostrz" kropka>Wymaga restartu serwera</Pigulka>{/if}
      <Pigulka stan={konfig?.tryb_sieci === 'publiczny' ? 'ostrz' : 'ok'}>
        {konfig?.tryb_sieci === 'publiczny' ? 'Tryb publiczny' : 'Tryb VPN'}
      </Pigulka>
    {/snippet}
  </NaglowekStrony>
  <div class="ustawienia" class:waskie={uklad.waski}>
    <nav class="kategorie" aria-label="Kategorie ustawień">
      {#each KATEGORIE as k (k.id)}
        <button type="button" class="kategoria" class:aktywna={k.id === kategoria}
          aria-current={k.id === kategoria ? 'page' : undefined} title={k.opis}
          onclick={() => idz('ustawienia', { kategoria: k.id })}>
          <Ikona nazwa={k.ikona} />
          <span class="napisy">
            <span class="nazwa">{k.nazwa}</span>
            {#if !uklad.waski}<span class="opis skroc">{k.opis}</span>{/if}
          </span>
        </button>
      {/each}
    </nav>

    <div class="karta">
      <Karta tytul={biezaca.nazwa} ikona={biezaca.ikona}>
        {#snippet akcje()}
          {#if zmienione}<Pigulka stan="info">Niezapisane zmiany</Pigulka>{/if}
        {/snippet}
        <div class="tresc">
          {#if kategoria === 'udzialy'}
            <UstawieniaUdzialy bind:udzialy={konfig.udzialy} />
          {:else if kategoria === 'siec'}
            <UstawieniaSiec bind:konfig {zapisana} {siec} {bladSieci} naTestuj={wczytajSiec} />
          {:else if kategoria === 'certyfikat'}
            <UstawieniaCertyfikat bind:konfig {zapisana} odcisk={siec?.odcisk_certyfikatu ?? ''} />
          {:else}
            <UstawieniaWyglad bind:konfig />
          {/if}

          {#if bladFormularza}<Komunikat rodzaj="blad">{bladFormularza}</Komunikat>{/if}
          {#if restartZmian}
            <Komunikat rodzaj="ostrz">Część zmian zadziała dopiero po restarcie serwera.</Komunikat>
          {/if}

          <div class="stopka">
            <Przycisk rodzaj="duch" ikona="odtworz" wylaczony={!zmienione || zapisuje} onclick={cofnij}>Cofnij zmiany</Przycisk>
            <Przycisk rodzaj="glowny" ikona="zapisz" wylaczony={!zmienione || zapisuje || !!bladFormularza} onclick={zapisz}>
              {zapisuje ? 'Zapisywanie…' : 'Zapisz'}
            </Przycisk>
          </div>
        </div>
      </Karta>
    </div>
  </div>
{/if}

<style>
  .ustawienia {
    display: grid;
    grid-template-columns: var(--kolumna-xl) minmax(0, 1fr);
    gap: var(--odstep-7);
    align-items: start;
    min-width: 0;
  }
  .ustawienia.waskie {
    grid-template-columns: minmax(0, 1fr);
  }
  .kategorie {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    padding: var(--odstep-s);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
  }
  .waskie .kategorie {
    flex-direction: row;
    overflow-x: auto;
  }
  .kategoria {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    min-height: var(--wysokosc-pozycja-paska);
    padding: var(--odstep-s) var(--odstep-l);
    border: 0;
    border-left: var(--odstep-pozycje) solid transparent;
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-opis);
    text-align: left;
    min-width: 0;
  }
  .waskie .kategoria {
    flex: 1 0 auto;
    border-left: 0;
    border-bottom: var(--odstep-pozycje) solid transparent;
  }
  .kategoria:hover {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  .kategoria.aktywna {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
    border-left-color: var(--kolor-niebieski);
  }
  .waskie .kategoria.aktywna {
    border-bottom-color: var(--kolor-niebieski);
  }
  .napisy {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
    color: var(--kolor-tekst);
  }
  .aktywna .nazwa {
    color: var(--kolor-niebieski);
  }
  .karta {
    min-width: 0;
  }
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
  }
  .stopka {
    display: flex;
    justify-content: flex-end;
    gap: var(--odstep-m);
    padding-top: var(--odstep-xl);
    border-top: var(--ramka);
  }
</style>
