package http

import (
	"net"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/siec"
)

// opcjaProtokolu: pozycja menu „Pobierz przez…”.
type opcjaProtokolu struct {
	ID       string `json:"id"`
	Nazwa    string `json:"nazwa"`
	Opis     string `json:"opis"`
	Adres    string `json:"adres,omitempty"`
	Port     int    `json:"port,omitempty"`
	Dostepne bool   `json:"dostepne"`
	Powod    string `json:"powod,omitempty"`
}

// hostZadania: host z nagłówka Host (tak, jak klient wszedł: LAN, WireGuard), tylko bezpieczne znaki.
func hostZadania(r *nethttp.Request) string {
	h := r.Host
	if h == "" || len(h) > 255 {
		return "localhost"
	}
	for _, c := range h {
		dozwolony := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.ContainsRune(".-:[]", c)
		if !dozwolony {
			return "localhost"
		}
	}
	return h
}

// bazaAdresu: schemat://host[:port] żądania.
func bazaAdresu(r *nethttp.Request) string {
	if r.TLS != nil {
		return "https://" + hostZadania(r)
	}
	return "http://" + hostZadania(r)
}

// samHost: host bez portu (IPv6 w nawiasach).
func samHost(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	if strings.Contains(h, ":") {
		return "[" + strings.Trim(h, "[]") + "]"
	}
	return h
}

// adresAPI: pełny adres /api/<co>?udzial&sciezka.
func adresAPI(r *nethttp.Request, co, udzial, sciezka string) string {
	q := url.Values{"udzial": {udzial}, "sciezka": {sciezka}}
	return bazaAdresu(r) + konfiguracja.PrefiksAPI + co + "?" + q.Encode()
}

// protokoly: GET /api/protokoly?udzial&sciezka.
func (s *serwer) protokoly(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
	udzial, sciezka, err := s.celZadania(r, u, rdzen.OpPobranie)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	wpis, err := s.magazyn.Info(udzial, sciezka)
	if err != nil {
		s.blad(w, r, err)
		return
	}
	adresy := s.siec.adresyLokalne()
	if adresy == nil {
		adresy = []siec.Adres{}
	}
	piszJSON(w, nethttp.StatusOK, map[string]any{
		"opcje":              s.opcjeProtokolow(r, u, udzial, sciezka, wpis.Folder),
		"adresy":             adresy,
		"odcisk_certyfikatu": s.odcisk,
	})
}

func (s *serwer) opcjeProtokolow(r *nethttp.Request, u rdzen.Uzytkownik, udzial, sciezka string,
	folder bool) []opcjaProtokolu {
	nazwaWWW := "Przeglądarka (HTTP)"
	if r.TLS != nil {
		nazwaWWW = "Przeglądarka (HTTPS)"
	}
	pobierz := adresAPI(r, "pobierz", udzial, sciezka)
	www := opcjaProtokolu{ID: "https", Nazwa: nazwaWWW, Opis: "Pobranie z wznawianiem", Adres: pobierz, Dostepne: true}
	link := opcjaProtokolu{ID: "link", Nazwa: "Kopiuj link bezpośredni", Opis: "Wymaga zalogowania",
		Adres: pobierz, Dostepne: true}
	if folder {
		www.Dostepne, www.Adres, www.Powod = false, "", "To jest folder. Pobierz go jako ZIP."
		link.Dostepne, link.Adres, link.Powod = false, "", www.Powod
	}
	zip := opcjaProtokolu{ID: "zip", Nazwa: "Jako ZIP", Opis: "Dla folderów i wielu plików",
		Adres: adresAPI(r, "zip", udzial, sciezka), Dostepne: true}
	pub := opcjaProtokolu{ID: "publiczny", Nazwa: "Link publiczny…", Opis: "Bez logowania, z datą ważności",
		Dostepne: true}
	if err := rdzen.CzyWolno(u, rdzen.OpUdostepnienie, udzial, false); err != nil {
		pub.Dostepne, pub.Powod = false, "Brak prawa do udostępniania."
	}
	return []opcjaProtokolu{www, s.opcjaFTPS(r, u, udzial, sciezka), link, zip, pub}
}

// opcjaFTPS: adres ftps://login@host:port/udzial/sciezka albo powód niedostępności.
func (s *serwer) opcjaFTPS(r *nethttp.Request, u rdzen.Uzytkownik, udzial, sciezka string) opcjaProtokolu {
	k := s.konfiguracja()
	o := opcjaProtokolu{ID: "ftps", Nazwa: "Klient FTP (FTPS)", Opis: "FileZilla, Total Commander, WinSCP",
		Port: k.PortFTPS}
	switch {
	case !k.FTPSWlaczone:
		o.Powod = "FTPS jest wyłączony w ustawieniach."
		return o
	case !u.FTPS:
		o.Powod = "To konto nie ma dostępu przez FTPS."
		return o
	}
	adres := url.URL{
		Scheme: "ftps", User: url.User(u.Login),
		Host: samHost(hostZadania(r)) + ":" + strconv.Itoa(k.PortFTPS),
		Path: "/" + udzial + strings.TrimSuffix(sciezka, "/"),
	}
	o.Adres, o.Dostepne = adres.String(), true
	return o
}
