<script lang="ts">
  // Okno „Link publiczny…”: ważność (1 dzień, 7 dni, 30 dni, rok albo własna data, najdłużej rok), hasło,
  // limit pobrań; POST /api/udostepnienia, potem adres do skopiowania i kod QR dla telefonu.
  import type { Link } from '../../api/typy'
  import { api, trescBledu } from '../../api/klient'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Wybor from '../wspolne/Wybor.svelte'
  import KodQR from '../wspolne/KodQR.svelte'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import { kopiuj, nazwaZeSciezki } from './narzedzia'
  import { MAKS_WAZNOSC_LINKU_DNI, WAZNOSC_LINKU_DNI, WAZNOSCI_LINKU_DNI } from './stale'

  let { otwarte = $bindable(false), udzial, sciezka }: { otwarte: boolean; udzial: string; sciezka: string } = $props()

  const DZIEN_MS = 86_400_000
  const WLASNA = 'wlasna'
  const NAZWY: Record<number, string> = { 1: '1 dzień', 7: '7 dni', 30: '30 dni', 365: '1 rok' }
  const OPCJE = [
    ...WAZNOSCI_LINKU_DNI.map((d) => ({ wartosc: String(d), etykieta: NAZWY[d] ?? `${d} dni` })),
    { wartosc: WLASNA, etykieta: 'Własna data' },
  ]

  function dataZaDni(dni: number): string {
    const d = new Date(Date.now() + dni * DZIEN_MS)
    const p = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
  }

  let waznosc = $state(String(WAZNOSC_LINKU_DNI))
  let wlasnaData = $state(dataZaDni(WAZNOSC_LINKU_DNI))
  let haslo = $state('')
  let limit = $state<string | number>(0)
  let trwa = $state(false)
  let link = $state<Link | null>(null)

  // nowe okno = czysty formularz
  $effect(() => {
    if (otwarte) {
      link = null
      haslo = ''
      limit = 0
      waznosc = String(WAZNOSC_LINKU_DNI)
      wlasnaData = dataZaDni(WAZNOSC_LINKU_DNI)
    }
  })

  /** Koniec ważności (RFC 3339) albo komunikat błędu własnej daty. */
  const koniec = $derived.by((): { iso: string; blad: string } => {
    if (waznosc !== WLASNA) return { iso: new Date(Date.now() + Number(waznosc) * DZIEN_MS).toISOString(), blad: '' }
    const d = new Date(`${wlasnaData}T23:59:59`)
    if (Number.isNaN(d.getTime())) return { iso: '', blad: 'Podaj datę.' }
    if (d.getTime() < Date.now()) return { iso: '', blad: 'Data musi być w przyszłości.' }
    if (d.getTime() > Date.now() + MAKS_WAZNOSC_LINKU_DNI * DZIEN_MS) return { iso: '', blad: 'Najdłużej rok od dziś.' }
    return { iso: d.toISOString(), blad: '' }
  })

  async function utworz() {
    if (koniec.blad) return
    trwa = true
    try {
      link = await api.post<Link>('/udostepnienia', {
        udzial, sciezka, haslo,
        wygasa: waznosc === WLASNA ? koniec.iso : new Date(Date.now() + Number(waznosc) * DZIEN_MS).toISOString(),
        limit_pobran: Math.max(0, Math.floor(Number(limit) || 0)),
      })
      powiadom('Utworzono link publiczny.', 'ok')
    } catch (e) {
      powiadom(trescBledu(e), 'blad')
    } finally {
      trwa = false
    }
  }

  const pelnyAdres = $derived(link?.adres ? new URL(link.adres, location.origin).href : '')

  async function kopiujAdres() {
    if (await kopiuj(pelnyAdres)) powiadom('Skopiowano link.', 'ok')
    else powiadom('Nie udało się skopiować. Zaznacz link ręcznie.', 'blad')
  }
</script>

<Okno tytul="Link publiczny" bind:otwarte>
  <div class="tresc">
    <p class="opis">Link pozwala pobrać <b class="skroc">{nazwaZeSciezki(sciezka)}</b> bez logowania.</p>
    {#if link}
      <div class="adres">
        <code class="mono skroc" title={pelnyAdres}>{pelnyAdres}</code>
        <Przycisk rodzaj="glowny" ikona="kopiuj" onclick={kopiujAdres}>Kopiuj</Przycisk>
      </div>
      <p class="opis">
        Ważny do {new Date(link.wygasa).toLocaleString('pl-PL')}.
        {link.ma_haslo ? ' Chroniony hasłem.' : ''}
        {link.limit_pobran > 0 ? ` Limit pobrań: ${link.limit_pobran}.` : ''}
      </p>
      <div class="qr">
        <KodQR tekst={pelnyAdres} rozmiar="m" etykieta="Kod QR linku publicznego" />
        <span class="opis">Zeskanuj telefonem, aby otworzyć link.</span>
      </div>
    {:else}
      <Wybor etykieta="Ważność" opcje={OPCJE} bind:value={waznosc} />
      {#if waznosc === WLASNA}
        <Pole etykieta="Ważny do" typ="date" bind:value={wlasnaData} ikona="kalendarz" blad={koniec.blad || undefined}
          opis="Najdłużej rok od dziś." />
      {/if}
      <Pole etykieta="Hasło (opcjonalne)" typ="password" bind:value={haslo} autocomplete="new-password"
        podpowiedz="Bez hasła" ikona="klodka" />
      <Pole etykieta="Limit pobrań" typ="number" bind:value={limit} min={0} opis="0 oznacza bez limitu." />
    {/if}
  </div>
  {#snippet stopka()}
    {#if link}
      <Przycisk onclick={() => (otwarte = false)}>Zamknij</Przycisk>
    {:else}
      <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
      <Przycisk rodzaj="glowny" ikona="link" wylaczony={trwa || !!koniec.blad} onclick={utworz}>
        Utwórz link
      </Przycisk>
    {/if}
  {/snippet}
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
  .qr {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--odstep-s);
  }
</style>
