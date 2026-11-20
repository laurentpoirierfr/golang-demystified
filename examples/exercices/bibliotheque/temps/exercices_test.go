package temps

import (
	"testing"
	"time"
)

func date(a int, m time.Month, j int) time.Time { return time.Date(a, m, j, 12, 0, 0, 0, time.UTC) }

func TestAge(t *testing.T) {
	naissance := date(1990, time.June, 15)
	for jour, attendu := range map[time.Time]int{
		date(2026, time.June, 14): 35, date(2026, time.June, 15): 36, date(2026, time.December, 1): 36,
	} {
		if got := Age(naissance, jour); got != attendu {
			t.Errorf("le %s : %d, attendu %d", jour.Format(time.DateOnly), got, attendu)
		}
	}
	// Né un 29 février : un an de plus le 1er mars.
	if got := Age(date(2000, time.February, 29), date(2025, time.March, 1)); got != 25 {
		t.Errorf("29 février : %d", got)
	}
}

func TestJoursOuvres(t *testing.T) {
	// Du lundi 16 au lundi 23 novembre 2026 : une semaine, cinq jours.
	if n := JoursOuvres(date(2026, time.November, 16), date(2026, time.November, 23)); n != 5 {
		t.Errorf("une semaine : %d", n)
	}
	if n := JoursOuvres(date(2026, time.November, 21), date(2026, time.November, 23)); n != 0 {
		t.Errorf("un week-end : %d", n)
	}
	// À travers le passage à l'heure d'hiver (25 octobre 2026), à Paris.
	paris, _ := time.LoadLocation("Europe/Paris")
	debut := time.Date(2026, time.October, 19, 0, 0, 0, 0, paris)
	if n := JoursOuvres(debut, debut.AddDate(0, 0, 14)); n != 10 {
		t.Errorf("deux semaines avec changement d'heure : %d", n)
	}
}

func TestLitDateParis(t *testing.T) {
	for texte, utc := range map[string]string{
		"20/11/2026 14:30": "2026-11-20T13:30:00Z", // heure d'hiver : UTC+1
		"14/07/2026 22:00": "2026-07-14T20:00:00Z", // heure d'été : UTC+2
	} {
		got, err := LitDateParis(texte)
		if err != nil || got.Format(time.RFC3339) != utc {
			t.Errorf("%s : %v, %v ; attendu %s", texte, got, err, utc)
		}
	}
	if _, err := LitDateParis("2026-11-20"); err == nil {
		t.Error("un autre format doit être refusé")
	}
}
