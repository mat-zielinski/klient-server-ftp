package dziennik

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
)

// Opcje do OtworzZOpcjami (testy: zegar, mały bufor).
type Opcje struct {
	// Teraz: źródło czasu; nil = time.Now.
	Teraz func() time.Time
	// Bufor: pojemność kanału; 0 = konfiguracja.DziennikBufor.
	Bufor int
	// przedWpisem: haczyk testowy wołany w gorutynie zapisującej przed każdym wpisem.
	przedWpisem func()
}

// zadanie dla gorutyny zapisującej: wpis albo bariera (opróżnij bufor i daj znać).
type zadanie struct {
	wpis    Wpis
	bariera chan struct{}
}

// Dziennik zapisuje i czyta wpisy; bezpieczny dla wielu gorutyn.
type Dziennik struct {
	katalog     string
	teraz       func() time.Time
	przedWpisem func()

	stan       sync.RWMutex // chroni zamkniety i wysyłanie do kanału
	zamkniety  bool
	kanal      chan zadanie
	zakonczono chan struct{}
	bladKonca  error

	przepelnienie atomic.Int64 // wpisy utracone od ostatniego wpisu o przepełnieniu
	bledyZapisu   atomic.Int64

	pisarz pisarz
	pamiec pamiecMiesiecy
}

// pisarz: stan pliku bieżącego miesiąca; używany wyłącznie przez gorutynę zapisującą.
type pisarz struct {
	katalog string
	klucz   string
	plik    *os.File
	bufor   *bufio.Writer
	brudny  bool
}

// Otworz katalog logów (tworzy, gdy brak).
func Otworz(katalog string) (*Dziennik, error) { return OtworzZOpcjami(katalog, Opcje{}) }

// OtworzZOpcjami jak Otworz, z wstrzykniętym zegarem i rozmiarem bufora.
func OtworzZOpcjami(katalog string, o Opcje) (*Dziennik, error) {
	if err := os.MkdirAll(katalog, konfiguracja.UprawnieniaKatalogu); err != nil {
		return nil, fmt.Errorf("Nie można utworzyć katalogu dziennika: %w", err)
	}
	if o.Teraz == nil {
		o.Teraz = time.Now
	}
	if o.Bufor <= 0 {
		o.Bufor = konfiguracja.DziennikBufor
	}
	d := &Dziennik{
		katalog:     katalog,
		teraz:       o.Teraz,
		przedWpisem: o.przedWpisem,
		kanal:       make(chan zadanie, o.Bufor),
		zakonczono:  make(chan struct{}),
		pisarz:      pisarz{katalog: katalog},
		pamiec:      nowaPamiec(konfiguracja.DziennikPamiecMiesiecy),
	}
	go d.petla()
	return d, nil
}

// Zapisz dopisuje wpis (Czas pusty = teraz). Błąd zapisu nie zatrzymuje serwera.
// Nie blokuje: przy pełnym buforze wpis przepada, a liczba utraconych trafia do dziennika.
func (d *Dziennik) Zapisz(w Wpis) {
	if w.Czas.IsZero() {
		w.Czas = d.teraz()
	}
	w.Opis = przytnij(w.Opis, konfiguracja.DziennikOpisMax)
	d.stan.RLock()
	defer d.stan.RUnlock()
	if d.zamkniety {
		return
	}
	select {
	case d.kanal <- zadanie{wpis: w}:
	default:
		d.przepelnienie.Add(1)
	}
}

// Zamknij kończy zapis: opróżnia bufor, fsync, zamyka plik. Kolejne wywołania nic nie robią.
func (d *Dziennik) Zamknij() error {
	d.stan.Lock()
	if d.zamkniety {
		d.stan.Unlock()
		return nil
	}
	d.zamkniety = true
	close(d.kanal)
	d.stan.Unlock()
	<-d.zakonczono
	return d.bladKonca
}

// BledyZapisu: ile razy zapis na dysk się nie udał (diagnostyka).
func (d *Dziennik) BledyZapisu() int64 { return d.bledyZapisu.Load() }

// oproznij czeka, aż wszystkie wcześniejsze wpisy trafią do pliku (dla odczytu).
func (d *Dziennik) oproznij() {
	d.stan.RLock()
	defer d.stan.RUnlock()
	if d.zamkniety {
		return
	}
	b := make(chan struct{})
	d.kanal <- zadanie{bariera: b}
	<-b
}

// petla: gorutyna zapisująca.
func (d *Dziennik) petla() {
	defer close(d.zakonczono)
	tik := time.NewTicker(konfiguracja.DziennikOkresSynchronizacji)
	defer tik.Stop()
	for {
		select {
		case z, ok := <-d.kanal:
			if !ok {
				d.zapiszPrzepelnienie()
				d.bladKonca = d.pisarz.zamknij()
				return
			}
			d.obsluz(z)
		case <-tik.C:
			d.zapiszPrzepelnienie()
			d.policzBlad(d.pisarz.synchronizuj())
		}
	}
}

func (d *Dziennik) obsluz(z zadanie) {
	d.zapiszPrzepelnienie()
	if z.bariera != nil {
		d.policzBlad(d.pisarz.oproznijBufor())
		close(z.bariera)
		return
	}
	if d.przedWpisem != nil {
		d.przedWpisem()
	}
	d.policzBlad(d.pisarz.dopisz(z.wpis))
}

// zapiszPrzepelnienie dopisuje wpis o utraconych wpisach, gdy jakieś przepadły.
func (d *Dziennik) zapiszPrzepelnienie() {
	n := d.przepelnienie.Swap(0)
	if n == 0 {
		return
	}
	d.policzBlad(d.pisarz.dopisz(Wpis{
		Czas:   d.teraz(),
		Rodzaj: RodzajSerwer,
		Wynik:  WynikBlad,
		Opis:   fmt.Sprintf("Przepełnienie bufora dziennika: utracono %d wpisów.", n),
	}))
}

func (d *Dziennik) policzBlad(err error) {
	if err != nil {
		d.bledyZapisu.Add(1)
	}
}

// dopisz zapisuje jedną linię JSON do pliku miesiąca wpisu.
func (p *pisarz) dopisz(w Wpis) error {
	linia, err := json.Marshal(w)
	if err != nil {
		return err
	}
	bladOtwarcia := p.otworzMiesiac(kluczMiesiaca(w.Czas))
	if p.plik == nil {
		return bladOtwarcia
	}
	linia = append(linia, '\n')
	if _, err := p.bufor.Write(linia); err != nil {
		return err
	}
	p.brudny = true
	return bladOtwarcia
}

// otworzMiesiac przełącza plik, gdy wpis należy do innego miesiąca niż otwarty.
func (p *pisarz) otworzMiesiac(klucz string) error {
	if p.plik != nil && p.klucz == klucz {
		return nil
	}
	// Błąd zamknięcia starego pliku nie może zablokować zapisu do nowego: zgłaszamy go razem.
	bladStarego := p.zamknij()
	f, err := os.OpenFile(filepath.Join(p.katalog, nazwaPliku(klucz)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, konfiguracja.UprawnieniaPlikuDziennika)
	if err != nil {
		return errors.Join(bladStarego, err)
	}
	p.plik, p.klucz, p.bufor = f, klucz, bufio.NewWriter(f)
	return bladStarego
}

func (p *pisarz) oproznijBufor() error {
	if p.bufor == nil {
		return nil
	}
	return p.bufor.Flush()
}

// synchronizuj: bufor do pliku i fsync, tylko gdy coś zapisano od ostatniego razu.
func (p *pisarz) synchronizuj() error {
	if p.plik == nil || !p.brudny {
		return nil
	}
	if err := p.bufor.Flush(); err != nil {
		return err
	}
	p.brudny = false
	return p.plik.Sync()
}

func (p *pisarz) zamknij() error {
	if p.plik == nil {
		return nil
	}
	err := p.synchronizuj()
	err = errors.Join(err, p.plik.Close())
	p.plik, p.bufor, p.klucz, p.brudny = nil, nil, "", false
	return err
}

// kluczMiesiaca: "RRRR-MM" w strefie lokalnej.
func kluczMiesiaca(t time.Time) string {
	return t.In(time.Local).Format(konfiguracja.DziennikFormatMiesiaca)
}

func nazwaPliku(klucz string) string {
	return konfiguracja.DziennikPrefiks + klucz + konfiguracja.DziennikRozszerzenie
}

// przytnij skraca tekst do max znaków (nie bajtów), nie psując UTF-8.
func przytnij(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}
