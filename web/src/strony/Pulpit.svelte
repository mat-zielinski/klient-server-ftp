<script lang="ts">
  // Pulpit (admin): kafle (aktywne, łączna prędkość, dziś pobrano, czas pracy), wykres prędkości łącznej na żywo
  // z 5 minut, tabela aktywnych transferów z przerywaniem, adresy serwera i zajętość udziałów.
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../api/klient'
  import type { Pulpit } from '../api/typy'
  import Kafel from '../komponenty/wspolne/Kafel.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Pigulka from '../komponenty/wspolne/Pigulka.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import WykresLiniowy from '../komponenty/wykresy/WykresLiniowy.svelte'
  import { NA_ZYWO } from '../komponenty/wykresy/stale'
  import TabelaTransferow from '../komponenty/transfery/TabelaTransferow.svelte'
  import KartaAdresow from '../komponenty/transfery/KartaAdresow.svelte'
  import KartaUdzialow from '../komponenty/transfery/KartaUdzialow.svelte'
  import { MonitorTransferow } from '../komponenty/transfery/monitor.svelte'
  import { PULPIT } from '../komponenty/transfery/stale'
  import { odmiana, type InfoSieci } from '../komponenty/transfery/typy'
  import { czasTrwania, liczbaCala, predkosc, rozmiar } from '../stan/format'
  import { sesja } from '../stan/sesja.svelte'
  import { uklad } from '../stan/uklad.svelte'

  const monitor = new MonitorTransferow()
  let pulpit = $state<Pulpit | null>(null)
  let bladPulpitu = $state('')
  let siec = $state<InfoSieci | null>(null)
  let bladSieci = $state('')
  let wczytanoO = $state(Date.now())
  let teraz = $state(Date.now())

  async function wczytajPulpit() {
    try {
      pulpit = await api.get<Pulpit>('/pulpit')
      wczytanoO = Date.now()
      bladPulpitu = ''
    } catch (b) {
      bladPulpitu = trescBledu(b)
    }
  }

  async function wczytajSiec() {
    try {
      siec = await api.get<InfoSieci>('/siec')
      bladSieci = ''
    } catch (b) {
      bladSieci = trescBledu(b)
    }
  }

  onMount(() => {
    if (!sesja.admin) return
    const stop = monitor.start()
    void wczytajPulpit()
    void wczytajSiec()
    const odswiez = setInterval(wczytajPulpit, PULPIT.odswiezanieMs)
    const zegar = setInterval(() => (teraz = Date.now()), NA_ZYWO.krokMs)
    return () => {
      stop()
      clearInterval(odswiez)
      clearInterval(zegar)
    }
  })

  // Zakończony transfer zmienia liczniki „dziś”: odśwież po chwili (kilka zakończeń = jedno żądanie).
  $effect(() => {
    if (monitor.zakonczone === 0) return
    const t = setTimeout(wczytajPulpit, PULPIT.poKoncuMs)
    return () => clearTimeout(t)
  })

  const aktywne = $derived(monitor.polaczono ? monitor.suma.aktywne : (pulpit?.aktywne ?? 0))
  const predkoscLaczna = $derived(monitor.polaczono ? monitor.suma.predkosc_bps : (pulpit?.predkosc_bps ?? 0))
  const szczyt = $derived(monitor.seria.reduce((a, p) => Math.max(a, p.v), 0))
  const wgrania = $derived(monitor.transfery.filter((t) => t.kierunek === 'wgranie').length)
  const pobrania = $derived(monitor.transfery.length - wgrania)
  const czasPracy = $derived(pulpit ? pulpit.czas_pracy_s + Math.max(0, (teraz - wczytanoO) / 1000) : null)
  const dzisPobrania = $derived(pulpit?.dzis.pobrania ?? 0)
</script>

{#if !sesja.admin}
  <Pusto ikona="klodka" tytul="Pulpit jest dostępny dla administratora" opis="Twoje pobrania znajdziesz w Dzienniku." />
{:else}
  <div class="pulpit">
    <div class="siatka-kafli">
      <Kafel etykieta="Aktywne transfery" wartosc={liczbaCala(aktywne)} ikona="aktywnosc" stan={aktywne > 0 ? 'info' : 'wyl'}
        opis="{pobrania} {odmiana(pobrania, 'pobranie', 'pobrania', 'pobrań')}, {wgrania} {odmiana(wgrania, 'wgranie', 'wgrania', 'wgrań')}" />
      <Kafel etykieta="Łączna prędkość" wartosc={predkosc(predkoscLaczna)} ikona="predkosc" stan={predkoscLaczna > 0 ? 'ok' : 'wyl'}
        opis="Szczyt z 5 min: {predkosc(szczyt)}" />
      <Kafel etykieta="Dziś pobrano" wartosc={pulpit ? rozmiar(pulpit.dzis.bajty) : 'Brak'} ikona="pobierz"
        opis="{liczbaCala(dzisPobrania)} {odmiana(dzisPobrania, 'pobranie', 'pobrania', 'pobrań')}" />
      <Kafel etykieta="Czas pracy" wartosc={czasPracy === null ? 'Brak' : czasTrwania(czasPracy)} ikona="zegar"
        opis={bladPulpitu || 'Od uruchomienia serwera'} stan={bladPulpitu ? 'blad' : undefined} />
    </div>

    <div class="rzad glowny" class:jedna-kolumna={uklad.waski}>
      <Karta tytul="Prędkość łączna" ikona="wykres">
        {#snippet akcje()}
          <Pigulka stan={monitor.polaczono ? 'ok' : 'wyl'} kropka>{monitor.polaczono ? 'Na żywo' : 'Łączenie'}</Pigulka>
        {/snippet}
        <WykresLiniowy punkty={monitor.seria} oknoMs={NA_ZYWO.oknoMs} formatY={predkosc} os="bajty" nazwa="Prędkość łączna" />
      </Karta>
      <KartaAdresow {siec} blad={bladSieci} />
    </div>

    <Karta tytul="Aktywne transfery" ikona="aktywnosc" bezMarginesu>
      {#snippet akcje()}
        <span class="licznik">{monitor.transfery.length}</span>
      {/snippet}
      <TabelaTransferow {monitor} />
    </Karta>

    <KartaUdzialow udzialy={pulpit?.udzialy ?? []} />
  </div>
{/if}

<style>
  .pulpit {
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
  .glowny {
    grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  }
  .glowny.jedna-kolumna {
    grid-template-columns: minmax(0, 1fr);
  }
</style>
