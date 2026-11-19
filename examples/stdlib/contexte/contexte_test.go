// Exemples du chapitre « Annulation et délais : context ».
package contexte

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// calcul simule un traitement long qui respecte l'annulation.
func calcul(ctx context.Context, duree time.Duration) (int, error) {
	select {
	case <-time.After(duree):
		return 42, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func Example_timeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := calcul(ctx, time.Second)
	fmt.Println(err, errors.Is(err, context.DeadlineExceeded))

	v, err := calcul(context.Background(), time.Millisecond)
	fmt.Println(v, err)
	// Output:
	// context deadline exceeded true
	// 42 <nil>
}

func Example_cancel() {
	ctx, cancel := context.WithCancel(context.Background())
	resultat := make(chan error, 1)
	go func() {
		_, err := calcul(ctx, time.Hour)
		resultat <- err
	}()
	cancel() // l'utilisateur a fermé la page, par exemple
	fmt.Println(<-resultat)
	// Output: context canceled
}

func Example_cause() {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("quota dépassé"))
	fmt.Println(ctx.Err())
	fmt.Println(context.Cause(ctx))
	// Output:
	// context canceled
	// quota dépassé
}

type cleContexte struct{}

func avecRequeteID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, cleContexte{}, id)
}

func requeteID(ctx context.Context) string {
	id, _ := ctx.Value(cleContexte{}).(string)
	return id
}

func Example_valeur() {
	ctx := avecRequeteID(context.Background(), "a1b2")
	fmt.Printf("%q %q\n", requeteID(ctx), requeteID(context.Background()))
	// Output: "a1b2" ""
}

func Example_afterFunc() {
	ctx, cancel := context.WithCancel(context.Background())
	termine := make(chan struct{})
	context.AfterFunc(ctx, func() {
		fmt.Println("nettoyage après annulation")
		close(termine)
	})
	cancel()
	<-termine
	// Output: nettoyage après annulation
}

func Example_withoutCancel() {
	parent, cancel := context.WithCancel(context.Background())
	detache := context.WithoutCancel(parent)
	cancel()
	fmt.Println(parent.Err(), detache.Err())
	// Output: context canceled <nil>
}
