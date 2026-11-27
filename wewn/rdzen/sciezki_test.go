package rdzen

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"skrytka/wewn/konfiguracja"
)

func TestOczyscSciezke(t *testing.T) {
	przypadki := []struct {
		wej  string
		chce string
		blad error
	}{
		// poprawne
		{"", "/", nil},
		{".", "/", nil},
		{"/", "/", nil},
		{"///", "", ErrPozaUdzialem}, // zaczyna się od "//" = UNC
		{"a", "/a", nil},
		{"/a/b", "/a/b", nil},
		{"a/b/", "/a/b", nil},
		{"/a//b", "/a/b", nil},
		{"/./a/./b/.", "/a/b", nil},
		{"/zdjęcia/Żółw.JPG", "/zdjęcia/Żółw.JPG", nil},
		{"/.ukryty", "/.ukryty", nil},
		{"/a..b", "/a..b", nil},
		{"/...a", "/...a", nil},
		{"/%2e%2e", "/%2e%2e", nil}, // dekodowanie URL to sprawa HTTP; tu to zwykła nazwa
		{"/" + konfiguracja.KatalogKosza + "/x", "/" + konfiguracja.KatalogKosza + "/x", nil},
		{"/console.txt", "/console.txt", nil},
		{"/com10", "/com10", nil},
		// wyjście w górę
		{"..", "", ErrPozaUdzialem},
		{"/..", "", ErrPozaUdzialem},
		{"../a", "", ErrPozaUdzialem},
		{"/a/../b", "", ErrPozaUdzialem},
		{"/a/../../b", "", ErrPozaUdzialem},
		{"/a/..", "", ErrPozaUdzialem},
		{"a/b/../../..", "", ErrPozaUdzialem},
		// ukośnik wsteczny
		{`\`, "", ErrPozaUdzialem},
		{`..\a`, "", ErrPozaUdzialem},
		{`/a\..\..\b`, "", ErrPozaUdzialem},
		{`a\b`, "", ErrPozaUdzialem},
		{`\\serwer\udzial`, "", ErrPozaUdzialem},
		// UNC i ścieżki urządzeń
		{"//serwer/udzial", "", ErrPozaUdzialem},
		{"//?/C:/Windows", "", ErrPozaUdzialem},
		{"//./PhysicalDrive0", "", ErrPozaUdzialem},
		// dyski
		{"C:", "", ErrPozaUdzialem},
		{"c:/Windows", "", ErrPozaUdzialem},
		{"/C:/Windows", "", ErrPozaUdzialem},
		{"C:Windows", "", ErrPozaUdzialem},
		{"/a/d:", "", ErrPozaUdzialem},
		{"z:", "", ErrPozaUdzialem},
		// strumienie NTFS i inne dwukropki
		{"/plik.txt:ukryty", "", ErrZlaNazwa},
		{"/plik.txt::$DATA", "", ErrZlaNazwa},
		{"/folder:$I30:$INDEX_ALLOCATION", "", ErrZlaNazwa},
		{"/1:", "", ErrZlaNazwa},
		{":", "", ErrZlaNazwa},
		// kropka/spacja na końcu (Windows je obcina: ".. " = "..")
		{"/.. ", "", ErrZlaNazwa},
		{"/. .", "", ErrZlaNazwa},
		{"/...", "", ErrZlaNazwa},
		{"/a/.. /b", "", ErrZlaNazwa},
		{"/a/..../b", "", ErrZlaNazwa},
		{"/plik.", "", ErrZlaNazwa},
		{"/plik ", "", ErrZlaNazwa},
		{"/ ", "", ErrZlaNazwa},
		// nazwy urządzeń
		{"/CON", "", ErrZlaNazwa},
		{"/a/nul", "", ErrZlaNazwa},
		{"/NUL.txt", "", ErrZlaNazwa},
		{"/aux.tar.gz", "", ErrZlaNazwa},
		{"/Com1", "", ErrZlaNazwa},
		{"/lpt9.log", "", ErrZlaNazwa},
		{"/COM¹", "", ErrZlaNazwa},
		{"/prn .txt", "", ErrZlaNazwa},
		{"/CONIN$", "", ErrZlaNazwa},
		// znaki sterujące i zakazane
		{"/a\x00b", "", ErrZlaNazwa},
		{"/a\x00/../..", "", ErrZlaNazwa},
		{"/a\nb", "", ErrZlaNazwa},
		{"/a\tb", "", ErrZlaNazwa},
		{"/a\x7fb", "", ErrZlaNazwa},
		{"/a\u0085b", "", ErrZlaNazwa},
		{"/zdjecie\u202Egpj.exe", "", ErrZlaNazwa},
		{"/a*b", "", ErrZlaNazwa},
		{"/a?b", "", ErrZlaNazwa},
		{`/a"b`, "", ErrZlaNazwa},
		{"/a<b", "", ErrZlaNazwa},
		{"/a>b", "", ErrZlaNazwa},
		{"/a|b", "", ErrZlaNazwa},
		{"/a\xffb", "", ErrZlaNazwa},
		// długości
		{"/" + strings.Repeat("a", konfiguracja.MaksDlugoscNazwy+1), "", ErrZlaNazwa},
		{strings.Repeat("/a", konfiguracja.MaksGlebokoscSciezki+1), "", ErrZlaNazwa},
		{strings.Repeat("/", konfiguracja.MaksDlugoscSciezki+1), "", ErrZlaNazwa},
	}
	for _, p := range przypadki {
		got, err := OczyscSciezke(p.wej)
		if !errors.Is(err, p.blad) || (p.blad == nil && err != nil) {
			t.Errorf("OczyscSciezke(%q) błąd = %v, chcę %v", p.wej, err, p.blad)
			continue
		}
		if got != p.chce {
			t.Errorf("OczyscSciezke(%q) = %q, chcę %q", p.wej, got, p.chce)
		}
	}
}

func TestOczyscSciezkeIdempotentna(t *testing.T) {
	for _, s := range []string{"", "a/b", "/x/./y//z/", "/zdjęcia/Żółw"} {
		raz, err := OczyscSciezke(s)
		if err != nil {
			t.Fatalf("OczyscSciezke(%q): %v", s, err)
		}
		dwa, err := OczyscSciezke(raz)
		if err != nil || dwa != raz {
			t.Errorf("drugie OczyscSciezke(%q) = %q, %v; chcę %q", raz, dwa, err, raz)
		}
	}
}

func TestSciezkaNaDysku(t *testing.T) {
	korzen := filepath.Join(t.TempDir(), "udzial")
	przypadki := []struct {
		korzen, wej string
		chce        string
		blad        error
	}{
		{korzen, "", korzen, nil},
		{korzen, "/", korzen, nil},
		{korzen, "/a/b.txt", filepath.Join(korzen, "a", "b.txt"), nil},
		{korzen + string(filepath.Separator), "/a", filepath.Join(korzen, "a"), nil},
		{korzen, "/a/./b//c", filepath.Join(korzen, "a", "b", "c"), nil},
		{korzen, "/..", "", ErrPozaUdzialem},
		{korzen, "/a/../../x", "", ErrPozaUdzialem},
		{korzen, `..\..\Windows`, "", ErrPozaUdzialem},
		{korzen, "C:/Windows/win.ini", "", ErrPozaUdzialem},
		{korzen, "//serwer/udzial/x", "", ErrPozaUdzialem},
		{korzen, "/plik:strumien", "", ErrZlaNazwa},
		{korzen, "/.. ", "", ErrZlaNazwa},
		{korzen, "/NUL", "", ErrZlaNazwa},
		{"", "/a", "", ErrPozaUdzialem},
		{"udzial", "/a", "", ErrPozaUdzialem},
		{`\udzial`, "/a", "", ErrPozaUdzialem},
	}
	for _, p := range przypadki {
		got, err := SciezkaNaDysku(p.korzen, p.wej)
		if !errors.Is(err, p.blad) || (p.blad == nil && err != nil) {
			t.Errorf("SciezkaNaDysku(%q, %q) błąd = %v, chcę %v", p.korzen, p.wej, err, p.blad)
			continue
		}
		if got != p.chce {
			t.Errorf("SciezkaNaDysku(%q, %q) = %q, chcę %q", p.korzen, p.wej, got, p.chce)
		}
		if err == nil && !wewnatrzKorzenia(filepath.Clean(p.korzen), got) {
			t.Errorf("SciezkaNaDysku(%q, %q) = %q poza korzeniem", p.korzen, p.wej, got)
		}
	}
}

// TestPolaczZKorzeniem: druga linia obrony musi odrzucić wyjście poza korzeń nawet bez OczyscSciezke.
func TestPolaczZKorzeniem(t *testing.T) {
	korzen := filepath.Join(t.TempDir(), "a")
	przypadki := []struct {
		wej  string
		chce string
		blad error
	}{
		{"/", korzen, nil},
		{"/b/c", filepath.Join(korzen, "b", "c"), nil},
		{"/b/../c", filepath.Join(korzen, "c"), nil},
		{"/..", "", ErrPozaUdzialem},
		{"/../ab", "", ErrPozaUdzialem}, // sąsiedni folder o wspólnym prefiksie nazwy
		{"/../a.bak/x", "", ErrPozaUdzialem},
		{"/b/../../a2", "", ErrPozaUdzialem},
		{"/../../../../../../../../Windows", "", ErrPozaUdzialem},
	}
	for _, p := range przypadki {
		got, err := polaczZKorzeniem(korzen, p.wej)
		if got != p.chce || !errors.Is(err, p.blad) || (p.blad == nil && err != nil) {
			t.Errorf("polaczZKorzeniem(%q, %q) = %q, %v; chcę %q, %v", korzen, p.wej, got, err, p.chce, p.blad)
		}
	}
}

func TestWewnatrzKorzenia(t *testing.T) {
	s := string(filepath.Separator)
	przypadki := []struct {
		kor, wynik string
		chce       bool
	}{
		{`C:\a`, `C:\a`, true},
		{`C:\a`, `C:\a\b`, true},
		{`C:\a`, `C:\a\b\c.txt`, true},
		{`C:\a`, `C:\ab`, false},
		{`C:\a`, `C:\ab\c`, false},
		{`C:\a`, `C:\a.bak\c`, false},
		{`C:\a`, `C:\`, false},
		{`C:\a`, `C:\b`, false},
		{`C:\a`, `D:\a\b`, false},
		{`C:\`, `C:\x`, true},
		{`\\srv\udzial`, `\\srv\udzial2\x`, false},
		{`\\srv\udzial`, `\\srv\udzial\x`, true},
	}
	for _, p := range przypadki {
		kor := strings.ReplaceAll(p.kor, `\`, s)
		wynik := strings.ReplaceAll(p.wynik, `\`, s)
		if got := wewnatrzKorzenia(kor, wynik); got != p.chce {
			t.Errorf("wewnatrzKorzenia(%q, %q) = %v, chcę %v", kor, wynik, got, p.chce)
		}
	}
}
