package md

import (
	"fmt"
	"testing"
)

func strcmp(t *testing.T, name, got, expected string) {
	if got != expected {
		t.Fatal("Mismatch", name, "<expected/got>:\n", expected, "\n", got)
	}
}

func TestRenderMarkup(t *testing.T) {
	const _md = `**hello *world*!**` + "\n```" + `go
package main

func main() {
	fmt.Println("Bruh moment.")
}
` + "```"

	html := ParseToMarkup([]byte(_md))
	fmt.Println(string(html))
}
