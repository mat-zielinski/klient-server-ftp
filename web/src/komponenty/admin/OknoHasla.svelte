<script lang="ts">
  // Zmiana hasła konta przez administratora (POST /api/konta/haslo?login): nowe hasło 2x, min. długość ze stale.ts.
  import { api, trescBledu } from '../../api/klient'
  import Okno from '../wspolne/Okno.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Komunikat from './Komunikat.svelte'
  import { bladHasla } from './narzedzia'
  import { MIN_DLUGOSC_HASLA } from './stale'

  interface Props {
    otwarte: boolean
    login: string
    naZapisano: () => void
  }

  let { otwarte = $bindable(), login, naZapisano }: Props = $props()

  let haslo = $state('')
  let haslo2 = $state('')
  let pokazBledy = $state(false)
  let trwa = $state(false)
  let blad = $state('')

  const bladHasel = $derived(bladHasla(haslo, haslo2))

  $effect(() => {
    if (!otwarte) return
    haslo = ''
    haslo2 = ''
    pokazBledy = false
    blad = ''
  })

  async function zapisz(e?: Event) {
    e?.preventDefault()
    if (bladHasel) {
      pokazBledy = true
      return
    }
    trwa = true
    blad = ''
    try {
      await api.post('/konta/haslo', { haslo }, { login })
      otwarte = false
      naZapisano()
    } catch (b) {
      blad = trescBledu(b)
    } finally {
      trwa = false
    }
  }
</script>

<Okno tytul="Zmiana hasła: {login}" bind:otwarte>
  <form class="formularz" onsubmit={zapisz} novalidate>
    <Pole etykieta="Nowe hasło" typ="password" bind:value={haslo} ikona="klodka" autocomplete="new-password" autofocus
      opis="Co najmniej {MIN_DLUGOSC_HASLA} znaków."
      blad={pokazBledy && haslo.length < MIN_DLUGOSC_HASLA ? bladHasel : undefined} />
    <Pole etykieta="Powtórz hasło" typ="password" bind:value={haslo2} ikona="klodka" autocomplete="new-password"
      blad={pokazBledy && haslo.length >= MIN_DLUGOSC_HASLA && bladHasel ? bladHasel : undefined} />
    {#if blad}<Komunikat rodzaj="blad">{blad}</Komunikat>{/if}
    <button type="submit" class="tylko-czytnik" tabindex="-1">Zmień hasło</button>
  </form>
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj="glowny" ikona="klucz" wylaczony={trwa} onclick={() => zapisz()}>Zmień hasło</Przycisk>
  {/snippet}
</Okno>

<style>
  .formularz {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
</style>
