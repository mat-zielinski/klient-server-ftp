<script lang="ts">
  // Dziennik pobrań: zakres i krok, kafle sum, wykres słupkowy bajtów i liczby pobrań, pierścień protokołów,
  // top pliki, top klienci (admin), tabela wpisów z filtrami i eksportem CSV.
  // Zwykły użytkownik widzi tylko własne wpisy (filtruje serwer); interfejs chowa wtedy „kto” i ranking klientów.
  // Na żywo: po zakończonym transferze (SSE) odświeża statystyki i tabelę, gdy zakres obejmuje chwilę obecną.
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import { sluchajTransferow } from '../api/transfery'
  import type { PozycjaRankingu, Statystyki } from '../api/typy'
  import Kafel from '../komponenty/wspolne/Kafel.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Pigulka from '../komponenty/wspolne/Pigulka.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import WykresSlupkowy from '../komponenty/wykresy/WykresSlupkowy.svelte'
  import Pierscien from '../komponenty/wykresy/Pierscien.svelte'
  import ListaRankingu from '../komponenty/wykresy/ListaRankingu.svelte'
  import type { Pozycja, PunktSlupka, Seria } from '../komponenty/wykresy/skala'
  import WyborZakresu from '../komponenty/transfery/WyborZakresu.svelte'
  import TabelaDziennika from '../komponenty/transfery/TabelaDziennika.svelte'
  import { DZIENNIK, PULPIT } from '../komponenty/transfery/stale'
  import { nazwaProtokolu, odmiana } from '../komponenty/transfery/typy'
  import { dataPola, etykietaPunktu, granice as graniceZakresu, opisPunktu, parametryZakresu, type Zakres } from '../komponenty/transfery/zakres'
  import { liczbaCala, rozmiar } from '../stan/format'
  import { sesja } from '../stan/sesja.svelte'
  import { uklad } from '../stan/uklad.svelte'

  const dzis = dataPola(new Date())
  let zakres = $state<Zakres>({ rodzaj: 'dzis', od: dzis, do: dzis, krok: 'godzina' })
  let statystyki = $state<Statystyki | null>(null)
  let blad = $state('')
  let wczytywanie = $state(false)
  /** Licznik odświeżeń: zmiana wymusza ponowne zapytanie tabeli, nawet gdy granice zakresu się nie zmieniły. */
  let odswiezenie = $state(0)
  /** Czy strumień transferów działa (przyszła pierwsza suma). */
  let naZywo = $state(false)

  const admin = $derived(sesja.admin)
  // Granice liczone raz na zmianę zakresu (nie co sekundę), wspólne dla statystyk, tabeli i eksportu.
  const granice = $derived(parametryZakresu(zakres, new Date()))

  $effect(() => {
    const { od, do: doCzasu } = granice
    const krok = zakres.krok
    let aktualne = true
    wczytywanie = true
    api.get<Statystyki>('/dziennik/statystyki', { od, do: doCzasu, krok })
      .then((s) => {
        if (!aktualne) return
        statystyki = s
        blad = ''
      })
      .catch((b) => {
        if (aktualne) blad = trescBledu(b)
      })
      .finally(() => {
        if (aktualne) wczytywanie = false
      })
    return () => {
      aktualne = false
    }
  })

  function odswiez() {
    zakres = { ...zakres }
    odswiezenie += 1
  }

  /** Czy zakres obejmuje chwilę obecną (dla zakresów kończących się „teraz” koniec liczony od nowa). */
  function obejmujeTeraz(): boolean {
    const teraz = new Date()
    const g = graniceZakresu(zakres, teraz)
    return g.od.getTime() <= teraz.getTime() && teraz.getTime() <= g.do.getTime()
  }

  // Koniec transferu zapisuje wpis dziennika zaraz potem: odświeżamy z opóźnieniem, kilka końców = jedno odświeżenie.
  onMount(() => {
    let zegar: ReturnType<typeof setTimeout> | null = null
    const zatrzymaj = sluchajTransferow(
      (z) => {
        if (z.typ !== 'koniec' || zegar) return
        zegar = setTimeout(() => {
          zegar = null
          if (obejmujeTeraz()) odswiez()
        }, PULPIT.poKoncuMs)
      },
      () => {
        naZywo = true
      },
    )
    return () => {
      if (zegar) clearTimeout(zegar)
      zatrzymaj()
    }
  })

  const seriaBajtow: Seria = { nazwa: 'Pobrano', barwa: 'niebieski', format: rozmiar, os: 'bajty' }
  const seriaPobran: Seria = { nazwa: 'Liczba pobrań', barwa: 'zielony', format: liczbaCala, os: 'liczba' }

  const punkty = $derived<PunktSlupka[]>(
    (statystyki?.wykres ?? []).map((p) => ({
      etykieta: etykietaPunktu(p.czas, zakres.krok),
      opis: opisPunktu(p.czas, zakres.krok),
      a: p.bajty,
      b: p.pobrania,
    })),
  )
  const protokoly = $derived<Pozycja[]>(
    Object.entries(statystyki?.protokoly ?? {})
      .map(([nazwa, wartosc]) => ({ nazwa: nazwaProtokolu(nazwa), wartosc }))
      .sort((a, b) => b.wartosc - a.wartosc),
  )

  function ranking(lista: PozycjaRankingu[] | undefined): Pozycja[] {
    return (lista ?? []).slice(0, DZIENNIK.pozycjiRankingu).map((p) => ({
      nazwa: p.nazwa,
      wartosc: p.bajty,
      opis: `${liczbaCala(p.liczba)} ${odmiana(p.liczba, 'pobranie', 'pobrania', 'pobrań')}`,
    }))
  }

  const sumaBajtow = $derived(statystyki?.suma_bajtow ?? 0)
  const sumaPobran = $derived(statystyki?.suma_pobran ?? 0)
  const szczyt = $derived(punkty.reduce<PunktSlupka | null>((a, p) => (!a || p.a > a.a ? p : a), null))
  const liczbaKlientow = $derived(statystyki?.najaktywniejsi_klienci?.length ?? 0)
</script>

<div class="dziennik">
  <div class="pasek-zakresu">
    <WyborZakresu bind:zakres />
    <Przycisk rodzaj="duch" rozmiar="maly" ikona="odswiez" tytul="Odśwież" onclick={odswiez} />
    {#if wczytywanie}<span class="opis">Wczytywanie…</span>{/if}
    <span class="na-zywo">
      <Pigulka stan={naZywo ? 'ok' : 'wyl'} kropka>{naZywo ? 'Na żywo' : 'Łączenie'}</Pigulka>
    </span>
  </div>

  {#if blad}
    <Karta>
      <Pusto ikona="alert" tytul="Nie udało się wczytać statystyk" opis={blad} />
    </Karta>
  {/if}

  <div class="siatka-kafli">
    <Kafel etykieta="Pobrano" wartosc={rozmiar(sumaBajtow)} ikona="pobierz" stan={sumaBajtow > 0 ? 'info' : 'wyl'} />
    <Kafel etykieta="Liczba pobrań" wartosc={liczbaCala(sumaPobran)} ikona="dziennik" />
    <Kafel etykieta="Średnio na pobranie" wartosc={sumaPobran > 0 ? rozmiar(sumaBajtow / sumaPobran) : 'Brak'} ikona="plik" />
    {#if admin}
      <Kafel etykieta="Klienci" wartosc={liczbaCala(liczbaKlientow)} ikona="uzytkownicy"
        opis={liczbaKlientow >= DZIENNIK.pozycjiRankingu ? 'Najaktywniejsi w rankingu' : undefined} />
    {:else}
      <Kafel etykieta="Najwięcej" wartosc={szczyt && szczyt.a > 0 ? rozmiar(szczyt.a) : 'Brak'} ikona="wykres"
        opis={szczyt && szczyt.a > 0 ? szczyt.opis : undefined} />
    {/if}
  </div>

  <Karta tytul={zakres.krok === 'godzina' ? 'Pobrania w kolejnych godzinach' : 'Pobrania w kolejnych dniach'} ikona="wykres">
    {#if punkty.length === 0 && !wczytywanie}
      <Pusto ikona="wykres" tytul="Brak pobrań w tym zakresie" />
    {:else}
      <WykresSlupkowy {punkty} seriaA={seriaBajtow} seriaB={seriaPobran} />
    {/if}
  </Karta>

  <div class="rzad" class:jedna-kolumna={uklad.waski} class:bez-klientow={!admin}>
    <Karta tytul="Protokoły (bajty)" ikona="siec">
      <Pierscien pozycje={protokoly} format={rozmiar} podpisSumy="Pobrano" />
    </Karta>
    <Karta tytul="Najczęściej pobierane" ikona="plik">
      <ListaRankingu pozycje={ranking(statystyki?.najczestsze_pliki)} format={rozmiar} barwa="niebieski" />
    </Karta>
    {#if admin}
      <Karta tytul="Najaktywniejsi klienci" ikona="uzytkownicy">
        <ListaRankingu pozycje={ranking(statystyki?.najaktywniejsi_klienci)} format={rozmiar} barwa="fioletowy" />
      </Karta>
    {/if}
  </div>

  <TabelaDziennika od={granice.od} doCzasu={granice.do} {admin} {odswiezenie} />
</div>

<style>
  .dziennik {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .pasek-zakresu {
    display: flex;
    align-items: center;
    gap: var(--odstep-4);
    flex-wrap: wrap;
  }
  .na-zywo {
    margin-left: auto;
  }
  .rzad {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--odstep-7);
  }
  .rzad.bez-klientow {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .rzad.jedna-kolumna {
    grid-template-columns: minmax(0, 1fr);
  }
</style>
