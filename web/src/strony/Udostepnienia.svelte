<script lang="ts">
  // Udostępnienia: linki publiczne (admin widzi wszystkie, inni swoje): ścieżka, autor, wygasa, hasło,
  // pobrania / limit, adres z przyciskiem Kopiuj, usuwanie z potwierdzeniem. Nowe linki powstają na stronie Pliki
  // (menu „Pobierz przez…” → „Link publiczny…”).
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import type { Link } from '../api/typy'
  import OknoPotwierdzenia from '../komponenty/admin/OknoPotwierdzenia.svelte'
  import { adresLinku, kopiuj, maDate, wyczerpany, wygasl } from '../komponenty/admin/narzedzia'
  import NaglowekStrony from '../komponenty/NaglowekStrony.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Pigulka from '../komponenty/wspolne/Pigulka.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import Tabela, { type KolumnaTabeli } from '../komponenty/wspolne/Tabela.svelte'
  import { data, dataWzgledna } from '../stan/format'
  import { idz } from '../stan/nawigacja.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'

  let linki = $state<Link[]>([])
  let wczytano = $state(false)
  let blad = $state('')
  let oknoUsun = $state(false)
  let wybrany = $state<Link | null>(null)
  let sortKlucz = $state('utworzono')
  let malejaco = $state(true)

  const kolumny = $derived<KolumnaTabeli[]>([
    { klucz: 'sciezka', etykieta: 'Ścieżka' },
    ...(sesja.admin ? [{ klucz: 'autor', etykieta: 'Autor', szerokosc: 'm', ukryjPonizej: 'waski' as const }] : []),
    { klucz: 'wygasa', etykieta: 'Wygasa', szerokosc: 'l', ukryjPonizej: 'telefon' },
    { klucz: 'haslo', etykieta: 'Hasło', szerokosc: 's', ukryjPonizej: 'sredni', sortowalna: false },
    { klucz: 'pobrano', etykieta: 'Pobrania', szerokosc: 'm', wyrownanie: 'prawo', ukryjPonizej: 'waski' },
    { klucz: 'adres', etykieta: 'Adres', szerokosc: 'xl', ukryjPonizej: 'sredni', sortowalna: false },
    { klucz: 'akcje', etykieta: '', szerokosc: 'm', wyrownanie: 'prawo', sortowalna: false },
  ])

  const posortowane = $derived(sortuj(linki, sortKlucz, malejaco))

  function sortuj(lista: Link[], klucz: string, mal: boolean): Link[] {
    const kier = mal ? -1 : 1
    const w = (l: Link): string | number => {
      switch (klucz) {
        case 'autor':
          return l.autor.toLowerCase()
        case 'wygasa':
          return maDate(l.wygasa) ? Date.parse(l.wygasa) : Number.MAX_SAFE_INTEGER
        case 'pobrano':
          return l.pobrano
        case 'sciezka':
          return `${l.udzial}${l.sciezka}`.toLowerCase()
        default:
          return Date.parse(l.utworzono) || 0
      }
    }
    return [...lista].sort((a, b) => {
      const x = w(a)
      const y = w(b)
      return (typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y), 'pl')) * kier
    })
  }

  function stanLinku(l: Link): { stan: 'ok' | 'blad' | 'ostrz'; tekst: string } {
    if (wygasl(l)) return { stan: 'blad', tekst: 'Wygasł' }
    if (wyczerpany(l)) return { stan: 'ostrz', tekst: 'Limit wyczerpany' }
    return { stan: 'ok', tekst: 'Aktywny' }
  }

  async function wczytaj() {
    try {
      linki = await api.get<Link[]>('/udostepnienia')
      blad = ''
    } catch (b) {
      blad = trescBledu(b)
    } finally {
      wczytano = true
    }
  }

  async function kopiujAdres(l: Link) {
    const ok = await kopiuj(adresLinku(l))
    powiadom(ok ? 'Skopiowano adres linku.' : 'Nie udało się skopiować. Zaznacz adres ręcznie.', ok ? 'ok' : 'blad')
  }

  function pytajUsun(l: Link) {
    wybrany = l
    oknoUsun = true
  }

  async function usun() {
    if (!wybrany) return
    try {
      await api.usun('/udostepnienia', { id: wybrany.id })
      powiadom('Usunięto link. Kto go ma, nie pobierze już pliku.', 'ok')
      await wczytaj()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
      throw b
    }
  }

  onMount(() => {
    void wczytaj()
  })
</script>

<div class="strona">
<NaglowekStrony tytul="Udostępnienia" ikona="link"
  podtytul={sesja.admin ? "Wszystkie linki publiczne na serwerze" : "Twoje linki publiczne"} />
<Karta tytul="Linki publiczne" ikona="link" bezMarginesu>
  {#snippet akcje()}
    <span class="licznik">{linki.length}</span>
    <Przycisk rodzaj="duch" ikona="odswiez" rozmiar="maly" tytul="Odśwież" onclick={wczytaj} />
  {/snippet}
  {#if blad}
    <Pusto ikona="blad" tytul="Nie udało się wczytać linków" opis={blad}>
      <Przycisk ikona="odswiez" onclick={wczytaj}>Spróbuj ponownie</Przycisk>
    </Pusto>
  {:else}
    <Tabela {kolumny} wiersze={posortowane} bind:sortKlucz bind:malejaco naSortuj={() => {}}>
      {#snippet wiersz(l)}
        {@const s = stanLinku(l)}
        <tr>
          <td>
            <span class="sciezka">
              <span class="skroc mono" title="{l.udzial}{l.sciezka}">
                <span class="opis">{l.udzial}</span>{l.sciezka}
              </span>
              {#if s.stan !== 'ok'}<Pigulka stan={s.stan}>{s.tekst}</Pigulka>{/if}
            </span>
          </td>
          {#if sesja.admin}<td class="skroc">{l.autor}</td>{/if}
          <td>
            {#if maDate(l.wygasa)}
              <span title={data(l.wygasa)}>{dataWzgledna(l.wygasa)}</span>
            {:else}
              <span class="opis">Nigdy</span>
            {/if}
          </td>
          <td>
            <Pigulka stan={l.ma_haslo ? 'info' : 'wyl'}>{l.ma_haslo ? 'Tak' : 'Nie'}</Pigulka>
          </td>
          <td class="prawo mono">{l.pobrano} / {l.limit_pobran > 0 ? l.limit_pobran : 'bez limitu'}</td>
          <td><span class="mono opis skroc adres" title={adresLinku(l)}>{adresLinku(l)}</span></td>
          <td class="prawo">
            <span class="akcje">
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="kopiuj" tytul="Kopiuj adres" onclick={() => kopiujAdres(l)} />
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="kosz" tytul="Usuń link" onclick={() => pytajUsun(l)} />
            </span>
          </td>
        </tr>
      {/snippet}
      {#snippet pusto()}
        <Pusto ikona="link" tytul={wczytano ? 'Brak linków publicznych' : 'Wczytywanie…'}
          opis={wczytano ? 'Link utworzysz na stronie Pliki: prawy przycisk na pliku, Pobierz przez…, Link publiczny.' : undefined}>
          {#if wczytano}<Przycisk ikona="folder" onclick={() => idz('pliki')}>Przejdź do plików</Przycisk>{/if}
        </Pusto>
      {/snippet}
    </Tabela>
  {/if}
</Karta>
</div>

<OknoPotwierdzenia tytul="Usunąć link?" bind:otwarte={oknoUsun} napis="Usuń link" naPotwierdz={usun}>
  <p>Link do <span class="mono">{wybrany?.udzial}{wybrany?.sciezka}</span> przestanie działać.</p>
  <p class="opis">Plik zostaje na dysku. Tej operacji nie można cofnąć.</p>
</OknoPotwierdzenia>

<style>
  .strona {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .sciezka {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-width: 0;
  }
  .sciezka .opis {
    margin-right: var(--odstep-xs);
  }
  .adres {
    display: block;
  }
  .akcje {
    display: inline-flex;
    gap: var(--odstep-xs);
  }
</style>
