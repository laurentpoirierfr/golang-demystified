package controle

import "testing"

func TestFizzBuzz(t *testing.T) {
	attendus := []string{"1", "2", "Fizz", "4", "Buzz", "Fizz", "7", "8", "Fizz", "Buzz", "11", "Fizz", "13", "14", "FizzBuzz"}
	for i, attendu := range attendus {
		if got := FizzBuzz(i + 1); got != attendu {
			t.Errorf("FizzBuzz(%d) = %q, attendu %q", i+1, got, attendu)
		}
	}
}

func TestMention(t *testing.T) {
	for note, attendu := range map[float64]string{
		20: "très bien", 16: "très bien", 15.5: "bien", 12: "assez bien",
		10: "passable", 9.99: "ajourné", 0: "ajourné", -1: "invalide", 21: "invalide",
	} {
		if got := Mention(note); got != attendu {
			t.Errorf("Mention(%v) = %q, attendu %q", note, got, attendu)
		}
	}
}

func TestSyracuse(t *testing.T) {
	for n, etapes := range map[int]int{1: 0, 2: 1, 3: 7, 6: 8, 27: 111, 0: -1, -5: -1} {
		if got := Syracuse(n); got != etapes {
			t.Errorf("Syracuse(%d) = %d, attendu %d", n, got, etapes)
		}
	}
}
