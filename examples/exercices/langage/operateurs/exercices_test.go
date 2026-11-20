package operateurs

import (
	"math/bits"
	"testing"
)

func TestEstPair(t *testing.T) {
	for n, pair := range map[int]bool{0: true, 1: false, 2: true, -3: false, -4: true} {
		if EstPair(n) != pair {
			t.Errorf("EstPair(%d) = %v", n, !pair)
		}
	}
}

func TestDroits(t *testing.T) {
	d := Lecture | Ecriture
	if !A(d, Lecture) || !A(d, Lecture|Ecriture) || A(d, Execution) || A(d, Lecture|Execution) {
		t.Error("A se trompe")
	}
	if d = Retire(d, Ecriture); d != Lecture {
		t.Errorf("Retire : %03b", d)
	}
	if Retire(d, Execution) != Lecture {
		t.Error("retirer un droit absent ne change rien")
	}
}

func TestBits(t *testing.T) {
	for _, n := range []uint64{0, 1, 0b1011, 255, 1 << 63, ^uint64(0)} {
		if got, attendu := Bits(n), bits.OnesCount64(n); got != attendu {
			t.Errorf("Bits(%d) = %d, attendu %d", n, got, attendu)
		}
	}
}
