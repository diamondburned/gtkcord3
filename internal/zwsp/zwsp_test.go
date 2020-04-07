package zwsp

import (
	"strings"
	"testing"
)

const testString = "this is a test string ```go" + `
package main

func main() {
	// long comment aaaaaa
}
` + "``` **bold ***italicized and bold*** lol** *italics* `test code block asdasdasd` <@29039481294823842390402348234832> https://dklqyfdsfsdf.com/joemama/io23eu238e8238e23edloooool"

func TestInsert(t *testing.T) {
	s := Insert(testString)

	// The actual count is 7.
	count := strings.Count(s, "\u200b")

	if count < 5 {
		t.Fatal("Less than 5 zero-width spaces found.")
	}

	if count > 10 {
		t.Fatal("More than 10 zero-width spaces found.")
	}
}

var dumb string

func BenchmarkInsert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dumb = Insert(testString)
	}
}
