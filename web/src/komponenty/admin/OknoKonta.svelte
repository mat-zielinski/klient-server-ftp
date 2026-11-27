<script lang="ts">
  // Okno dodania (POST /api/konta) albo edycji konta (PUT /api/konta?login): login, hasło 2x (tylko nowe konto),
  // administrator, FTPS, wyłączone, prawa do udziałów. Po zapisie woła naZapisano (strona odświeża listę).
  import { api, trescBledu } from '../../api/klient'
  import type { Prawo, Udzial, Uzytkownik } from '../../api/typy'
  import Okno from '../wspolne/Okno.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przelacznik from '../wspolne/Przelacznik.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Komunikat from './Komunikat.svelte'
  import PrawaUdzialow from './PrawaUdzialow.svelte'
  import { bladHasla } from './narzedzia'
  import { MIN_DLUGOSC_HASLA } from './stale'
  import type { KontoZHaslem } from './typy'

  interface Props {
    otwarte: boolean
    /** null = nowe konto. */
    konto: Uzytkownik | null
    udzialy: Udzial[]
    naZapisano: (login: string) => void
  }

  let { otwarte = $bindable(), konto, udzialy, naZapisano }: Props = $props()

  let login = $state('')
  let haslo = $state('')
  let haslo2 = $state('')
  let admin = $state(false)
  let ftps = $state(true)
  let wylaczony = $state(false)
  let prawa = $state<Record<string, Prawo>>({})
  let pokazBledy = $state(false)
  let trwa = $state(false)
  let blad = $state('')

  const nowe = $derived(konto === null)
  const bladLoginu = $derived(login.trim() ? '' : 'Podaj login.')
  const bladHasel = $derived(nowe ? bladHasla(haslo, haslo2) : '')

  // Wypełnienie pól przy każdym otwarciu okna.
  $effect(() => {
    if (!otwarte) return
    login = konto?.login ?? ''
    admin = konto?.admin ?? false
    ftps = konto?.ftps ?? true
    wylaczony = konto?.wylaczony ?? false
    prawa = { ...(konto?.prawa ?? {}) }
    haslo = ''
    haslo2 = ''
    pokazBledy = false
    blad = ''
  })

  async function zapisz(e?: Event) {
    e?.preventDefault()
    if (bladLoginu || bladHasel) {
      pokazBledy = true
      return
    }
    trwa = true
    blad = ''
    const dane: KontoZHaslem = { login: login.trim(), admin, ftps, wylaczony, prawa }
    try {
      if (nowe) await api.post('/konta', { ...dane, haslo })
      else await api.put('/konta', dane, { login: konto!.login })
      otwarte = false
      naZapisano(dane.login)
    } catch (b) {
      blad = trescBledu(b)
    } finally {
      trwa = false
    }
  }
</script>

<Okno tytul={nowe ? 'Nowe konto' : `Edycja konta ${konto?.login ?? ''}`} bind:otwarte szerokie>
  <form class="formularz" onsubmit={zapisz} novalidate>
    <div class="kolumny">
      <div class="kolumna">
        <Pole etykieta="Login" bind:value={login} ikona="uzytkownik" wylaczone={!nowe} autocomplete="off"
          autofocus={nowe} blad={pokazBledy ? bladLoginu : undefined}
          opis={nowe ? 'Tym loginem klient loguje się w przeglądarce i w FTPS.' : 'Loginu nie można zmienić.'} />
        {#if nowe}
          <Pole etykieta="Hasło" typ="password" bind:value={haslo} ikona="klodka" autocomplete="new-password"
            opis="Co najmniej {MIN_DLUGOSC_HASLA} znaków."
            blad={pokazBledy && haslo.length < MIN_DLUGOSC_HASLA ? bladHasel : undefined} />
          <Pole etykieta="Powtórz hasło" typ="password" bind:value={haslo2} ikona="klodka" autocomplete="new-password"
            blad={pokazBledy && haslo.length >= MIN_DLUGOSC_HASLA && bladHasel ? bladHasel : undefined} />
        {/if}
        <Przelacznik etykieta="Administrator" bind:value={admin}
          opis="Zarządza kontami, udziałami i ustawieniami, widzi wszystkie transfery." />
        <Przelacznik etykieta="Dostęp przez FTPS" bind:value={ftps}
          opis="Logowanie programem FTP (FileZilla, WinSCP, Total Commander)." />
        <Przelacznik etykieta="Konto wyłączone" bind:value={wylaczony}
          opis="Nie zaloguje się, ale konto i jego linki zostają." />
      </div>
      <div class="kolumna">
        <span class="etykieta">Prawa do udziałów</span>
        <PrawaUdzialow {udzialy} bind:value={prawa} />
      </div>
    </div>
    {#if blad}<Komunikat rodzaj="blad">{blad}</Komunikat>{/if}
    <button type="submit" class="tylko-czytnik" tabindex="-1">Zapisz</button>
  </form>
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj="glowny" ikona="zapisz" wylaczony={trwa} onclick={() => zapisz()}>
      {nowe ? 'Dodaj konto' : 'Zapisz'}
    </Przycisk>
  {/snippet}
</Okno>

<style>
  .formularz {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .kolumny {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(var(--uklad-pole-w), 1fr));
    gap: var(--odstep-7);
  }
  .kolumna {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
    min-width: 0;
  }
</style>
