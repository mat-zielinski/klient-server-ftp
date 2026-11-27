package main

import "testing"

func TestBrakTestow(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		pliki      map[string]string
		oczekiwane []string
	}{
		{"logika bez testów", map[string]string{"wewn/rdzen/r.go": "package rdzen\nfunc F() {}\n"},
			[]string{"testy.brak_testow@wewn/rdzen"}},
		{"logika z testami", map[string]string{
			"wewn/rdzen/r.go":      "package rdzen\nfunc F() {}\n",
			"wewn/rdzen/r_test.go": "package rdzen\n",
		}, nil},
		{"same typy i stałe", map[string]string{"wewn/rdzen/typy.go": "package rdzen\ntype T int\nconst C = 1\n"}, nil},
		{"sam kontrakt z zaślepkami", map[string]string{"wewn/ftp/kontrakt.go": "package ftp\nfunc Uruchom() error { return nil }\n"}, nil},
		{"kontrakt i logika", map[string]string{
			"wewn/ftp/kontrakt.go": "package ftp\n",
			"wewn/ftp/serwer.go":   "package ftp\nfunc F() {}\n",
		}, []string{"testy.brak_testow@wewn/ftp"}},
		{"poza wewn nie wymagane", map[string]string{
			"cmd/skrytka/main.go":  "package main\nfunc main() {}\n",
			"narzedzia_dev/x/x.go": "package x\nfunc F() {}\n",
		}, nil},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			porownaj(t, sprawdzTesty(wczytaj(t, p.pliki)), p.oczekiwane...)
		})
	}
}

func TestPlikTestow(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		tresc      string
		oczekiwane []string
	}{
		{"time.Sleep", "import (\n\"testing\"\n\"time\"\n)\nfunc TestA(t *testing.T) { time.Sleep(time.Millisecond) }",
			[]string{"testy.sleep@wewn/rdzen/r_test.go"}},
		{"time.Sleep z aliasem", "import (\n\"testing\"\ncz \"time\"\n)\nfunc TestA(t *testing.T) { cz.Sleep(1) }",
			[]string{"testy.sleep@wewn/rdzen/r_test.go"}},
		{"inne funkcje time wolno", "import (\n\"testing\"\n\"time\"\n)\nfunc TestA(t *testing.T) { _ = time.Unix(0, 0) }", nil},
		{"Skip z powodem", "import \"testing\"\nfunc TestA(t *testing.T) { t.Skip(\"tylko Windows\") }", nil},
		{"Skipf z powodem", "import \"testing\"\nfunc TestA(t *testing.T) { t.Skipf(\"brak %s\", \"x\") }", nil},
		{"Skip bez powodu", "import \"testing\"\nfunc TestA(t *testing.T) { t.Skip() }",
			[]string{"testy.skip_bez_powodu@wewn/rdzen/r_test.go"}},
		{"Skip z pustym napisem", "import \"testing\"\nfunc TestA(t *testing.T) { t.Skip(\"  \") }",
			[]string{"testy.skip_bez_powodu@wewn/rdzen/r_test.go"}},
		{"SkipNow", "import \"testing\"\nfunc TestA(t *testing.T) { t.SkipNow() }",
			[]string{"testy.skip_bez_powodu@wewn/rdzen/r_test.go"}},
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			pliki := map[string]string{
				"wewn/rdzen/r.go":      "package rdzen\n",
				"wewn/rdzen/r_test.go": "package rdzen\n" + p.tresc + "\n",
			}
			porownaj(t, sprawdzTesty(wczytaj(t, pliki)), p.oczekiwane...)
		})
	}
}

func TestSleepPozaTestamiNieJestRegulaTestow(t *testing.T) {
	pliki := map[string]string{"wewn/magazyn/m.go": "package magazyn\nimport \"time\"\nvar _ = func() int { time.Sleep(1); return 0 }\n"}
	porownaj(t, sprawdzTesty(wczytaj(t, pliki)))
}
