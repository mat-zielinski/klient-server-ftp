<script lang="ts">
  // Logowanie (POST /api/logowanie): login, hasło (oko + Caps Lock), Enter wysyła. Złe dane = komunikat błędu,
  // 429 = blokada konta po złych hasłach (komunikat z serwera, np. ile czekać). Po sukcesie odświeża sesję
  // (App pokaże aplikację).
  import { tick } from 'svelte'
  import { api, BladApi } from '../api/klient'
  import type { Uzytkownik } from '../api/typy'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import { KOD_BLOKADY } from '../komponenty/admin/stale'
  import Pole from '../komponenty/wspolne/Pole.svelte'
  import PoleHasla from '../komponenty/wspolne/PoleHasla.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import { sesja } from '../stan/sesja.svelte'

  let login = $state('')
  let haslo = $state('')
  let trwa = $state(false)
  let blad = $state('')
  let zablokowane = $state(false)
  let formularz = $state<HTMLFormElement>()

  const mozna = $derived(!trwa && login.trim() !== '' && haslo !== '')

  async function wyslij(e?: Event) {
    e?.preventDefault()
    if (!mozna) return
    trwa = true
    blad = ''
    zablokowane = false
    try {
      await api.post<Uzytkownik>('/logowanie', { login: login.trim(), haslo })
      haslo = ''
      await sesja.odswiez()
    } catch (b) {
      await obsluzBlad(b)
    } finally {
      trwa = false
    }
  }

  async function obsluzBlad(b: unknown) {
    const bl = b instanceof BladApi ? b : new BladApi(-1, 'Nie udało się zalogować.')
    zablokowane = bl.kod === KOD_BLOKADY
    blad = bl.komunikat
    haslo = ''
    await tick()
    formularz?.querySelector<HTMLInputElement>('input[type="password"]')?.focus()
  }
</script>

<EkranSrodkowy>
  <form class="formularz" bind:this={formularz} onsubmit={wyslij} novalidate>
    <header>
      <h2>Zaloguj się</h2>
      <p class="opis">Witaj ponownie. Podaj login i hasło do Skrytki.</p>
    </header>
    <Pole etykieta="Login" bind:value={login} ikona="uzytkownik" autocomplete="username" autofocus wymagane duze
      podpowiedz="Twój login" />
    <PoleHasla etykieta="Hasło" bind:value={haslo} autocomplete="current-password" podpowiedz="Hasło" />

    {#if blad}
      <Komunikat rodzaj={zablokowane ? 'ostrz' : 'blad'}>
        {#if zablokowane}<strong class="blokada">Konto tymczasowo zablokowane.</strong>{/if}
        {blad}
      </Komunikat>
    {/if}

    <div class="akcje">
      <Przycisk rodzaj="glowny" typ="submit" ikona="zaloguj" rozmiar="duzy" wylaczony={!mozna}>
        {trwa ? 'Logowanie…' : 'Zaloguj'}
      </Przycisk>
    </div>
  </form>
</EkranSrodkowy>

<style>
  .formularz {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-sekcja);
  }
  header {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    margin-bottom: var(--odstep-xs);
  }
  h2 {
    margin: 0;
    font-size: var(--czcionka-tytul-strony);
  }
  header .opis {
    margin: 0;
  }
  .akcje {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
    margin-top: var(--odstep-xs);
  }
  .blokada {
    margin-right: var(--odstep-xs);
  }
</style>
