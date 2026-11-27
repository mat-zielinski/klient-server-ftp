package ftp

import (
	"bytes"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	ftpserver "github.com/fclairamb/ftpserverlib"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
	"skrytka/wewn/uslugi/transfery"
)

// kontaTest: konta w pamięci (hasło jawne, tylko w testach).
type kontaTest struct {
	mu     sync.Mutex
	konta  map[string]rdzen.Uzytkownik
	hasla  map[string]string
	blokuj bool
}

func (k *kontaTest) Zaloguj(login, haslo, _ string) (rdzen.Uzytkownik, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.blokuj {
		return rdzen.Uzytkownik{}, rdzen.ErrKontoZablokowane
	}
	u, ok := k.konta[login]
	if !ok || k.hasla[login] != haslo {
		return rdzen.Uzytkownik{}, rdzen.ErrZleDane
	}
	return u, nil
}

func (k *kontaTest) Uzytkownik(login string) (rdzen.Uzytkownik, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	u, ok := k.konta[login]
	if !ok {
		return rdzen.Uzytkownik{}, rdzen.ErrNieZnaleziono
	}
	return u, nil
}

func (k *kontaTest) ustaw(u rdzen.Uzytkownik) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.konta[u.Login] = u
}

// magazynTest: pliki w pamięci; Otworz kopiuje treść do pliku w t.TempDir.
type magazynTest struct {
	t       *testing.T
	mu      sync.Mutex
	udzialy []konfiguracja.Udzial
	pliki   map[string][]byte // "udzial:/sciezka"
	foldery map[string]bool
	kosz    []string
	bladZap error
}

func nowyMagazynTest(t *testing.T) *magazynTest {
	return &magazynTest{
		t: t,
		udzialy: []konfiguracja.Udzial{
			{Nazwa: "publiczny", Sciezka: "x"},
			{Nazwa: "tajny", Sciezka: "y"},
			{Nazwa: "archiwum", Sciezka: "z", TylkoOdczyt: true},
		},
		pliki:   map[string][]byte{"publiczny:/a.txt": []byte("abcdef"), "archiwum:/b.txt": []byte("b")},
		foldery: map[string]bool{"publiczny:/f": true, "publiczny:/pelny": true},
	}
}

func klucz(u, s string) string { return u + ":" + s }

func (m *magazynTest) Udzialy() []konfiguracja.Udzial { return m.udzialy }

func (m *magazynTest) Udzial(n string) (konfiguracja.Udzial, error) {
	for _, u := range m.udzialy {
		if u.Nazwa == n {
			return u, nil
		}
	}
	return konfiguracja.Udzial{}, rdzen.ErrNieZnaleziono
}

func (m *magazynTest) Lista(u, s string) ([]rdzen.Wpis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var w []rdzen.Wpis
	for k, d := range m.pliki {
		if strings.HasPrefix(k, u+":") && path.Dir(strings.TrimPrefix(k, u+":")) == s {
			w = append(w, rdzen.Wpis{Nazwa: path.Base(k), Rozmiar: int64(len(d))})
		}
	}
	if s == "/pelny" {
		w = append(w, rdzen.Wpis{Nazwa: "cos"})
	}
	return w, nil
}

func (m *magazynTest) Info(u, s string) (rdzen.Wpis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.pliki[klucz(u, s)]; ok {
		return rdzen.Wpis{Nazwa: path.Base(s), Sciezka: s, Rozmiar: int64(len(d))}, nil
	}
	if m.foldery[klucz(u, s)] {
		return rdzen.Wpis{Nazwa: path.Base(s), Sciezka: s, Folder: true}, nil
	}
	return rdzen.Wpis{}, rdzen.ErrNieZnaleziono
}

func (m *magazynTest) Otworz(u, s string) (*os.File, rdzen.Wpis, error) {
	w, err := m.Info(u, s)
	if err != nil || w.Folder {
		return nil, w, rdzen.ErrNieZnaleziono
	}
	p := filepath.Join(m.t.TempDir(), "plik")
	m.mu.Lock()
	dane := m.pliki[klucz(u, s)]
	m.mu.Unlock()
	if err := os.WriteFile(p, dane, 0o600); err != nil {
		return nil, w, err
	}
	f, err := os.Open(p)
	return f, w, err
}

func (m *magazynTest) Zapisz(u, s string, r io.Reader, _ bool) (int64, error) {
	if m.bladZap != nil {
		return 0, m.bladZap
	}
	var b bytes.Buffer
	n, err := io.Copy(&b, r)
	if err != nil {
		return 0, err
	}
	m.mu.Lock()
	m.pliki[klucz(u, s)] = b.Bytes()
	m.mu.Unlock()
	return n, nil
}

func (m *magazynTest) NowyFolder(u, s string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.foldery[klucz(u, s)] = true
	return nil
}

func (m *magazynTest) ZmienNazwe(u, z, na string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.pliki[klucz(u, z)]
	if !ok {
		return rdzen.ErrNieZnaleziono
	}
	delete(m.pliki, klucz(u, z))
	m.pliki[klucz(u, na)] = d
	return nil
}

func (m *magazynTest) DoKosza(u, s string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pliki, klucz(u, s))
	delete(m.foldery, klucz(u, s))
	m.kosz = append(m.kosz, klucz(u, s))
	return nil
}

// dziennikTest zbiera wpisy.
type dziennikTest struct {
	mu    sync.Mutex
	wpisy []dziennik.Wpis
}

func (d *dziennikTest) Zapisz(w dziennik.Wpis) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wpisy = append(d.wpisy, w)
}

func (d *dziennikTest) kopia() []dziennik.Wpis {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]dziennik.Wpis(nil), d.wpisy...)
}

// transferTest zapamiętuje wynik.
type transferTest struct {
	mu     sync.Mutex
	stany  []transfery.Stan
	wyniki []string
}

type uchwytTest struct {
	r *transferTest
	s transfery.Stan
}

func (u uchwytTest) CzytajZ(src io.Reader) io.Reader { return src }
func (u uchwytTest) PiszDo(dst io.Writer) io.Writer  { return dst }
func (u uchwytTest) Zakoncz(w string) {
	u.r.mu.Lock()
	defer u.r.mu.Unlock()
	u.r.stany = append(u.r.stany, u.s)
	u.r.wyniki = append(u.r.wyniki, w)
}

func (r *transferTest) rozpocznij(s transfery.Stan) uchwytTransferu { return uchwytTest{r: r, s: s} }

func (r *transferTest) kopia() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.wyniki...)
}

// srodowisko: usługi na zaślepkach; użytkownik „ola”: publiczny=zapis, archiwum=zapis (tylko do odczytu), tajny=brak.
type srodowisko struct {
	u  uslugi
	k  *kontaTest
	m  *magazynTest
	d  *dziennikTest
	tr *transferTest
	kl *klient
}

const hasloTestowe = "dobre-haslo-123"

func noweSrodowisko(t *testing.T) *srodowisko {
	k := &kontaTest{
		konta: map[string]rdzen.Uzytkownik{
			"ola": {Login: "ola", FTPS: true, Prawa: map[string]rdzen.Prawo{
				"publiczny": rdzen.PrawoZapis, "archiwum": rdzen.PrawoZapis, "tajny": rdzen.PrawoBrak,
			}},
			"web": {Login: "web", FTPS: false, Prawa: map[string]rdzen.Prawo{"publiczny": rdzen.PrawoOdczyt}},
		},
		hasla: map[string]string{"ola": hasloTestowe, "web": hasloTestowe},
	}
	s := &srodowisko{k: k, m: nowyMagazynTest(t), d: &dziennikTest{}, tr: &transferTest{}}
	s.u = uslugi{
		magazyn: s.m, konta: k, dziennik: s.d, rozpocznij: s.tr.rozpocznij, czyWolno: rdzen.CzyWolno,
	}
	s.kl = &klient{u: s.u, login: "ola", ip: "10.0.0.1"}
	return s
}

// kontekstTest: minimalny ftpserver.ClientContext.
type kontekstTest struct {
	ftpserver.ClientContext
	id        uint32
	adres     net.Addr
	sciezka   string
	zamkniety bool
}

func (c *kontekstTest) ID() uint32           { return c.id }
func (c *kontekstTest) RemoteAddr() net.Addr { return c.adres }
func (c *kontekstTest) SetPath(p string)     { c.sciezka = p }
func (c *kontekstTest) Close() error         { c.zamkniety = true; return nil }

func kontekst(id uint32, ip string) *kontekstTest {
	return &kontekstTest{id: id, adres: &net.TCPAddr{IP: net.ParseIP(ip), Port: 40000 + int(id)}}
}
