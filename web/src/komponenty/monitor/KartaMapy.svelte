<script lang="ts">
  // Mapa ciepła pobrań (GET /api/dziennik/mapa): dzień tygodnia × godzina z wybranego zakresu dni.
  // Ładowanie = szkielet, brak adresu na serwerze = stan pusty z opisem, inny błąd = stan pusty z komunikatem.
  import { BladApi, trescBledu } from '../../api/klient'
  import { pobierzMape, type MapaPobran } from '../../api/monitor'
  import Karta from '../wspolne/Karta.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import MapaCiepla from '../wykresy/MapaCiepla.svelte'
  import { rozmiar } from '../../stan/format'
  import Segmenty from './Segmenty.svelte'
  import Szkielet from './Szkielet.svelte'
  import { MS_DZIEN, ZAKRESY_MAPY } from './stale'

  let zakres = $state(ZAKRESY_MAPY[0].id)
  let mapa = $state.raw<MapaPobran | null>(null)
  let ladowanie = $state(true)
  let blad = $state('')
  let brakEndpointu = $state(false)

  $effect(() => {
    const dni = ZAKRESY_MAPY.find((z) => z.id === zakres)?.dni ?? ZAKRESY_MAPY[0].dni
    const teraz = new Date()
    const poczatekDzis = new Date(teraz.getFullYear(), teraz.getMonth(), teraz.getDate()).getTime()
    const od = new Date(poczatekDzis - (dni - 1) * MS_DZIEN).toISOString()
    let aktualne = true
    ladowanie = true
    pobierzMape(od, teraz.toISOString())
      .then((m) => {
        if (!aktualne) return
        mapa = m
        blad = ''
        brakEndpointu = false
      })
      .catch((b) => {
        if (!aktualne) return
        brakEndpointu = b instanceof BladApi && b.kod === 404
        blad = trescBledu(b)
      })
      .finally(() => {
        if (aktualne) ladowanie = false
      })
    return () => {
      aktualne = false
    }
  })
</script>

<Karta tytul="Kiedy pobierano" ikona="kalendarz">
  {#snippet akcje()}
    <Segmenty opcje={ZAKRESY_MAPY} bind:value={zakres} etykieta="Zakres mapy pobrań" />
  {/snippet}
  {#if ladowanie && !mapa}
    <Szkielet rodzaj="wykres" />
  {:else if brakEndpointu}
    <Pusto ikona="kalendarz" tytul="Mapa pobrań jeszcze niedostępna"
      opis="Serwer nie udostępnia jeszcze adresu /api/dziennik/mapa. Mapa pojawi się po aktualizacji serwera." />
  {:else if blad && !mapa}
    <Pusto ikona="blad" tytul="Nie udało się wczytać mapy pobrań" opis={blad} />
  {:else}
    <MapaCiepla komorki={mapa?.komorki ?? []} format={rozmiar} nazwa="Pobrano" />
  {/if}
</Karta>
