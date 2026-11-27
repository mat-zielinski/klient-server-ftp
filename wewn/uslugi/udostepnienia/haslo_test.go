package udostepnienia

import (
	"strings"
	"testing"
)

func TestHashujISprawdzHaslo(t *testing.T) {
	zapis, err := hashujHaslo("Żółw 123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(zapis, "$argon2id$v=19$m=") {
		t.Fatalf("zły format PHC: %s", zapis)
	}
	drugi, err := hashujHaslo("Żółw 123")
	if err != nil || drugi == zapis {
		t.Fatal("ta sama sól dla dwóch hashy")
	}
	przypadki := []struct {
		nazwa, haslo string
		dobre        bool
	}{
		{"to samo", "Żółw 123", true},
		{"inna wielkość liter", "żółw 123", false},
		{"puste", "", false},
		{"dłuższe", "Żółw 1234", false},
	}
	for _, p := range przypadki {
		if got := sprawdzHaslo(p.haslo, zapis); got != p.dobre {
			t.Errorf("%s: %v, oczekiwano %v", p.nazwa, got, p.dobre)
		}
	}
}

func TestCzytajHashOdrzucaZle(t *testing.T) {
	przypadki := []string{
		"",
		"jawne",
		"$argon2i$v=19$m=19456,t=2,p=1$c29s$a2x1Y3o",
		"$argon2id$v=16$m=19456,t=2,p=1$c29s$a2x1Y3o",
		"$argon2id$v=19$m=0,t=2,p=1$c29s$a2x1Y3o",
		"$argon2id$v=19$m=99999999,t=2,p=1$c29s$a2x1Y3o",
		"$argon2id$v=19$m=19456,t=0,p=1$c29s$a2x1Y3o",
		"$argon2id$v=19$m=19456,t=2,p=0$c29s$a2x1Y3o",
		"$argon2id$v=19$m=19456,t=2,p=1$$a2x1Y3o",
		"$argon2id$v=19$m=19456,t=2,p=1$c29s$",
		"$argon2id$v=19$m=19456,t=2,p=1$!!$a2x1Y3o",
	}
	for _, z := range przypadki {
		if _, err := czytajHash(z); err == nil {
			t.Errorf("przyjęto zły zapis %q", z)
		}
		if sprawdzHaslo("", z) {
			t.Errorf("hasło pasuje do złego zapisu %q", z)
		}
	}
}
