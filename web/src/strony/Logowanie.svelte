<script lang="ts">
  // Logowanie (POST /api/logowanie): login, hasło, Enter wysyła. Złe dane = komunikat błędu, 429 = blokada konta
  // po złych hasłach (komunikat z serwera, np. ile czekać). Po sukcesie odświeża sesję (App pokaże aplikację).
  import { api, BladApi, trescBledu } from '../api/klient'
  import type { Uzytkownik } from '../api/typy'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import { KOD_BLOKADY } from '../komponenty/admin/stale'
  import Pole from '../komponenty/wspolne/Pole.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import { sesja } from '../stan/sesja.svelte'

  let login = $state('')
  let haslo = $state('')
  let trwa = $state(false)
  let blad = $state('')
  let zablokowane = $state(false)

  const mozna = $derived(login.trim() !== '' && haslo !== '' && !trwa)

  async function zaloguj(e?: Event) {
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
      zablokowane = b instanceof BladApi && b.kod === KOD_BLOKADY
      blad = trescBledu(b)
      haslo = ''
    } finally {
      trwa = false
    }
  }
</script>

<EkranSrodkowy podtytul="Zaloguj się, aby zobaczyć udostępnione pliki">
  <form class="formularz" onsubmit={zaloguj} novalidate>
    <h2>Logowanie</h2>
    <Pole etykieta="Login" bind:value={login} ikona="uzytkownik" autocomplete="username" autofocus wymagane />
    <Pole etykieta="Hasło" typ="password" bind:value={haslo} ikona="klodka" autocomplete="current-password" wymagane />
    {#if blad}
      <Komunikat rodzaj={zablokowane ? 'ostrz' : 'blad'}>
        {#if zablokowane}<strong>Konto tymczasowo zablokowane.</strong>{/if}
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
    gap: var(--odstep-xl);
  }
  .akcje {
    display: flex;
    flex-direction: column;
  }
  strong {
    margin-right: var(--odstep-xs);
  }
</style>
