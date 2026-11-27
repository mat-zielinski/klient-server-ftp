package ftp

import (
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

func TestRozloz(t *testing.T) {
	testy := []struct {
		wej     string
		udzial  string
		sciezka string
		blad    bool
	}{
		{"/", "", "/", false},
		{"", "", "/", false},
		{"/publiczny", "publiczny", "/", false},
		{"/publiczny/a/b.txt", "publiczny", "/a/b.txt", false},
		{"publiczny/a", "publiczny", "/a", false},
		{"/publiczny/../tajny/x", "tajny", "/x", false}, // path.Clean, potem bramka dla „tajny”
		{"/../../etc/passwd", "etc", "/passwd", false},  // nie wychodzi ponad korzeń wirtualny
		{"\\publiczny\\a.txt", "publiczny", "/a.txt", false},
		{"/publiczny/C:/Windows", "", "", true},
		{"/publiczny/a.txt:strumien", "", "", true},
	}
	for _, tt := range testy {
		c, err := rozloz(tt.wej)
		if (err != nil) != tt.blad {
			t.Fatalf("%q: błąd %v, oczekiwano błędu=%v", tt.wej, err, tt.blad)
		}
		if err == nil && (c.udzial != tt.udzial || c.sciezka != tt.sciezka) {
			t.Errorf("%q: %+v, oczekiwano %s %s", tt.wej, c, tt.udzial, tt.sciezka)
		}
	}
}

func TestTlumacz(t *testing.T) {
	testy := []struct {
		wej error
		cel error
	}{
		{rdzen.ErrNieZnaleziono, os.ErrNotExist},
		{rdzen.ErrZlaNazwa, ftpserver.ErrFileNameNotAllowed},
		{rdzen.ErrPozaUdzialem, ftpserver.ErrFileNameNotAllowed},
		{rdzen.ErrBrakPrawa, os.ErrPermission},
		{rdzen.ErrTylkoOdczyt, os.ErrPermission},
	}
	for _, tt := range testy {
		e := tlumacz(tt.wej)
		if !errors.Is(e, tt.cel) || !errors.Is(e, tt.wej) {
			t.Errorf("%v: %v nie zawiera obu błędów", tt.wej, e)
		}
	}
	if tlumacz(nil) != nil {
		t.Error("nil musi zostać nil")
	}
}

func nazwy(lista []os.FileInfo) string {
	var n []string
	for _, i := range lista {
		n = append(n, i.Name())
	}
	sort.Strings(n)
	return strings.Join(n, ",")
}

func TestKorzenPokazujeTylkoUdzialyZPrawem(t *testing.T) {
	s := noweSrodowisko(t)
	lista, err := s.kl.ReadDir("/")
	if err != nil {
		t.Fatal(err)
	}
	if got := nazwy(lista); got != "archiwum,publiczny" {
		t.Fatalf("korzeń: %s", got)
	}
}

func TestOdczytBramka(t *testing.T) {
	s := noweSrodowisko(t)
	testy := []struct {
		sciezka string
		cel     error
	}{
		{"/", nil},
		{"/publiczny", nil},
		{"/publiczny/a.txt", nil},
		{"/tajny", os.ErrNotExist}, // ukryty udział = nie istnieje
		{"/tajny/x.txt", os.ErrNotExist},
		{"/brak", os.ErrNotExist},
		{"/publiczny/../tajny", os.ErrNotExist},
	}
	for _, tt := range testy {
		_, err := s.kl.Stat(tt.sciezka)
		if !errors.Is(err, tt.cel) && !(tt.cel == nil && err == nil) {
			t.Errorf("Stat %s: %v, oczekiwano %v", tt.sciezka, err, tt.cel)
		}
		_, err = s.kl.ReadDir(tt.sciezka)
		if tt.cel != nil && err == nil {
			t.Errorf("ReadDir %s: brak błędu", tt.sciezka)
		}
	}
}

func TestZapisBramka(t *testing.T) {
	testy := []struct {
		nazwa string
		op    func(k *klient) error
		cel   error
	}{
		{"mkdir w korzeniu", func(k *klient) error { return k.Mkdir("/nowy", 0) }, os.ErrPermission},
		{"mkdir udział", func(k *klient) error { return k.Mkdir("/publiczny", 0) }, os.ErrPermission},
		{"mkdir tajny", func(k *klient) error { return k.Mkdir("/tajny/x", 0) }, os.ErrNotExist},
		{"mkdir tylko odczyt", func(k *klient) error { return k.Mkdir("/archiwum/x", 0) }, os.ErrPermission},
		{"mkdir zła nazwa", func(k *klient) error { return k.Mkdir("/publiczny/CON", 0) }, ftpserver.ErrFileNameNotAllowed},
		{"mkdir ok", func(k *klient) error { return k.Mkdir("/publiczny/nowy", 0) }, nil},
		{"usuń udział", func(k *klient) error { return k.Remove("/publiczny") }, os.ErrPermission},
		{"usuń w tylko odczyt", func(k *klient) error { return k.Remove("/archiwum/b.txt") }, os.ErrPermission},
		{"DELE na folderze", func(k *klient) error { return k.Remove("/publiczny/f") }, errToFolder},
		{"RMD na pliku", func(k *klient) error { return k.RemoveDir("/publiczny/a.txt") }, errToPlik},
		{"RMD niepusty", func(k *klient) error { return k.RemoveDir("/publiczny/pelny") }, errFolderNiePusty},
		{"rename między udziałami", func(k *klient) error { return k.Rename("/publiczny/a.txt", "/archiwum/a.txt") }, errInnyUdzial},
		{"rename udziału", func(k *klient) error { return k.Rename("/publiczny", "/publiczny2") }, errInnyUdzial},
		{"rename zła nazwa", func(k *klient) error { return k.Rename("/publiczny/a.txt", "/publiczny/a?.txt") }, ftpserver.ErrFileNameNotAllowed},
		{"RemoveAll", func(k *klient) error { return k.RemoveAll("/publiczny/f") }, errNieObslugiwane},
		{"Chmod", func(k *klient) error { return k.Chmod("/publiczny/a.txt", 0o777) }, errNieObslugiwane},
	}
	for _, tt := range testy {
		s := noweSrodowisko(t)
		err := tt.op(s.kl)
		if tt.cel == nil && err != nil || tt.cel != nil && !errors.Is(err, tt.cel) {
			t.Errorf("%s: %v, oczekiwano %v", tt.nazwa, err, tt.cel)
		}
	}
}

func TestUsuniecieDoKoszaIDziennik(t *testing.T) {
	s := noweSrodowisko(t)
	if err := s.kl.Remove("/publiczny/a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := s.kl.RemoveDir("/publiczny/f"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(s.m.kosz, ",") != "publiczny:/a.txt,publiczny:/f" {
		t.Fatalf("kosz: %v", s.m.kosz)
	}
	w := s.d.kopia()
	if len(w) != 2 || w[0].Rodzaj != dziennik.RodzajZmiana || w[0].Kto != "ola" || w[0].Protokol != "ftps" {
		t.Fatalf("dziennik: %+v", w)
	}
}

func TestZmianaPrawWTrakcieSesji(t *testing.T) {
	testy := []struct {
		nazwa string
		zmien func(u *rdzen.Uzytkownik)
	}{
		{"wyłączone", func(u *rdzen.Uzytkownik) { u.Wylaczony = true }},
		{"bez FTPS", func(u *rdzen.Uzytkownik) { u.FTPS = false }},
		{"odebrane prawo", func(u *rdzen.Uzytkownik) { u.Prawa = map[string]rdzen.Prawo{} }},
	}
	for _, tt := range testy {
		s := noweSrodowisko(t)
		u, _ := s.k.Uzytkownik("ola")
		tt.zmien(&u)
		s.k.ustaw(u)
		if _, err := s.kl.Stat("/publiczny/a.txt"); err == nil {
			t.Errorf("%s: Stat przeszedł", tt.nazwa)
		}
		if _, err := s.kl.GetHandle("/publiczny/a.txt", os.O_RDONLY, 0); err == nil {
			t.Errorf("%s: pobranie przeszło", tt.nazwa)
		}
	}
}

func TestPobranieZPrzesunieciem(t *testing.T) {
	s := noweSrodowisko(t)
	h, err := s.kl.GetHandle("/publiczny/a.txt", os.O_RDONLY, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Seek(2, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	dane, err := io.ReadAll(h)
	if err != nil || string(dane) != "cdef" {
		t.Fatalf("%q %v", dane, err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	_ = h.Close() // celowo: drugie zamknięcie nie może drugi raz kończyć transferu
	if w := s.tr.kopia(); len(w) != 1 || w[0] != transfery.WynikOk {
		t.Fatalf("wyniki: %v", w)
	}
	if st := s.tr.stany[0]; st.Kierunek != transfery.KierunekPobranie || st.Rozmiar != 6 || st.Protokol != "ftps" {
		t.Fatalf("stan: %+v", st)
	}
}

func TestPobraniePrzerwane(t *testing.T) {
	s := noweSrodowisko(t)
	h, err := s.kl.GetHandle("/publiczny/a.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	h.(ftpserver.FileTransferError).TransferError(io.ErrUnexpectedEOF)
	_ = h.Close() // celowo: wynik sprawdzamy w transferach
	if w := s.tr.kopia(); len(w) != 1 || w[0] != transfery.WynikPrzerwane {
		t.Fatalf("wyniki: %v", w)
	}
}

func TestPobranieBramka(t *testing.T) {
	s := noweSrodowisko(t)
	for _, p := range []string{"/", "/publiczny", "/tajny/x.txt", "/publiczny/brak.txt"} {
		if _, err := s.kl.GetHandle(p, os.O_RDONLY, 0); err == nil {
			t.Errorf("%s: pobranie przeszło", p)
		}
	}
}

func TestWgranie(t *testing.T) {
	s := noweSrodowisko(t)
	h, err := s.kl.GetHandle("/publiczny/nowy.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Write([]byte("treść")); err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if string(s.m.pliki["publiczny:/nowy.txt"]) != "treść" {
		t.Fatal("plik nie zapisany")
	}
	if w := s.tr.kopia(); len(w) != 1 || w[0] != transfery.WynikOk || s.tr.stany[0].Kierunek != transfery.KierunekWgranie {
		t.Fatalf("wyniki: %v", w)
	}
}

func TestWgraniePrzerwaneNieZostawiaPliku(t *testing.T) {
	s := noweSrodowisko(t)
	h, err := s.kl.GetHandle("/publiczny/nowy.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Write([]byte("pół")); err != nil {
		t.Fatal(err)
	}
	h.(ftpserver.FileTransferError).TransferError(io.ErrUnexpectedEOF)
	_ = h.Close() // celowo: wynik sprawdzamy niżej
	if _, jest := s.m.pliki["publiczny:/nowy.txt"]; jest {
		t.Fatal("przerwane wgrywanie zostawiło plik")
	}
	if w := s.tr.kopia(); len(w) != 1 || w[0] != transfery.WynikPrzerwane {
		t.Fatalf("wyniki: %v", w)
	}
}

func TestWgranieBladMagazynu(t *testing.T) {
	s := noweSrodowisko(t)
	s.m.bladZap = rdzen.ErrJuzIstnieje
	h, err := s.kl.GetHandle("/publiczny/nowy.txt", os.O_WRONLY|os.O_CREATE, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Write([]byte("x")); err != nil {
		h.(ftpserver.FileTransferError).TransferError(err)
	}
	if err := h.Close(); !errors.Is(err, rdzen.ErrJuzIstnieje) {
		t.Fatalf("Close: %v", err)
	}
	if w := s.tr.kopia(); len(w) != 1 || w[0] != transfery.WynikBlad {
		t.Fatalf("wyniki: %v", w)
	}
}

func TestWgranieBramka(t *testing.T) {
	testy := []struct {
		sciezka string
		flagi   int
		przes   int64
		cel     error
	}{
		{"/plik.txt", os.O_WRONLY | os.O_CREATE, 0, os.ErrPermission},
		{"/archiwum/x.txt", os.O_WRONLY | os.O_CREATE, 0, os.ErrPermission},
		{"/tajny/x.txt", os.O_WRONLY | os.O_CREATE, 0, os.ErrNotExist},
		{"/publiczny/nul.txt", os.O_WRONLY | os.O_CREATE, 0, ftpserver.ErrFileNameNotAllowed},
		{"/publiczny/x.txt", os.O_WRONLY | os.O_APPEND, 0, errWznowienieWgrania},
		{"/publiczny/x.txt", os.O_WRONLY | os.O_CREATE, 10, errWznowienieWgrania},
	}
	for _, tt := range testy {
		s := noweSrodowisko(t)
		if _, err := s.kl.GetHandle(tt.sciezka, tt.flagi, tt.przes); !errors.Is(err, tt.cel) {
			t.Errorf("%s: %v, oczekiwano %v", tt.sciezka, err, tt.cel)
		}
	}
}

func TestMkdirAll(t *testing.T) {
	s := noweSrodowisko(t)
	if err := s.kl.MkdirAll("/publiczny/f/g/h", 0); err != nil {
		t.Fatal(err)
	}
	if !s.m.foldery["publiczny:/f/g"] || !s.m.foldery["publiczny:/f/g/h"] {
		t.Fatalf("foldery: %v", s.m.foldery)
	}
	if err := s.kl.MkdirAll("/archiwum/a", 0); err == nil {
		t.Fatal("MkdirAll w udziale tylko do odczytu przeszedł")
	}
}
