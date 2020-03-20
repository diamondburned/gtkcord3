package md

import "testing"

func strcmp(t *testing.T, name, got, expected string) {
	if got != expected {
		t.Fatal("Mismatch", name, "<expected/got>:\n", expected, "\n", got)
	}
}
