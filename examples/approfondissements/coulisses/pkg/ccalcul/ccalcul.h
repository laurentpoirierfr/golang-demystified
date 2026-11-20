// Fonctions C appelées depuis Go.
#include <stddef.h>
#include <stdint.h>

int64_t somme(const int64_t *t, size_t n);

// parcourt appelle, pour chaque élément, la fonction Go exportée
// appelleGo, en lui repassant la poignée reçue.
void parcourt(const int64_t *t, size_t n, uintptr_t poignee);
