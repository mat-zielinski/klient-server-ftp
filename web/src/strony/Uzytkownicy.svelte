<script lang="ts">
  // Użytkownicy (admin): tabela kont (login, administrator, FTPS, prawa do udziałów pigułkami, stan), dodawanie
  // i edycja w oknie, zmiana hasła, usuwanie z potwierdzeniem. Udziały do praw z GET /api/ustawienia.
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import type { Konfiguracja, Prawo, Udzial, Uzytkownik } from '../api/typy'
  import OknoHasla from '../komponenty/admin/OknoHasla.svelte'
  import OknoKonta from '../komponenty/admin/OknoKonta.svelte'
  import OknoPotwierdzenia from '../komponenty/admin/OknoPotwierdzenia.svelte'
  import { STAN_PRAWA } from '../komponenty/admin/stale'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Kafel from '../komponenty/wspolne/Kafel.svelte'
  import NaglowekStrony from '../komponenty/NaglowekStrony.svelte'
  import Pigulka from '../komponenty/wspolne/Pigulka.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import Tabela, { type KolumnaTabeli } from '../komponenty/wspolne/Tabela.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'

  const KOLUMNY: KolumnaTabeli[] = [
    { klucz: 'login', etykieta: 'Login', szerokosc: 'l' },
    { klucz: 'admin', etykieta: 'Rola', szerokosc: 'm', ukryjPonizej: 'telefon' },
    { klucz: 'ftps', etykieta: 'FTPS', szerokosc: 's', ukryjPonizej: 'waski' },
    { klucz: 'prawa', etykieta: 'Prawa do udziałów', ukryjPonizej: 'telefon' },
    { klucz: 'stan', etykieta: 'Stan', szerokosc: 'm' },
    { klucz: 'akcje', etykieta: '', szerokosc: 'l', wyrownanie: 'prawo', sortowalna: false },
  ]

  let konta = $state<Uzytkownik[]>([])
  let udzialy = $state<Udzial[]>([])
  let wczytano = $state(false)
  let blad = $state('')
  let sortKlucz = $state('login')
  let malejaco = $state(false)

  let oknoKonta = $state(false)
  let edytowane = $state<Uzytkownik | null>(null)
  let oknoHasla = $state(false)
  let oknoUsun = $state(false)
  let wybrane = $state<Uzytkownik | null>(null)

  const posortowane = $derived(sortuj(konta, sortKlucz, malejaco))
  const admini = $derived(konta.filter((k) => k.admin).length)
  const wylaczone = $derived(konta.filter((k) => k.wylaczony).length)

  function sortuj(lista: Uzytkownik[], klucz: string, mal: boolean): Uzytkownik[] {
    const kier = mal ? -1 : 1
    const wartosc = (k: Uzytkownik): string =>
      klucz === 'admin' ? String(+!k.admin) : klucz === 'ftps' ? String(+!k.ftps) : klucz === 'stan' ? String(+k.wylaczony) : k.login.toLowerCase()
    return [...lista].sort((a, b) => wartosc(a).localeCompare(wartosc(b), 'pl') * kier || a.login.localeCompare(b.login, 'pl'))
  }

  function prawaKonta(k: Uzytkownik): [string, Prawo][] {
    return Object.entries(k.prawa).filter(([, p]) => p !== 'brak')
  }

  async function wczytaj() {
    try {
      const [k, ust] = await Promise.all([
        api.get<Uzytkownik[]>('/konta'),
        api.get<Konfiguracja>('/ustawienia'),
      ])
      konta = k
      udzialy = ust.udzialy
      blad = ''
    } catch (b) {
      blad = trescBledu(b)
    } finally {
      wczytano = true
    }
  }

  function dodaj() {
    edytowane = null
    oknoKonta = true
  }

  function edytuj(k: Uzytkownik) {
    edytowane = k
    oknoKonta = true
  }

  function zmienHaslo(k: Uzytkownik) {
    wybrane = k
    oknoHasla = true
  }

  function pytajUsun(k: Uzytkownik) {
    wybrane = k
    oknoUsun = true
  }

  async function usun() {
    if (!wybrane) return
    const login = wybrane.login
    try {
      await api.usun('/konta', { login })
      powiadom(`Usunięto konto ${login}.`, 'ok')
      await wczytaj()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
      throw b
    }
  }

  function zapisano(login: string) {
    powiadom(edytowane ? `Zapisano konto ${login}.` : `Dodano konto ${login}.`, 'ok')
    void wczytaj()
  }

  onMount(() => {
    if (sesja.admin) void wczytaj()
  })
</script>

{#if !sesja.admin}
  <Pusto ikona="klodka" tytul="Użytkownicy są dostępni dla administratora" />
{:else}
  <div class="strona">
    <NaglowekStrony tytul="Użytkownicy" podtytul="Konta, prawa do udziałów i dostęp FTPS" ikona="uzytkownicy">
      {#snippet srodek()}
        <div class="kafle">
          <Kafel etykieta="Konta" wartosc={konta.length} ikona="uzytkownicy" />
          <Kafel etykieta="Administratorzy" wartosc={admini} ikona="tarcza" stan="info" />
          <Kafel etykieta="Wyłączone" wartosc={wylaczone} ikona="klodka" stan={wylaczone ? 'ostrz' : 'wyl'} />
        </div>
      {/snippet}
      {#snippet akcje()}
        <Przycisk rodzaj="glowny" ikona="plus" onclick={dodaj}>Dodaj konto</Przycisk>
      {/snippet}
    </NaglowekStrony>

    <Karta tytul="Konta" ikona="uzytkownicy" bezMarginesu>
      {#snippet akcje()}
        <span class="licznik">{konta.length}</span>
        <Przycisk rodzaj="duch" ikona="odswiez" rozmiar="maly" tytul="Odśwież" onclick={wczytaj} />
      {/snippet}
      {#if blad}
        <Pusto ikona="blad" tytul="Nie udało się wczytać kont" opis={blad}>
          <Przycisk ikona="odswiez" onclick={wczytaj}>Spróbuj ponownie</Przycisk>
        </Pusto>
      {:else}
        <Tabela kolumny={KOLUMNY} wiersze={posortowane} bind:sortKlucz bind:malejaco naSortuj={() => {}}>
          {#snippet wiersz(k)}
            <tr>
              <td>
                <span class="login">{k.login}</span>
                {#if k.login === sesja.stan?.uzytkownik?.login}<span class="opis">(Ty)</span>{/if}
              </td>
              <td>
                <Pigulka stan={k.admin ? 'info' : 'wyl'}>{k.admin ? 'Administrator' : 'Użytkownik'}</Pigulka>
              </td>
              <td>
                <Pigulka stan={k.ftps ? 'ok' : 'wyl'}>{k.ftps ? 'Tak' : 'Nie'}</Pigulka>
              </td>
              <td>
                <span class="prawa">
                  {#each prawaKonta(k) as [udzial, prawo] (udzial)}
                    <Pigulka stan={STAN_PRAWA[prawo]}>{udzial}: {prawo === 'zapis' ? 'zapis' : 'odczyt'}</Pigulka>
                  {:else}
                    <span class="opis">Brak praw</span>
                  {/each}
                </span>
              </td>
              <td>
                <Pigulka stan={k.wylaczony ? 'blad' : 'ok'} kropka>{k.wylaczony ? 'Wyłączone' : 'Aktywne'}</Pigulka>
              </td>
              <td class="prawo">
                <span class="akcje">
                  <Przycisk rodzaj="duch" rozmiar="maly" ikona="olowek" tytul="Edytuj" onclick={() => edytuj(k)} />
                  <Przycisk rodzaj="duch" rozmiar="maly" ikona="klucz" tytul="Zmień hasło" onclick={() => zmienHaslo(k)} />
                  <Przycisk rodzaj="duch" rozmiar="maly" ikona="kosz" tytul="Usuń"
                    wylaczony={k.login === sesja.stan?.uzytkownik?.login} onclick={() => pytajUsun(k)} />
                </span>
              </td>
            </tr>
          {/snippet}
          {#snippet pusto()}
            <Pusto ikona="uzytkownicy" tytul={wczytano ? 'Brak kont' : 'Wczytywanie…'} />
          {/snippet}
        </Tabela>
      {/if}
    </Karta>
  </div>

  <OknoKonta bind:otwarte={oknoKonta} konto={edytowane} {udzialy} naZapisano={zapisano} />
  <OknoHasla bind:otwarte={oknoHasla} login={wybrane?.login ?? ''}
    naZapisano={() => powiadom(`Zmieniono hasło konta ${wybrane?.login ?? ''}.`, 'ok')} />
  <OknoPotwierdzenia tytul="Usunąć konto?" bind:otwarte={oknoUsun} napis="Usuń konto" naPotwierdz={usun}>
    <p>Konto <strong>{wybrane?.login}</strong> zostanie usunięte. Tej operacji nie można cofnąć.</p>
    <p class="opis">Pliki w udziałach zostają.</p>
  </OknoPotwierdzenia>
{/if}

<style>
  .strona {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .kafle {
    display: flex;
    gap: var(--odstep-l);
  }
  .login {
    font-weight: var(--czcionka-polgruby);
    margin-right: var(--odstep-xs);
  }
  .prawa {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-xs);
  }
  .akcje {
    display: inline-flex;
    gap: var(--odstep-xs);
  }
</style>
