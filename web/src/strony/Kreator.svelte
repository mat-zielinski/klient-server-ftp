<script lang="ts">
  // Kreator pierwszego uruchomienia (POST /api/kreator, tylko gdy brak kont): 1 konto administratora (hasło 2x,
  // min. długość z stale.ts, siła i generator), 2 pierwszy udział (nazwa, folder na dysku, tylko do odczytu),
  // 3 tryb sieci (VPN zalecany albo Publiczny), 4 podsumowanie i „Uruchom serwer”. Serwer zakłada konto i loguje.
  // Walidacja „na dotknięciu”: błąd pola dopiero po jego opuszczeniu albo po „Dalej”; „Dalej” przy błędzie
  // przenosi fokus na pierwsze złe pole.
  import { tick } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import EkranSrodkowy from '../komponenty/admin/EkranSrodkowy.svelte'
  import Komunikat from '../komponenty/admin/Komunikat.svelte'
  import KrokiKreatora from '../komponenty/admin/KrokiKreatora.svelte'
  import PodsumowanieKreatora from '../komponenty/admin/PodsumowanieKreatora.svelte'
  import WyborTrybu from '../komponenty/admin/WyborTrybu.svelte'
  import { KREATOR, MIN_DLUGOSC_HASLA } from '../komponenty/admin/stale'
  import type { DaneKreatora, TrybSieci } from '../komponenty/admin/typy'
  import Pole from '../komponenty/wspolne/Pole.svelte'
  import PoleHasla from '../komponenty/wspolne/PoleHasla.svelte'
  import Przelacznik from '../komponenty/wspolne/Przelacznik.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'
  import { uklad } from '../stan/uklad.svelte'

  type NazwaPola = 'login' | 'haslo' | 'haslo2' | 'nazwa' | 'sciezka'

  const KROKI = ['Konto administratora', 'Pierwszy udział', 'Tryb sieci', 'Podsumowanie']
  const POLA_KROKU: NazwaPola[][] = [['login', 'haslo', 'haslo2'], ['nazwa', 'sciezka'], [], []]
  const PRZYKLADY_SCIEZEK = [
    { system: 'Windows', sciezki: ['D:\\Udostepnione', 'C:\\Users\\Jan\\Filmy'] },
    { system: 'Linux', sciezki: ['/srv/skrytka', '/home/jan/pliki'] },
  ]

  let formularz = $state<HTMLFormElement>()
  let krok = $state(0)
  /** Kierunek ostatniego przejścia (1 dalej, -1 wstecz): animacja wjeżdża z właściwej strony. */
  let kierunek = $state(1)
  let login = $state('')
  let haslo = $state('')
  let haslo2 = $state('')
  let pokazHasla = $state(false)
  let nazwaUdzialu = $state(KREATOR.nazwaUdzialu)
  let sciezka = $state(KREATOR.sciezkaUdzialu)
  let tylkoOdczyt = $state(KREATOR.tylkoOdczyt)
  let tryb = $state<TrybSieci>(KREATOR.tryb)
  let dotkniete = $state<Partial<Record<NazwaPola, boolean>>>({})
  let trwa = $state(false)
  let bladSerwera = $state('')

  const bledy = $derived<Record<NazwaPola, string>>({
    login: login.trim() ? '' : 'Podaj login.',
    haslo: haslo.length >= MIN_DLUGOSC_HASLA ? '' : `Hasło musi mieć co najmniej ${MIN_DLUGOSC_HASLA} znaków.`,
    haslo2: !haslo2 ? 'Powtórz hasło.' : haslo2 !== haslo ? 'Hasła nie są takie same.' : '',
    nazwa: nazwaUdzialu.trim() ? '' : 'Podaj nazwę udziału.',
    sciezka: sciezka.trim() ? '' : 'Podaj ścieżkę folderu na dysku.',
  })

  const haslaZgodne = $derived(!!haslo2 && haslo2 === haslo && haslo.length >= MIN_DLUGOSC_HASLA)
  const ostatni = $derived(krok === KROKI.length - 1)

  /** Błąd do pokazania: pole opuszczone albo „Dalej”; niezgodność haseł także, gdy powtórzenie jest już pełne. */
  function bladPola(p: NazwaPola): string | undefined {
    if (!bledy[p]) return undefined
    if (dotkniete[p]) return bledy[p]
    if (p === 'haslo2' && haslo2 && haslo2.length >= haslo.length) return bledy[p]
    return undefined
  }

  function dotknij(p: NazwaPola) {
    dotkniete[p] = true
  }

  async function dalej(e?: Event) {
    e?.preventDefault()
    if (trwa) return
    const pola = POLA_KROKU[krok] ?? []
    if (pola.some((p) => bledy[p])) {
      for (const p of pola) dotkniete[p] = true
      await tick()
      formularz?.querySelector<HTMLInputElement>('input[aria-invalid="true"]')?.focus()
      return
    }
    if (ostatni) {
      void zakoncz()
      return
    }
    kierunek = 1
    krok++
  }

  function idzDo(i: number) {
    if (i >= krok || trwa) return
    bladSerwera = ''
    kierunek = -1
    krok = i
  }

  function wygenerowano(h: string) {
    haslo2 = h
    pokazHasla = true
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

<EkranSrodkowy podtytul="Pierwsze uruchomienie: kilka kroków do działającego serwera" szeroka>
  <form class="kreator" bind:this={formularz} onsubmit={dalej} novalidate>
    <KrokiKreatora kroki={KROKI} biezacy={krok} onwybierz={idzDo} zwarte={uklad.telefon} />

    {#key krok}
      <section class="krok" style:--kierunek={kierunek}>
        {#if krok === 0}
          <header>
            <h2>Konto administratora</h2>
            <p class="opis">Administrator zakłada konta, udostępnia foldery i zmienia ustawienia serwera.</p>
          </header>
          <Pole etykieta="Login" bind:value={login} ikona="uzytkownik" autocomplete="username" autofocus duze
            podpowiedz="np. {KREATOR.login}" blad={bladPola('login')} onblur={() => dotknij('login')} />
          <PoleHasla etykieta="Hasło" bind:value={haslo} bind:pokaz={pokazHasla} autocomplete="new-password"
            sila generator minDlugosc={MIN_DLUGOSC_HASLA} onwygeneruj={wygenerowano}
            opis="Co najmniej {MIN_DLUGOSC_HASLA} znaków. Najlepiej długie zdanie albo wygenerowane hasło."
            blad={bladPola('haslo')} onblur={() => dotknij('haslo')} />
          <PoleHasla etykieta="Powtórz hasło" bind:value={haslo2} bind:pokaz={pokazHasla} autocomplete="new-password"
            zgodne={haslaZgodne} blad={bladPola('haslo2')} onblur={() => dotknij('haslo2')} />
        {:else if krok === 1}
          <header>
            <h2>Pierwszy udział</h2>
            <p class="opis">Folder na tym komputerze, który zobaczą klienci. Kolejne dodasz w Ustawieniach.</p>
          </header>
          <Pole etykieta="Nazwa udziału" bind:value={nazwaUdzialu} ikona="folder" autofocus duze
            podpowiedz="np. Filmy" opis="Tak udział zobaczą klienci." blad={bladPola('nazwa')}
            onblur={() => dotknij('nazwa')} />
          <div class="sciezka">
            <Pole etykieta="Ścieżka folderu na dysku" bind:value={sciezka} ikona="dysk" mono duze
              podpowiedz="Pełna ścieżka folderu" blad={bladPola('sciezka')} onblur={() => dotknij('sciezka')} />
            <div class="przyklady">
              {#each PRZYKLADY_SCIEZEK as p (p.system)}
                <div class="przyklad">
                  <span class="system">{p.system}</span>
                  {#each p.sciezki as s (s)}
                    <button type="button" class="wzor mono" title="Wstaw {s}" onclick={() => (sciezka = s)}>{s}</button>
                  {/each}
                </div>
              {/each}
            </div>
          </div>
          <Przelacznik etykieta="Tylko do odczytu" bind:value={tylkoOdczyt}
            opis="Nikt nie wgra, nie zmieni ani nie usunie plików w tym udziale, także administrator." />
        {:else if krok === 2}
          <header>
            <h2>Tryb sieci</h2>
            <p class="opis">Jak klienci dostaną się do serwera spoza domu. Zmienisz to później w Ustawieniach.</p>
          </header>
          <WyborTrybu bind:value={tryb} />
        {:else}
          <header>
            <h2>Wszystko gotowe</h2>
            <p class="opis">Sprawdź ustawienia. Każde zmienisz jeszcze teraz albo później w Ustawieniach.</p>
          </header>
          <PodsumowanieKreatora login={login.trim()} nazwa={nazwaUdzialu.trim()} sciezka={sciezka.trim()}
            {tylkoOdczyt} {tryb} onzmien={idzDo} />
          {#if bladSerwera}<Komunikat rodzaj="blad">{bladSerwera}</Komunikat>{/if}
        {/if}
      </section>
    {/key}

    <div class="stopka">
      {#if krok > 0}
        <Przycisk rodzaj="duch" ikona="strzalka_lewo" wylaczony={trwa} onclick={() => idzDo(krok - 1)}>Wstecz</Przycisk>
      {:else}
        <span></span>
      {/if}
      <Przycisk rodzaj="glowny" typ="submit" rozmiar="duzy" ikona={ostatni ? 'rakieta' : 'strzalka'} wylaczony={trwa}>
        {ostatni ? (trwa ? 'Uruchamianie…' : 'Uruchom serwer') : 'Dalej'}
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
    animation: wjazd var(--czas-krok) var(--krzywa-wejscia) both;
  }
  header {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    margin-bottom: var(--odstep-xs);
  }
  h2 {
    margin: 0;
  }
  header .opis {
    margin: 0;
  }
  .sciezka {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
  }
  .przyklady {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-xs) var(--odstep-7);
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
  }
  .przyklad {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--odstep-xs);
  }
  .system {
    font-weight: var(--czcionka-polgruby);
  }
  .wzor {
    padding: var(--odstep-pozycje) var(--odstep-s);
    border: var(--ramka);
    border-radius: var(--promien-segment);
    background: var(--kolor-karta);
    color: var(--kolor-tekst);
    font-size: var(--czcionka-drobny);
    cursor: pointer;
  }
  .wzor:hover {
    border-color: var(--kolor-niebieski);
    color: var(--kolor-niebieski);
  }
  .stopka {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-m);
    padding-top: var(--odstep-xl);
    border-top: var(--ramka);
  }
  @keyframes wjazd {
    from {
      opacity: 0;
      transform: translateX(calc(var(--przesuniecie-krok) * var(--kierunek)));
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .krok {
      animation: none;
    }
  }
</style>
