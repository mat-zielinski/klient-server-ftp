<script lang="ts">
  // Pasek informacji o maszynie (jak nagłówek urządzenia w UniFi): system i architektura, host, rdzenie, RAM,
  // czas pracy liczony na żywo i stan połączenia na żywo.
  import Ikona from '../wspolne/Ikona.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import { czasTrwania, liczbaCala, rozmiar } from '../../stan/format'
  import type { StanMonitora } from './stan.svelte'

  interface Props { stan: StanMonitora }
  let { stan }: Props = $props()

  const NAZWY_OS: Record<string, string> = { windows: 'Windows', linux: 'Linux', darwin: 'macOS', freebsd: 'FreeBSD' }

  const pola = $derived.by((): { ikona: NazwaIkony; etykieta: string; wartosc: string }[] => {
    const s = stan.system
    if (!s) return []
    const czas = stan.czasPracy
    return [
      { ikona: 'serwer', etykieta: 'Host', wartosc: s.host || '—' },
      { ikona: 'terminal', etykieta: 'System', wartosc: `${NAZWY_OS[s.os] ?? s.os} ${s.arch}`.trim() },
      { ikona: 'procesor', etykieta: 'Rdzenie', wartosc: liczbaCala(s.cpu_rdzenie) },
      { ikona: 'baza', etykieta: 'Pamięć RAM', wartosc: rozmiar(s.ram_razem) },
      { ikona: 'zegar', etykieta: 'Czas pracy', wartosc: czas === null ? '—' : czasTrwania(czas) },
    ]
  })
</script>

<div class="pasek">
  <ul class="pola">
    {#each pola as p (p.etykieta)}
      <li>
        <span class="ikona"><Ikona nazwa={p.ikona} rozmiar="s" /></span>
        <span class="etykieta">{p.etykieta}</span>
        <span class="wartosc" title={p.wartosc}>{p.wartosc}</span>
      </li>
    {/each}
  </ul>
  <Pigulka stan={stan.polaczono ? 'ok' : 'wyl'} kropka>{stan.polaczono ? 'Na żywo' : 'Łączenie'}</Pigulka>
</div>

<style>
  .pasek {
    display: flex;
    align-items: center;
    gap: var(--odstep-5);
    padding: var(--odstep-l) var(--odstep-xl);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
    min-width: 0;
  }
  .pola {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-3) var(--odstep-7);
    list-style: none;
    margin: 0;
    padding: 0;
    flex: 1;
    min-width: 0;
  }
  li {
    display: inline-grid;
    grid-template-columns: auto auto;
    column-gap: var(--odstep-2);
    align-items: center;
    min-width: 0;
  }
  .ikona {
    grid-row: span 2;
    display: inline-flex;
    color: var(--kolor-niebieski);
  }
  .etykieta {
    font-size: var(--czcionka-etykieta);
    font-weight: var(--czcionka-polgruby);
    letter-spacing: var(--czcionka-rozstaw-etykiety);
    text-transform: uppercase;
    color: var(--kolor-opis);
  }
  .wartosc {
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-tekst);
    color: var(--kolor-tekst);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
