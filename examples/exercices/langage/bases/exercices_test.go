package bases

import "testing"

func TestFahrenheit(t *testing.T) {
	for c, f := range map[float64]float64{0: 32, 100: 212, -40: -40, 37: 98.6} {
		if got := Fahrenheit(c); got < f-1e-9 || got > f+1e-9 {
			t.Errorf("Fahrenheit(%v) = %v, attendu %v", c, got, f)
		}
	}
}

func TestFormateTaille(t *testing.T) {
	if Kio != 1024 || Gio != 1<<30 {
		t.Fatalf("Kio = %d, Gio = %d", Kio, Gio)
	}
	for n, attendu := range map[int64]string{
		0: "0 o", 512: "512 o", 1024: "1.0 Kio", 1536: "1.5 Kio",
		5 * Mio: "5.0 Mio", 3*Gio + Gio/2: "3.5 Gio", 2 * Tio: "2.0 Tio",
	} {
		if got := FormateTaille(n); got != attendu {
			t.Errorf("FormateTaille(%d) = %q, attendu %q", n, got, attendu)
		}
	}
}

func TestAdditionneInt8(t *testing.T) {
	tests := []struct {
		a, b    int8
		somme   int8
		depasse bool
	}{
		{1, 2, 3, false},
		{100, 27, 127, false},
		{100, 28, -128, true}, // le résultat « fait le tour »
		{-100, -29, 127, true},
		{-128, 0, -128, false},
	}
	for _, tt := range tests {
		somme, depasse := AdditionneInt8(tt.a, tt.b)
		if somme != tt.somme || depasse != tt.depasse {
			t.Errorf("AdditionneInt8(%d, %d) = %d, %v ; attendu %d, %v", tt.a, tt.b, somme, depasse, tt.somme, tt.depasse)
		}
	}
}
