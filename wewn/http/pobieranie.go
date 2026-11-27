package http

import (
	"errors"
	"fmt"
	"io"
	"mime"
	nethttp "net/http"
	"path"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/transfery"
)

// zadaniePobrania: kto i co pobiera (konto albo link publiczny).
type zadaniePobrania struct {
	kto     string
	udzial  string
	sciezka string
	podglad bool
}

// pobierz: GET /api/pobierz?udzial&sciezka[&podglad=1].
func (s *serwer) pobierz(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpPobranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	s.wyslijPlik(w, r, zadaniePobrania{
		kto: u.Login, udzial: udzial, sciezka: sciezka, podglad: r.URL.Query().Get("podglad") == "1",
	})
}

// wyslijPlik: http.ServeContent (Range, ETag, If-Modified-Since), bajty liczone przez transfery
// (wpis dziennika robi callback rejestru). Bramka sprawdzona wcześniej przez wołającego.
func (s *serwer) wyslijPlik(w nethttp.ResponseWriter, r *nethttp.Request, z zadaniePobrania) {
	f, wpis, err := s.magazyn.Otworz(z.udzial, z.sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	defer func() { _ = f.Close() }() // celowo: plik tylko do odczytu, błąd zamknięcia bez znaczenia
	h := w.Header()
	typ, aktywny := typZawartosci(wpis.Nazwa, z.podglad)
	h.Set("Content-Type", typ)
	if aktywny {
		h.Set("Content-Security-Policy", konfiguracja.PolitykaCSPPlikuAktywnego)
	}
	h.Set("Content-Disposition", naglowekNazwy(!z.podglad, wpis.Nazwa))
	h.Set("ETag", etag(wpis))
	h.Set("Cache-Control", "private, no-cache")
	p := s.nowyPisarz(w, r, z)
	nethttp.ServeContent(p, r, wpis.Nazwa, wpis.Zmieniono, f)
	p.zakoncz()
}

// etag z rozmiaru i czasu zmiany (silny w obrębie jednego pliku).
func etag(w rdzen.Wpis) string {
	return fmt.Sprintf(`"%x-%x"`, w.Rozmiar, w.Zmieniono.UnixNano())
}

// typyAktywne: treść, którą przeglądarka mogłaby wykonać w naszym pochodzeniu.
var typyAktywne = []string{"text/html", "application/xhtml+xml", "image/svg+xml", "text/xml", "application/xml"}

// typZawartosci: bez podglądu zawsze octet-stream; skrypty nigdy z typem JS (nosniff blokuje
// wykonanie); HTML przy podglądzie jako tekst. Drugi wynik: czy dodać piaskownicę CSP.
func typZawartosci(nazwa string, podglad bool) (string, bool) {
	if !podglad {
		return "application/octet-stream", false
	}
	typ := mime.TypeByExtension(strings.ToLower(path.Ext(nazwa)))
	if typ == "" {
		return "application/octet-stream", false
	}
	baza, _, _ := strings.Cut(typ, ";")
	baza = strings.TrimSpace(strings.ToLower(baza))
	switch {
	case strings.Contains(baza, "javascript") || strings.Contains(baza, "ecmascript"):
		return "text/plain; charset=utf-8", false
	case baza == "text/html" || baza == "application/xhtml+xml":
		return "text/plain; charset=utf-8", true
	}
	for _, a := range typyAktywne {
		if baza == a {
			return typ, true
		}
	}
	return typ, false
}

// naglowekNazwy: Content-Disposition z nazwą ASCII zastępczą i pełną UTF-8 wg RFC 5987.
func naglowekNazwy(zalacznik bool, nazwa string) string {
	rodzaj := "inline"
	if zalacznik {
		rodzaj = "attachment"
	}
	return rodzaj + `; filename="` + nazwaASCII(nazwa) + `"; filename*=UTF-8''` + kodujRFC5987(nazwa)
}

// nazwaASCII: zastępcza nazwa dla starych klientów (bez cudzysłowów, ukośników i znaków spoza ASCII).
func nazwaASCII(nazwa string) string {
	var b strings.Builder
	for _, c := range nazwa {
		if c < 0x20 || c >= 0x7f || c == '"' || c == '\\' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// kodujRFC5987: attr-char bez zmian, reszta jako %XX bajtów UTF-8.
func kodujRFC5987(s string) string {
	const dozwolone = "!#$&+-.^_`|~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.IndexByte(dozwolone, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

// pisarzTransferu liczy wysłane bajty przez rejestr transferów. Transfer rusza dopiero przy
// odpowiedzi 200/206 na GET (304, 412, 416 i HEAD nie są pobraniem).
type pisarzTransferu struct {
	nethttp.ResponseWriter
	s       *serwer
	r       *nethttp.Request
	z       zadaniePobrania
	t       transfer
	cel     io.Writer
	kod     int
	oczek   int64
	bajty   int64
	bladZap error
	// bladZrodla: błąd odczytu z dysku po wysłaniu nagłówków (wynik „blad”).
	bladZrodla error
	// rozmiar: znany rozmiar do paska postępu, gdy brak Content-Length (ZIP).
	rozmiar int64
}

func (s *serwer) nowyPisarz(w nethttp.ResponseWriter, r *nethttp.Request, z zadaniePobrania) *pisarzTransferu {
	return &pisarzTransferu{ResponseWriter: w, s: s, r: r, z: z, oczek: -1}
}

// Unwrap dla http.ResponseController.
func (p *pisarzTransferu) Unwrap() nethttp.ResponseWriter { return p.ResponseWriter }

func (p *pisarzTransferu) WriteHeader(kod int) {
	if p.kod == 0 {
		p.kod = kod
		ok := kod == nethttp.StatusOK || kod == nethttp.StatusPartialContent
		if ok && p.r.Method == nethttp.MethodGet {
			p.rozpocznij()
		}
	}
	p.ResponseWriter.WriteHeader(kod)
}

func (p *pisarzTransferu) rozpocznij() {
	if n, err := strconv.ParseInt(p.Header().Get("Content-Length"), 10, 64); err == nil {
		p.oczek = n
	}
	rozmiar := p.oczek
	if rozmiar < 0 {
		rozmiar = p.rozmiar
	}
	if rozmiar < 0 {
		rozmiar = 0
	}
	p.t = p.s.transfery.Rozpocznij(transfery.Stan{
		Kto: p.z.kto, IP: adresIP(p.r), Protokol: protokol(p.r), Kierunek: transfery.KierunekPobranie,
		Udzial: p.z.udzial, Sciezka: p.z.sciezka, Rozmiar: rozmiar,
	})
	p.cel = p.t.PiszDo(p.ResponseWriter)
}

func (p *pisarzTransferu) Write(b []byte) (int, error) {
	if p.kod == 0 {
		p.WriteHeader(nethttp.StatusOK)
	}
	if p.cel == nil {
		return p.ResponseWriter.Write(b)
	}
	n, err := p.cel.Write(b)
	p.bajty += int64(n)
	if err != nil && p.bladZap == nil {
		p.bladZap = err
	}
	return n, err
}

// wynik: ok, gdy wysłano wszystko; przerwane przy rozłączeniu, przerwaniu przez admina albo braku bajtów.
func (p *pisarzTransferu) wynik() string {
	if p.bladZrodla != nil && p.bladZap == nil {
		return transfery.WynikBlad
	}
	if errors.Is(p.bladZap, transfery.ErrPrzerwany) {
		return transfery.WynikPrzerwane
	}
	if p.bladZap != nil || p.r.Context().Err() != nil {
		return transfery.WynikPrzerwane
	}
	if p.oczek >= 0 && p.bajty < p.oczek {
		return transfery.WynikPrzerwane
	}
	return transfery.WynikOk
}

// zakoncz zamyka transfer z wynikiem; wpis dziennika robi callback naKoniec rejestru (jedno źródło dla HTTP i FTPS).
func (p *pisarzTransferu) zakoncz() {
	if p.t == nil {
		return
	}
	wynik := p.wynik()
	p.t.Zakoncz(wynik)
}

// zipGet: GET /api/zip?udzial&sciezka=/a[&sciezka=/b.txt…] (zwykły link, strumień w przeglądarce).
func (s *serwer) zipGet(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	sciezki := r.URL.Query()["sciezka"]
	if len(sciezki) == 0 {
		sciezki = []string{"/"}
	}
	s.zipSciezek(w, r, u, r.URL.Query().Get("udzial"), sciezki)
}

// zipPost: POST /api/zip?udzial z {"sciezki":[...]}.
func (s *serwer) zipPost(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	var d struct {
		Sciezki []string `json:"sciezki"`
	}
	if err := czytajJSON(w, r, &d); err != nil {
		s.blad(w, r, err)
		return
	}
	s.zipSciezek(w, r, u, r.URL.Query().Get("udzial"), d.Sciezki)
}

// zipSciezek: bramka (OpPobranie) przed magazynem, każda ścieżka oczyszczona.
func (s *serwer) zipSciezek(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik, udzial string,
	sciezki []string) {
	if _, err := s.dostepDoUdzialu(u, rdzen.OpPobranie, udzial); err != nil {
		s.blad(w, r, err)
		return
	}
	if len(sciezki) == 0 {
		s.blad(w, r, nowyBlad(nethttp.StatusBadRequest, "Wybierz pliki do spakowania."))
		return
	}
	czyste := make([]string, 0, len(sciezki))
	for _, sc := range sciezki {
		c, err := rdzen.OczyscSciezke(sc)
		if err != nil {
			s.blad(w, r, err)
			return
		}
		czyste = append(czyste, c)
	}
	s.wyslijZip(w, r, u.Login, udzial, czyste)
}

// nazwaZip: jeden element = jego nazwa, korzeń albo wiele = nazwa udziału.
func nazwaZip(udzial string, sciezki []string) string {
	if len(sciezki) == 1 && sciezki[0] != "/" {
		return path.Base(sciezki[0]) + konfiguracja.PrzyrostekZip
	}
	return udzial + konfiguracja.PrzyrostekZip
}

// wyslijZip strumieniuje ZIP przez rejestr transferów.
func (s *serwer) wyslijZip(w nethttp.ResponseWriter, r *nethttp.Request, kto, udzial string, sciezki []string) {
	var rozmiar int64
	for _, sc := range sciezki {
		n, err := s.magazyn.RozmiarFolderu(udzial, sc)
		if err != nil {
			s.blad(w, r, err)
			return
		}
		rozmiar += n
	}
	nazwa := nazwaZip(udzial, sciezki)
	h := w.Header()
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Disposition", naglowekNazwy(true, nazwa))
	h.Set("Cache-Control", "no-store")
	z := zadaniePobrania{kto: kto, udzial: udzial, sciezka: strings.Join(sciezki, ", ")}
	p := s.nowyPisarz(w, r, z)
	p.rozmiar = rozmiar
	p.WriteHeader(nethttp.StatusOK)
	if _, err := s.magazyn.ZipDo(p, udzial, sciezki); err != nil && p.bladZap == nil {
		p.bladZrodla = err
	}
	p.zakoncz()
}
