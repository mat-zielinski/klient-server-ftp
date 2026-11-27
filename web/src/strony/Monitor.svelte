<script lang="ts">
  // Monitor serwera (admin), jak stacja Programator_Czarek (wykresy na żywo, kafle) i dashboard UniFi:
  // pasek systemu, wskaźniki CPU / RAM / dysków, kafle procesu, wykresy 10 min (SSE co 1 s) albo 24 h (co 1 min),
  // sieć wg interfejsu, analityka (mapa ciepła pobrań, rodzaje plików, stan indeksu).
  // Dane: GET /api/monitor + SSE /api/monitor/strumien; brak adresu na serwerze = stan pusty, reszta strony działa.
  import { onMount } from 'svelte'
  import Kafel from '../komponenty/wspolne/Kafel.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import WykresLiniowy from '../komponenty/wykresy/WykresLiniowy.svelte'
  import WykresDwuSerii from '../komponenty/monitor/WykresDwuSerii.svelte'
  import PasekSystemu from '../komponenty/monitor/PasekSystemu.svelte'
  import KartaObciazenia from '../komponenty/monitor/KartaObciazenia.svelte'
  import KartaInterfejsow from '../komponenty/monitor/KartaInterfejsow.svelte'
  import KartaMapy from '../komponenty/monitor/KartaMapy.svelte'
  import KartyIndeksu from '../komponenty/monitor/KartyIndeksu.svelte'
  import Segmenty from '../komponenty/monitor/Segmenty.svelte'
  import Szkielet from '../komponenty/monitor/Szkielet.svelte'
  import { StanMonitora, seria, seriaPodwojna } from '../komponenty/monitor/stan.svelte'
  import { MONITOR, OKRESY } from '../komponenty/monitor/stale'
  import { liczbaCala, predkosc, rozmiar } from '../stan/format'
  import { sesja } from '../stan/sesja.svelte'
  import { uklad } from '../stan/uklad.svelte'

  const stan = new StanMonitora()
  let okres = $state<string>(OKRESY[0].id)

  onMount(() => {
    if (!sesja.admin) return
    return stan.start()
  })

  const naZywo = $derived(okres === 'na_zywo')
  const probki = $derived(naZywo ? stan.historia : stan.historiaDoby)
  const oknoMs = $derived(naZywo ? MONITOR.oknoNaZywoMs : undefined)
  const p = $derived(stan.teraz)

  const procentCpu = (v: number) => `${Math.round(v)}%`
  const godzinaMinuta = (t: number) => new Date(t).toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit' })
  const formatDymka = $derived(naZywo ? undefined : godzinaMinuta)

  const seriaCpu = $derived(seria(probki, (x) => x.cpu_proc))
  const seriaRam = $derived(seria(probki, (x) => x.ram_uzyte))
  const seriaSieci = $derived(seriaPodwojna(probki, (x) => x.wysylanie_bps, (x) => x.odbior_bps))
  const seriaDysku = $derived(seriaPodwojna(probki, (x) => x.dysk_odczyt_bps, (x) => x.dysk_zapis_bps))
</script>

{#if !sesja.admin}
  <Pusto ikona="klodka" tytul="Monitor jest dostępny dla administratora" opis="Twoje pobrania znajdziesz w Dzienniku." />
{:else}
  <div class="monitor">
    {#if stan.ladowanie}
      <Szkielet rodzaj="linia" />
      <div class="rzad dwa" class:jedna-kolumna={uklad.waski}>
        <Szkielet rodzaj="wskaznik" ile={3} />
        <Szkielet rodzaj="kafel" ile={4} />
      </div>
      <Szkielet rodzaj="wykres" />
    {:else if stan.brakEndpointu || (!stan.system && stan.blad)}
      <Karta tytul="Monitor serwera" ikona="serwer">
        {#if stan.brakEndpointu}
          <Pusto ikona="serwer" tytul="Monitor serwera jeszcze niedostępny"
            opis="Serwer nie udostępnia jeszcze adresu /api/monitor. Wykresy CPU, pamięci, dysku i sieci pojawią się po aktualizacji serwera." />
        {:else}
          <Pusto ikona="blad" tytul="Nie udało się wczytać monitora" opis={stan.blad} />
        {/if}
      </Karta>
    {:else}
      <PasekSystemu {stan} />

      <div class="rzad dwa" class:jedna-kolumna={uklad.waski}>
        <KartaObciazenia {stan} />
        <div class="siatka-kafli kafle">
          <Kafel etykieta="Połączenia" wartosc={p ? liczbaCala(p.polaczenia) : '—'} ikona="siec"
            stan={p && p.polaczenia > 0 ? 'info' : 'wyl'} opis="Otwarte połączenia klientów" />
          <Kafel etykieta="Aktywne transfery" wartosc={p ? liczbaCala(p.transfery) : '—'} ikona="aktywnosc"
            stan={p && p.transfery > 0 ? 'ok' : 'wyl'} opis={p ? `${predkosc(p.wysylanie_bps)} wysyłania` : undefined} />
          <Kafel etykieta="Gorutyny" wartosc={p ? liczbaCala(p.gorutyny) : '—'} ikona="procesor" opis="Wątki serwera Go" />
          <Kafel etykieta="Pamięć procesu" wartosc={p ? rozmiar(p.pamiec_procesu) : '—'} ikona="baza"
            opis="Pamięć zajęta przez Skrytkę" />
        </div>
      </div>

      <div class="pasek-okresu">
        <h2>Wykresy</h2>
        <Segmenty opcje={OKRESY} bind:value={okres} etykieta="Okres wykresów" />
      </div>
      <div class="siatka-wykresow">
        <Karta tytul="Procesor" ikona="procesor">
          <WykresLiniowy punkty={seriaCpu} {oknoMs} {formatDymka} formatY={procentCpu} nazwa="Procesor" barwa="niebieski" />
        </Karta>
        <Karta tytul="Pamięć RAM" ikona="baza">
          <WykresLiniowy punkty={seriaRam} {oknoMs} {formatDymka} formatY={rozmiar} os="bajty" nazwa="Pamięć RAM" barwa="fioletowy" />
        </Karta>
        <Karta tytul="Sieć" ikona="siec">
          <WykresDwuSerii punkty={seriaSieci} nazwaA="Wysyłanie" nazwaB="Odbiór" formatY={predkosc} {oknoMs} />
        </Karta>
        <Karta tytul="Dysk" ikona="dysk">
          <WykresDwuSerii punkty={seriaDysku} nazwaA="Odczyt" nazwaB="Zapis" barwaA="pomaranczowy" barwaB="rozowy"
            formatY={predkosc} {oknoMs} />
        </Karta>
      </div>

      <KartaInterfejsow {probki} {oknoMs} />
    {/if}

    <h2>Analityka</h2>
    <KartaMapy />
    <div class="rzad dwa" class:jedna-kolumna={uklad.waski}>
      <KartyIndeksu />
    </div>
  </div>
{/if}

<style>
  .monitor {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .rzad {
    display: grid;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .dwa {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    align-items: start;
  }
  .jedna-kolumna {
    grid-template-columns: minmax(0, 1fr);
  }
  .kafle {
    align-content: start;
  }
  .siatka-wykresow {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--uklad-okno-w)), 1fr));
    gap: var(--odstep-7);
  }
  .pasek-okresu {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-5);
    flex-wrap: wrap;
  }
  h2 {
    margin: 0;
    font-size: var(--czcionka-tytul-karty);
    font-weight: var(--czcionka-polgruby);
    letter-spacing: var(--czcionka-rozstaw-etykiety);
    text-transform: uppercase;
    color: var(--kolor-opis);
  }
</style>
