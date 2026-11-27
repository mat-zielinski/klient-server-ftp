// Stałe Pulpitu i Dziennika (czasy odświeżania, limity, progi) w jednym miejscu.

export const PULPIT = {
  /** Co ile odświeżać kafle „dziś pobrano” i zajętość udziałów (GET /api/pulpit). */
  odswiezanieMs: 30_000,
  /** Opóźnienie odświeżenia po zakończonym transferze (zbiera kilka zakończeń w jedno żądanie). */
  poKoncuMs: 1500,
  /** Zajętość udziału: od tego progu pasek ostrzega, od drugiego alarmuje. */
  progOstrzezenia: 0.8,
  progAlarmu: 0.95,
} as const

export const DZIENNIK = {
  /** Najwięcej wpisów w tabeli jednego zapytania. */
  limitWpisow: 500,
  /** Ile pozycji w rankingach. */
  pozycjiRankingu: 8,
  /** Zwłoka po wpisaniu filtra tekstowego, zanim pójdzie zapytanie. */
  zwlokaSzukaniaMs: 350,
  /** Dni wstecz dla zakresów (wliczając dziś). */
  dniTydzien: 7,
  dniMiesiac: 30,
} as const

export const MS_DZIEN = 24 * 60 * 60 * 1000
