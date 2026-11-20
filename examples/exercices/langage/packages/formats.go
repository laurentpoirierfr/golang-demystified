package packages

import "strings"

// Les formats livrés avec le package s'enregistrent à son initialisation.
func init() {
	Enregistre("majuscules", strings.ToUpper)
	Enregistre("minuscules", strings.ToLower)
}
