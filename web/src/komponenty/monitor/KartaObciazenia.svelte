<script lang="ts">
  // Wskaźniki obciążenia: CPU %, RAM % (z bieżącej próbki) i zajętość dysków udziałów (GET /api/pulpit, suma
  // wszystkich udziałów, odświeżana co minutę). Brak danych = pusty wskaźnik z „—”, bez błędu strony.
  import { onMount } from 'svelte'
  import { api, trescBledu } from '../../api/klient'
  import type { Pulpit } from '../../api/typy'
  import Karta from '../wspolne/Karta.svelte'
  import Wskaznik from '../wykresy/Wskaznik.svelte'
  import { procent, rozmiar } from '../../stan/format'
  import type { StanMonitora } from './stan.svelte'
  import { udzial } from './stan.svelte'
  import { MONITOR } from './stale'

  interface Props { stan: StanMonitora }
  let { stan }: Props = $props()

  let udzialy = $state<Pulpit['udzialy'] | null>(null)
  let bladDysku = $state('')

  async function wczytajDyski() {
    try {
      udzialy = (await api.get<Pulpit>('/pulpit')).udzialy ?? []
      bladDysku = ''
    } catch (b) {
      bladDysku = trescBledu(b)
    }
  }

  onMount(() => {
    void wczytajDyski()
    const zegar = setInterval(wczytajDyski, MONITOR.odswiezanieDyskuMs)
    return () => clearInterval(zegar)
  })

  const progi: [number, number] = [MONITOR.progOstrzezenia, MONITOR.progAlarmu]
  const p = $derived(stan.teraz)
  const cpu = $derived(p ? Math.min(1, Math.max(0, p.cpu_proc / 100)) : null)
  const ramRazem = $derived(p?.ram_razem || stan.system?.ram_razem || 0)
  const ram = $derived(p && ramRazem > 0 ? udzial(p.ram_uzyte, ramRazem) : null)
  const dysk = $derived.by(() => {
    if (!udzialy || udzialy.length === 0) return null
    const razem = udzialy.reduce((a, u) => a + u.razem, 0)
    const zajete = udzialy.reduce((a, u) => a + Math.max(0, u.razem - u.wolne), 0)
    return razem > 0 ? { ulamek: udzial(zajete, razem), zajete, razem } : null
  })
  const opisDysku = $derived(
    dysk ? `${rozmiar(dysk.zajete)} z ${rozmiar(dysk.razem)}`
    : bladDysku || (udzialy && udzialy.length === 0 ? 'Brak udziałów' : 'Wczytywanie'),
  )
</script>

<Karta tytul="Obciążenie" ikona="aktywnosc">
  <div class="wskazniki">
    <Wskaznik etykieta="Procesor" wartosc={cpu} {progi}
      opis={stan.system ? `${stan.system.cpu_rdzenie} rdzeni` : undefined} />
    <Wskaznik etykieta="Pamięć RAM" wartosc={ram} {progi} barwa="fioletowy"
      opis={p && ramRazem ? `${rozmiar(p.ram_uzyte)} z ${rozmiar(ramRazem)}` : undefined} />
    <Wskaznik etykieta="Dyski udziałów" wartosc={dysk?.ulamek ?? null} {progi} barwa="zielony"
      opis={opisDysku} tekst={dysk ? procent(dysk.ulamek) : undefined} />
  </div>
</Karta>

<style>
  .wskazniki {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(var(--kolumna-l), 1fr));
    gap: var(--odstep-7);
    align-items: start;
  }
</style>
