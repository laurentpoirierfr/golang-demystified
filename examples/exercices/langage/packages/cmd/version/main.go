// Command version affiche sa version, fixée à la compilation (exercice 3) :
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/version
package main

import "fmt"

// version vaut « dev » par défaut ; -ldflags -X la remplace sans toucher au
// code. Ce doit être une variable de type string, pas une constante.
var version = "dev"

func main() {
	fmt.Println("version", version)
}
