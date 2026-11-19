// Exemples du chapitre « Le temps : time ».
package temps

import (
	"fmt"
	"time"
)

func Example_dates() {
	t := time.Date(2026, time.August, 12, 14, 30, 0, 0, time.UTC)
	fmt.Println(t)
	fmt.Println(t.Year(), t.Month(), t.Day(), t.Weekday(), t.YearDay())
	// Output:
	// 2026-08-12 14:30:00 +0000 UTC
	// 2026 August 12 Wednesday 224
}

func Example_format() {
	t := time.Date(2026, time.August, 12, 14, 30, 5, 0, time.UTC)
	fmt.Println(t.Format(time.RFC3339))
	fmt.Println(t.Format(time.DateTime))
	fmt.Println(t.Format(time.DateOnly))
	fmt.Println(t.Format("02/01/2006 à 15h04"))
	fmt.Println(t.Format("Mon 2 Jan 06, 3:04 PM"))
	// Output:
	// 2026-08-12T14:30:05Z
	// 2026-08-12 14:30:05
	// 2026-08-12
	// 12/08/2026 à 14h30
	// Wed 12 Aug 26, 2:30 PM
}

func Example_parse() {
	t, err := time.Parse(time.DateOnly, "2012-03-28")
	fmt.Println(t, err)

	_, err = time.Parse(time.DateOnly, "28/03/2012")
	fmt.Println(err)
	// Output:
	// 2012-03-28 00:00:00 +0000 UTC <nil>
	// parsing time "28/03/2012" as "2006-01-02": cannot parse "28/03/2012" as "2006"
}

func Example_durees() {
	d := 90 * time.Minute
	fmt.Println(d, d.Hours(), d.Seconds())

	attente, _ := time.ParseDuration("1h15m30.5s")
	fmt.Println(attente, attente.Round(time.Minute))

	// Une durée à partir d'une variable entière demande une conversion.
	n := 3
	fmt.Println(time.Duration(n) * time.Second)
	// Output:
	// 1h30m0s 1.5 5400
	// 1h15m30.5s 1h16m0s
	// 3s
}

func Example_calculs() {
	debut := time.Date(2026, time.January, 31, 9, 0, 0, 0, time.UTC)
	fmt.Println(debut.Add(36 * time.Hour).Format(time.DateTime))
	fmt.Println(debut.AddDate(0, 1, 0).Format(time.DateOnly)) // normalisé !

	fin := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	fmt.Println(fin.Sub(debut), fin.After(debut), debut.Equal(debut.UTC()))
	fmt.Println(debut.Truncate(24 * time.Hour).Format(time.DateTime))
	// Output:
	// 2026-02-01 21:00:00
	// 2026-03-03
	// 696h0m0s true true
	// 2026-01-31 00:00:00
}

func Example_fuseaux() {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		fmt.Println(err)
		return
	}
	tokyo, _ := time.LoadLocation("Asia/Tokyo")

	reunion := time.Date(2026, time.July, 1, 10, 0, 0, 0, paris)
	fmt.Println(reunion)
	fmt.Println(reunion.In(tokyo))
	fmt.Println(reunion.UTC())
	// Output:
	// 2026-07-01 10:00:00 +0200 CEST
	// 2026-07-01 17:00:00 +0900 JST
	// 2026-07-01 08:00:00 +0000 UTC
}

func Example_mesure() {
	debut := time.Now()
	time.Sleep(10 * time.Millisecond)
	ecoule := time.Since(debut)
	fmt.Println(ecoule >= 10*time.Millisecond)
	// Output: true
}
