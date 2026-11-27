package main

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"regexp"
	"strconv"
	"strings"
)

// Program działa na Windows i Linuksie: kod jednego systemu tylko w plikach tego systemu
// (przyrostek _windows.go / _linux.go albo //go:build).

var (
	wzorPlikuWindows = regexp.MustCompile(`_windows(_[a-z0-9]+)?(_test)?\.go$`)
	wzorPlikuUnix    = regexp.MustCompile(`_(linux|unix|darwin|freebsd|openbsd|netbsd)(_[a-z0-9]+)?(_test)?\.go$`)
)

// PakietyWindows / PakietyUnix: importy tylko dla jednego systemu (prefiks obejmuje podpakiety).
var (
	PakietyWindows = []string{"golang.org/x/sys/windows"}
	PakietyUnix    = []string{"golang.org/x/sys/unix"}
)

// NazwySyscallWindows: elementy pakietu syscall istniejące tylko na Windows (reszta wychodzi w GOOS=linux go vet).
var NazwySyscallWindows = map[string]bool{
	"Handle": true, "InvalidHandle": true, "LoadDLL": true, "MustLoadDLL": true, "NewLazyDLL": true, "LazyDLL": true,
	"LazyProc": true, "DLL": true, "Proc": true, "UTF16PtrFromString": true, "UTF16FromString": true,
	"UTF16ToString": true, "StringToUTF16Ptr": true, "StringToUTF16": true, "GetFileAttributesEx": true,
	"GetFileAttributes": true, "Win32FileAttributeData": true, "Win32finddata": true, "CreateFile": true,
	"CloseHandle": true, "GetLastError": true, "Filetime": true, "GetCurrentProcess": true, "GetFileExInfoStandard": true,
	"Win32finddata1": true, "FindFirstFile": true, "GetVolumeInformation": true,
}

// PrefiksySyscallWindows: grupy stałych tylko z Windows (np. syscall.FILE_ATTRIBUTE_HIDDEN).
var PrefiksySyscallWindows = []string{"FILE_ATTRIBUTE_", "ERROR_", "GENERIC_", "FILE_SHARE_", "Win32", "UTF16"}

// budujeSieNa: czy plik kompiluje się dla systemu (nazwa pliku i //go:build; "unix" = każdy poza windows).
func budujeSieNa(f *plikGo, system string) bool {
	if wzorPlikuWindows.MatchString(f.Sciezka) {
		return system == "windows"
	}
	if m := wzorPlikuUnix.FindStringSubmatch(f.Sciezka); m != nil {
		return system == m[1] || (m[1] == "unix" && system != "windows")
	}
	for _, grupa := range f.Drzewo.Comments {
		if grupa.End() >= f.Drzewo.Package {
			break
		}
		for _, k := range grupa.List {
			if !constraint.IsGoBuild(k.Text) {
				continue
			}
			wyr, err := constraint.Parse(k.Text)
			if err != nil {
				return true
			}
			return wyr.Eval(func(tag string) bool { return tag == system || (tag == "unix" && system != "windows") })
		}
	}
	return true
}

// sprawdzSystem: importy i wywołania jednego systemu tylko w plikach tego systemu.
func sprawdzSystem(p *projekt) []Naruszenie {
	var wynik []Naruszenie
	for _, pk := range p.Pakiety {
		for _, f := range pk.Pliki {
			wynik = append(wynik, p.sprawdzPlikSystemu(f)...)
		}
	}
	return wynik
}

func (p *projekt) sprawdzPlikSystemu(f *plikGo) []Naruszenie {
	var wynik []Naruszenie
	naLinuksie, naWindows := budujeSieNa(f, "linux"), budujeSieNa(f, "windows")
	for _, imp := range f.Drzewo.Imports {
		sciezka, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if naLinuksie && maPrefiks(sciezka, PakietyWindows) {
			wynik = append(wynik, naruszenieSystemu(f, p.linia(imp.Pos()), sciezka, "windows"))
		}
		if naWindows && maPrefiks(sciezka, PakietyUnix) {
			wynik = append(wynik, naruszenieSystemu(f, p.linia(imp.Pos()), sciezka, "linux"))
		}
	}
	sys := nazwaImportu(f.Drzewo, "syscall")
	if !naLinuksie || sys == "" {
		return wynik
	}
	ast.Inspect(f.Drzewo, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == sys && czyNazwaWindows(sel.Sel.Name) {
			wynik = append(wynik, naruszenieSystemu(f, p.linia(sel.Pos()), "syscall."+sel.Sel.Name, "windows"))
		}
		return true
	})
	return wynik
}

func naruszenieSystemu(f *plikGo, linia int, co, system string) Naruszenie {
	return Naruszenie{Regula: "architektura.system", Plik: f.Sciezka, Linia: linia, Nazwa: co,
		Opis: fmt.Sprintf("%s jest tylko dla %s: przenieś do pliku *_%s.go albo dodaj //go:build %s (program działa na Windows i Linuksie)",
			co, system, system, system)}
}

func maPrefiks(sciezka string, prefiksy []string) bool {
	for _, p := range prefiksy {
		if sciezka == p || strings.HasPrefix(sciezka, p+"/") {
			return true
		}
	}
	return false
}

func czyNazwaWindows(nazwa string) bool {
	if NazwySyscallWindows[nazwa] {
		return true
	}
	for _, p := range PrefiksySyscallWindows {
		if strings.HasPrefix(nazwa, p) {
			return true
		}
	}
	return false
}
