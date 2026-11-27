package main

import "testing"

func TestSystem(t *testing.T) {
	przypadki := []struct {
		nazwa      string
		plik       string
		tresc      string
		oczekiwane []string
	}{
		{"x/sys/windows w zwykłym pliku", "wewn/siec/s.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)",
			[]string{"architektura.system@wewn/siec/s.go"}},
		{"x/sys/windows/registry w zwykłym pliku", "wewn/siec/s.go", "import \"golang.org/x/sys/windows/registry\"\nvar _ = registry.CURRENT_USER",
			[]string{"architektura.system@wewn/siec/s.go"}},
		{"x/sys/windows w pliku _windows.go", "wewn/siec/s_windows.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)", nil},
		{"x/sys/windows w teście _windows_test.go", "wewn/siec/s_windows_test.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)", nil},
		{"x/sys/windows z //go:build windows", "wewn/siec/s.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)", nil},
		{"x/sys/windows z //go:build !linux", "wewn/siec/s.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)", nil},
		{"x/sys/windows z //go:build !windows", "wewn/siec/s.go", "import \"golang.org/x/sys/windows\"\nvar _ = windows.Handle(0)",
			[]string{"architektura.system@wewn/siec/s.go"}},
		{"syscall Windows w zwykłym pliku", "wewn/magazyn/m.go", "import \"syscall\"\nvar _, _ = syscall.UTF16PtrFromString(\"a\")",
			[]string{"architektura.system@wewn/magazyn/m.go"}},
		{"stała syscall Windows", "wewn/magazyn/m.go", "import \"syscall\"\nconst A = syscall.FILE_ATTRIBUTE_HIDDEN",
			[]string{"architektura.system@wewn/magazyn/m.go"}},
		{"syscall Windows w _windows.go", "wewn/magazyn/m_windows.go", "import \"syscall\"\nvar _, _ = syscall.UTF16PtrFromString(\"a\")", nil},
		{"syscall wspólny wolno", "wewn/magazyn/m.go", "import \"syscall\"\nvar _ = syscall.ENOENT", nil},
		{"x/sys/unix w zwykłym pliku", "wewn/siec/s.go", "import \"golang.org/x/sys/unix\"\nvar _ = unix.Getpid",
			[]string{"architektura.system@wewn/siec/s.go"}},
		{"x/sys/unix w _linux.go", "wewn/siec/s_linux.go", "import \"golang.org/x/sys/unix\"\nvar _ = unix.Getpid", nil},
		{"x/sys/unix z //go:build unix", "wewn/siec/s.go", "import \"golang.org/x/sys/unix\"\nvar _ = unix.Getpid", nil},
	}
	ograniczenia := map[string]string{
		"x/sys/windows z //go:build windows":  "//go:build windows\n\n",
		"x/sys/windows z //go:build !linux":   "//go:build !linux\n\n",
		"x/sys/windows z //go:build !windows": "//go:build !windows\n\n",
		"x/sys/unix z //go:build unix":        "//go:build unix\n\n",
	}
	for _, p := range przypadki {
		t.Run(p.nazwa, func(t *testing.T) {
			tresc := ograniczenia[p.nazwa] + "package x\n" + p.tresc + "\n"
			porownaj(t, sprawdzSystem(wczytaj(t, map[string]string{p.plik: tresc})), p.oczekiwane...)
		})
	}
}
