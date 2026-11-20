//go:build cgo

#include "ccalcul.h"
#include "_cgo_export.h" // déclare appelleGo, exportée par rappel.go

int64_t somme(const int64_t *t, size_t n) {
	int64_t s = 0;
	for (size_t i = 0; i < n; i++) {
		s += t[i];
	}
	return s;
}

void parcourt(const int64_t *t, size_t n, uintptr_t poignee) {
	for (size_t i = 0; i < n; i++) {
		appelleGo(poignee, t[i]);
	}
}
