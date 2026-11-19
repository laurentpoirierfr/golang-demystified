package main

import "testing"

func TestGreeting(t *testing.T) {
	if got, want := greeting("Gopher"), "Bonjour, Gopher !"; got != want {
		t.Errorf("greeting() = %q, attendu %q", got, want)
	}
}
