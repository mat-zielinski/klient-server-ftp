<script lang="ts">
  // Kreator pierwszego uruchomienia (POST /api/kreator, tylko gdy brak kont): 1 konto administratora (hasło 2x,
  // min. długość z stale.ts), 2 pierwszy udział (nazwa, folder na dysku, tylko do odczytu), 3 tryb sieci
  // (VPN zalecany albo Publiczny), 4 podsumowanie. Serwer zakłada konto i loguje; potem odświeżamy sesję.
  import { api, trescBledu } from '../api/klient'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import KrokiKreatora from '../komponenty/admin/KrokiKreatora.svelte'
  import WyborTrybu from '../komponenty/admin/WyborTrybu.svelte'
  import { bladHasla } from '../komponenty/admin/narzedzia'
  import { KREATOR, MIN_DLUGOSC_HASLA, TRYBY_SIECI } from '../komponenty/admin/stale'
  import type { DaneKreatora, TrybSieci } from '../komponenty/admin/typy'
  import Pole from '../komponenty/wspolne/Pole.svelte'
  import Przelacznik from '../komponenty/wspolne/Przelacznik.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'

  const KROKI = ['Konto administratora', 'Pierwszy udział', 'Tryb sieci', 'Podsumowanie']

  let krok = $state(0)
  let login = $state(KREATOR.login)
  let haslo = $state('')
  let haslo2 = $state('')
  let nazwaUdzialu = $state(KREATOR.nazwaUdzialu)
  let sciezka = $state(KREATOR.sciezkaUdzialu)
  let tylkoOdczyt = $state(KREATOR.tylkoOdczyt)
  let tryb = $state<TrybSieci>(KREATOR.tryb)
  let pokazBledy = $state(false)
  let trwa = $state(false)
  let bladSerwera = $state('')

  const bladLoginu = $derived(login.trim() ? '' : 'Podaj login.')
  const bladHasel = $derived(bladHasla(haslo, haslo2))
  const bladNazwy = $derived(nazwaUdzialu.trim() ? '' : 'Podaj nazwę udziału.')
  const bladSciezki = $derived(sciezka.trim() ? '' : 'Podaj ścieżkę folderu na dysku.')

  const krokDobry = $derived(
    [!bladLoginu && !bladHasel, !bladNazwy && !bladSciezki, true, true][krok] ?? true,
  )

  function dalej(e?: Event) {
    e?.preventDefault()
    if (!krokDobry) {
      pokazBledy = true
      return
    }
    pokazBledy = false
    if (krok < KROKI.length - 1) krok++
    else void zakoncz()
  }

  function wstecz() {
    bladSerwera = ''
    if (krok > 0) krok--
  }

  async function zakoncz() {
    trwa = true
    bladSerwera = ''
    const dane: DaneKreatora = {
      login: login.trim(),
      haslo,
      udzial: { nazwa: nazwaUdzialu.trim(), sciezka: sciezka.trim(), tylko_odczyt: tylkoOdczyt },
      tryb_sieci: tryb,
    }
    try {
      await api.post('/kreator', dane)
      powiadom('Serwer skonfigurowany. Witaj w Skrytce.', 'ok')
      await sesja.odswiez()
    } catch (b) {
      bladSerwera = trescBledu(b)
    } finally {
      trwa = false
    }
  }
</script>

<EkranSrodkowy podtytul="Pierwsze uruchomienie: trzy kroki do działającego serwera" szeroka>
  <form class="kreator" onsubmit={dalej} novalidate>
    <KrokiKreatora kroki={KROKI} biezacy={krok} />

    {#if krok === 0}
      <section class="krok">
        <h2>Konto administratora</h2>
        <p class="opis">Administrator zakłada konta, udostępnia foldery i zmienia ustawienia serwera.</p>
        <Pole etykieta="Login" bind:value={login} ikona="uzytkownik" autocomplete="username" autofocus
          blad={pokazBledy ? bladLoginu : undefined} />
        <Pole etykieta="Hasło" typ="password" bind:value={haslo} ikona="klodka" autocomplete="new-password"
          opis="Co najmniej {MIN_DLUGOSC_HASLA} znaków." blad={pokazBledy && haslo.length < MIN_DLUGOSC_HASLA ? bladHasel : undefined} />
        <Pole etykieta="Powtórz hasło" typ="password" bind:value={haslo2} ikona="klodka" autocomplete="new-password"
          blad={pokazBledy && haslo.length >= MIN_DLUGOSC_HASLA ? bladHasel || undefined : undefined} />
      </section>
    {:else if krok === 1}
      <section class="krok">
        <h2>Pierwszy udział</h2>
        <p class="opis">Folder na tym komputerze, który zobaczą klienci. Kolejne dodasz w Ustawieniach.</p>
        <Pole etykieta="Nazwa udziału" bind:value={nazwaUdzialu} ikona="folder" autofocus
          opis="Tak udział zobaczą klienci, np. Filmy." blad={pokazBledy ? bladNazwy : undefined} />
        <Pole etykieta="Ścieżka folderu na dysku" bind:value={sciezka} ikona="dysk" mono
          opis="Pełna ścieżka, np. D:\Udostepnione." blad={pokazBledy ? bladSciezki : undefined} />
        <Przelacznik etykieta="Tylko do odczytu" bind:value={tylkoOdczyt}
          opis="Nikt nie wgra, nie zmieni ani nie usunie plików w tym udziale, także administrator." />
      </section>
    {:else if krok === 2}
      <section class="krok">
        <h2>Tryb sieci</h2>
        <p class="opis">Jak klienci dostaną się do serwera spoza domu. Zmienisz to później w Ustawieniach.</p>
        <WyborTrybu bind:value={tryb} />
      </section>
    {:else}
      <section class="krok">
        <h2>Podsumowanie</h2>
        <dl class="podsumowanie">
          <dt>Administrator</dt>
          <dd>{login.trim()}</dd>
          <dt>Udział</dt>
          <dd>{nazwaUdzialu.trim()} {tylkoOdczyt ? '(tylko do odczytu)' : '(odczyt i zapis)'}</dd>
          <dt>Folder</dt>
          <dd class="mono skroc" title={sciezka}>{sciezka.trim()}</dd>
          <dt>Tryb sieci</dt>
          <dd>{TRYBY_SIECI[tryb].nazwa}</dd>
        </dl>
        {#if tryb === 'publiczny'}
          <Komunikat rodzaj="ostrz">{TRYBY_SIECI.publiczny.ostrzezenie}</Komunikat>
        {:else}
          <Komunikat rodzaj="info">
            Klienci łączą się najpierw tunelem WireGuard, potem otwierają adres LAN tego komputera w przeglądarce.
          </Komunikat>
        {/if}
        {#if bladSerwera}<Komunikat rodzaj="blad">{bladSerwera}</Komunikat>{/if}
      </section>
    {/if}

    <div class="stopka">
      <Przycisk rodzaj="duch" ikona="strzalka_lewo" wylaczony={krok === 0 || trwa} onclick={wstecz}>Wstecz</Przycisk>
      <span class="opis">Krok {krok + 1} z {KROKI.length}</span>
      <Przycisk rodzaj="glowny" typ="submit" ikona={krok === KROKI.length - 1 ? 'ok' : 'strzalka'} wylaczony={trwa}>
        {krok === KROKI.length - 1 ? (trwa ? 'Zapisywanie…' : 'Zakończ') : 'Dalej'}
      </Przycisk>
    </div>
  </form>
</EkranSrodkowy>

<style>
  .kreator {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
  }
  .krok {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .stopka {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-m);
    padding-top: var(--odstep-xl);
    border-top: var(--ramka);
  }
  .podsumowanie {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: var(--odstep-m) var(--odstep-7);
    margin: 0;
  }
  dt {
    color: var(--kolor-opis);
  }
  dd {
    margin: 0;
    font-weight: var(--czcionka-polgruby);
  }
</style>
