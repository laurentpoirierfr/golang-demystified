// Le premier programme du livre.
package main

import "fmt"

func main() {
	fmt.Println(greeting("Gopher"))
}

func greeting(name string) string {
	return "Bonjour, " + name + " !"
}
