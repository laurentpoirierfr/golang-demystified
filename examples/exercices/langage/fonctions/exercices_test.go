package fonctions

import "testing"

func TestMoyenne(t *testing.T) {
	if m, ok := Moyenne(10, 12, 17); !ok || m != 13 {
		t.Errorf("Moyenne = %v, %v", m, ok)
	}
	notes := []float64{8, 12}
	if m, _ := Moyenne(notes...); m != 10 {
		t.Errorf("Moyenne(notes...) = %v", m)
	}
	if _, ok := Moyenne(); ok {
		t.Error("sans argument, pas de moyenne")
	}
}

func TestCompteur(t *testing.T) {
	a, b := Compteur(1), Compteur(10)
	if a() != 1 || a() != 2 || b() != 10 || a() != 3 || b() != 11 {
		t.Error("deux compteurs doivent être indépendants")
	}
}

func TestMemoise(t *testing.T) {
	appels := 0
	carre := Memoise(func(n int) int { appels++; return n * n })
	for range 3 {
		if carre(4) != 16 || carre(5) != 25 {
			t.Fatal("mauvais résultat")
		}
	}
	if appels != 2 {
		t.Errorf("f appelée %d fois, attendu 2", appels)
	}
}
