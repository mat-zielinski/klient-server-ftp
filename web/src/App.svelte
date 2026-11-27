<script lang="ts">
  // Powłoka aplikacji: wybór widoku (Publiczny / Kreator / Logowanie / strony z paskiem bocznym), układ jak stacja
  // (pasek boczny po lewej, obszar strony z nagłówkiem). Strony ładowane dynamicznie z ./strony/<Nazwa>.svelte;
  // brak pliku strony = stan pusty (strony robią inni agenci).
  import type { Component } from 'svelte'
  import PasekBoczny, { type PozycjaPaska } from './komponenty/PasekBoczny.svelte'
  import NaglowekStrony from './komponenty/NaglowekStrony.svelte'
  import Karta from './komponenty/wspolne/Karta.svelte'
  import Pusto from './komponenty/wspolne/Pusto.svelte'
  import Przycisk from './komponenty/wspolne/Przycisk.svelte'
  import PasekPostepu from './komponenty/wspolne/PasekPostepu.svelte'
  import Powiadomienia from './komponenty/wspolne/Powiadomienia.svelte'
  import { sesja } from './stan/sesja.svelte'
  import { nawigacja } from './stan/nawigacja.svelte'
  import { pasek } from './stan/pasek.svelte'
  import { powiadom } from './stan/powiadomienia.svelte'
  import { motyw, NAZWY_MOTYWOW, type WyborMotywu } from './stan/motyw.svelte'
  import { skroty, zarejestrujPolecenie, SKROT_POMOCY } from './stan/skroty.svelte'
  import Paleta from './komponenty/paleta/Paleta.svelte'
  import PomocSkrotow from './komponenty/paleta/PomocSkrotow.svelte'

  interface DefinicjaStrony extends PozycjaPaska {
    plik: string
    podtytul: string
    tylkoAdmin: boolean
  }

  // rejestr stron: JEDYNE miejsce, w którym strona jest zdefiniowana (jak STRONY w okno.py stacji)
  const STRONY: DefinicjaStrony[] = [
    { klucz: 'pulpit', etykieta: 'Pulpit', ikona: 'pulpit', plik: 'Pulpit', tylkoAdmin: true,
      podtytul: 'Aktywne transfery, przepustowość i adresy serwera', opis: 'Stan serwera i transfery na żywo' },
    { klucz: 'monitor', etykieta: 'Monitor', ikona: 'wykres', plik: 'Monitor', tylkoAdmin: true,
      podtytul: 'Procesor, pamięć, dysk i sieć na żywo oraz analityka pobrań', opis: 'Zasoby serwera i analityka' },
    { klucz: 'pliki', etykieta: 'Pliki', ikona: 'folder', plik: 'Pliki', tylkoAdmin: false,
      podtytul: 'Przeglądanie, pobieranie i wgrywanie plików', opis: 'Udostępnione foldery' },
    { klucz: 'dziennik', etykieta: 'Dziennik', ikona: 'dziennik', plik: 'Dziennik', tylkoAdmin: false,
      podtytul: 'Historia pobrań i statystyki', opis: 'Kto, co i kiedy pobrał' },
    { klucz: 'udostepnienia', etykieta: 'Udostępnienia', ikona: 'link', plik: 'Udostepnienia', tylkoAdmin: false,
      podtytul: 'Linki publiczne z datą ważności i hasłem', opis: 'Linki publiczne' },
    { klucz: 'kosz', etykieta: 'Kosz', ikona: 'kosz', plik: 'Kosz', tylkoAdmin: false,
      podtytul: 'Usunięte pliki: przywracanie i trwałe usuwanie', opis: 'Usunięte pliki i foldery' },
    { klucz: 'uzytkownicy', etykieta: 'Użytkownicy', ikona: 'uzytkownicy', plik: 'Uzytkownicy', tylkoAdmin: true,
      podtytul: 'Konta, uprawnienia do folderów i dostęp FTPS', opis: 'Konta i uprawnienia' },
    { klucz: 'ustawienia', etykieta: 'Ustawienia', ikona: 'ustawienia', plik: 'Ustawienia', tylkoAdmin: true,
      podtytul: 'Foldery, sieć, porty i certyfikat', opis: 'Konfiguracja serwera' },
  ]

  type Modul = { default: Component }
  const moduly = import.meta.glob<Modul>('./strony/*.svelte')
  const pamiec = new Map<string, Promise<Component | null>>()

  function zaladuj(plik: string): Promise<Component | null> {
    const zapamietana = pamiec.get(plik)
    if (zapamietana) return zapamietana
    const f = moduly[`./strony/${plik}.svelte`]
    const p: Promise<Component | null> = f
      ? f().then(
          (m: Modul) => m.default,
          () => null,
        )
      : Promise.resolve(null)
    pamiec.set(plik, p)
    return p
  }

  const admin = $derived(sesja.admin)
  const dostepne = $derived(STRONY.filter((s) => admin || !s.tylkoAdmin))
  const biezaca = $derived(dostepne.find((s) => s.klucz === nawigacja.strona) ?? null)

  /** Co pokazać: link publiczny, ładowanie, błąd połączenia, kreator, logowanie albo aplikacja. */
  const widok = $derived.by(() => {
    if (nawigacja.publiczny) return 'publiczny'
    if (!sesja.wczytano) return 'ladowanie'
    if (!sesja.stan) return 'blad'
    if (sesja.stan.kreator) return 'kreator'
    if (!sesja.zalogowany) return 'logowanie'
    return 'aplikacja'
  })

  // nieznana lub niedostępna strona: strona startowa (admin: Pulpit, inni: Pliki), bez wpisu w historii
  $effect(() => {
    if (widok === 'aplikacja' && !biezaca) nawigacja.zastap(admin ? 'pulpit' : 'pliki')
  })

  $effect(() => {
    const nazwa = sesja.stan?.nazwa || 'Skrytka'
    document.title = biezaca && widok === 'aplikacja' ? `${biezaca.etykieta} · ${nazwa}` : nazwa
  })

  // polecenia wbudowane (paleta Ctrl+K i skróty), tylko po zalogowaniu
  $effect(() => {
    if (widok !== 'aplikacja') return
    const odinstaluj = skroty.instaluj()
    const wyrejestruj = [
      ...dostepne.map((s, i) =>
        zarejestrujPolecenie({
          id: `strona.${s.klucz}`,
          nazwa: `Przejdź: ${s.etykieta}`,
          ikona: s.ikona,
          skrot: `Alt+${i + 1}`,
          grupa: 'Strony',
          slowa: s.opis,
          akcja: () => nawigacja.idz(s.klucz),
        }),
      ),
      ...(['auto', 'ciemny', 'jasny'] as WyborMotywu[]).map((m) =>
        zarejestrujPolecenie({
          id: `motyw.${m}`,
          nazwa: NAZWY_MOTYWOW[m],
          ikona: m === 'auto' ? 'motyw_auto' : m === 'ciemny' ? 'ksiezyc' : 'slonce',
          grupa: 'Wygląd',
          slowa: 'motyw kolory system',
          akcja: () => motyw.ustaw(m),
        }),
      ),
      zarejestrujPolecenie({
        id: 'menu.zwin',
        nazwa: 'Zwiń lub rozwiń menu',
        ikona: 'pasek_boczny',
        skrot: 'Ctrl+B',
        grupa: 'Wygląd',
        akcja: () => pasek.przelaczZwiniecie(),
      }),
      zarejestrujPolecenie({
        id: 'pomoc.skroty',
        nazwa: 'Skróty klawiszowe',
        ikona: 'klawiatura',
        skrot: SKROT_POMOCY,
        grupa: 'Pomoc',
        akcja: () => skroty.otworzPomoc(),
      }),
      zarejestrujPolecenie({
        id: 'sesja.wyloguj',
        nazwa: 'Wyloguj',
        ikona: 'wyloguj',
        grupa: 'Konto',
        akcja: () => void wyloguj(),
      }),
    ]
    return () => {
      for (const f of wyrejestruj) f()
      odinstaluj()
      skroty.paleta = false
      skroty.pomoc = false
    }
  })

  async function wyloguj() {
    await sesja.wyloguj()
    powiadom('Wylogowano.', 'info')
  }
</script>

{#snippet strona(plik: string, tytul: string, podtytul: string)}
  {#await zaladuj(plik)}
    <div class="ladowanie"><PasekPostepu wartosc={null} etykieta="Wczytywanie strony" /></div>
  {:then Strona}
    {#if Strona}
      <Strona />
    {:else}
      <div class="brak-strony">
        <NaglowekStrony {tytul} {podtytul} />
        <Karta>
          <Pusto ikona="klepsydra" tytul="Strona w przygotowaniu" opis="Ten widok jeszcze nie jest gotowy." />
        </Karta>
      </div>
    {/if}
  {/await}
{/snippet}

{#if widok === 'aplikacja' && sesja.stan?.uzytkownik}
  <div class="uklad">
    <PasekBoczny
      nazwa={sesja.stan.nazwa || 'Skrytka'}
      wersja={sesja.stan.wersja}
      pozycje={dostepne}
      aktywna={biezaca?.klucz ?? ''}
      login={sesja.stan.uzytkownik.login}
      admin={sesja.stan.uzytkownik.admin}
      naWybierz={(k) => nawigacja.idz(k)}
      naWyloguj={wyloguj}
    />
    <main class="obszar" class:zablokowany={pasek.szuflada}>
      <div class="strona">
        {#if biezaca}
          {#key biezaca.klucz}
            {@render strona(biezaca.plik, biezaca.etykieta, biezaca.podtytul)}
          {/key}
        {/if}
      </div>
    </main>
  </div>
{:else if widok === 'publiczny'}
  <main class="samodzielna">{@render strona('Publiczny', 'Udostępniony plik', 'Link publiczny')}</main>
{:else if widok === 'kreator'}
  <main class="samodzielna">{@render strona('Kreator', 'Pierwsze uruchomienie', 'Konto administratora i folder')}</main>
{:else if widok === 'logowanie'}
  <main class="samodzielna">{@render strona('Logowanie', 'Logowanie', 'Zaloguj się, aby zobaczyć pliki')}</main>
{:else if widok === 'blad'}
  <main class="samodzielna srodek">
    <Karta>
      <Pusto ikona="siec" tytul="Brak połączenia z serwerem" opis={sesja.blad}>
        <Przycisk rodzaj="glowny" ikona="odswiez" onclick={() => sesja.odswiez()}>Spróbuj ponownie</Przycisk>
      </Pusto>
    </Karta>
  </main>
{:else}
  <main class="samodzielna srodek">
    <div class="ladowanie"><PasekPostepu wartosc={null} etykieta="Łączenie z serwerem" /></div>
  </main>
{/if}

{#if widok === 'aplikacja'}
  <Paleta />
  <PomocSkrotow />
{/if}
<Powiadomienia />

<style>
  .uklad {
    display: flex;
    height: 100%;
    background: var(--kolor-okno);
  }
  .obszar {
    flex: 1;
    min-width: 0;
    height: 100%;
    overflow: auto;
  }
  .zablokowany {
    overflow: hidden;
  }
  .strona {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
    max-width: var(--uklad-strona-maks-w);
    min-height: 100%;
    margin: 0 auto;
    padding: var(--odstep-xl);
  }
  .brak-strony {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
  }
  .samodzielna {
    height: 100%;
    overflow: auto;
    padding: var(--odstep-xl);
    background: var(--kolor-okno);
  }
  .srodek {
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .ladowanie {
    width: min(var(--uklad-pole-w), 60vw);
    margin: var(--odstep-sekcja) auto;
  }
</style>
