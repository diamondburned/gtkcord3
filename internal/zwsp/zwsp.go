// Package zwsp provides functions for inserting and removing zero-width spaces
// from a string. This package also works with Markdown.
package zwsp

import (
	"strings"
	"unicode"
)

// The frequency to insert a zero-width space into a paragraph.
const Frequency = 4

// Offset sets the offset for the number used to determine insertion.
const Offset = -2

func Insert(into string) string {
	// Break the string into runes:
	// Pre-grow the slice:
	runes := make([]rune, 0, len(into)+(len(into)/Frequency+1))
	runes = append(runes, []rune(into)...)

	var paused, inCode bool

	// Iterate rune-by-rune. <j> is the count used to determine frequency. It's
	// only incremented after the rune checks passed.
	for i, j := 0, Offset; i < len(runes); i++ {
		// Check if this is a markdown symbol:
		switch runes[i] {
		case '`':
			inCode = !inCode

			// Is this a codeblock?
			if readAhead(runes, i, 3) == "```" {
				// If so, we can skip the next 3 characters and go straight to
				// the content.
				i += 3
				continue
			}

		case '<':
			paused = true
		case '>':
			paused = false

		case '*', '_':
			// Is the next rune also these characters?
			switch opener := readAhead(runes, i, 3); opener {
			case "***", "**", "__":
				// Skip these characters:
				i += len(opener)
				continue
			}

		case ' ':
			// Set paused to false, as we know spaces never happen inside <> or
			// a URL.
			if paused {
				paused = false
			}

			// Skip spaces.
			continue

		case 'h': // [h]ttp

		}

		// Don't obfuscate if paused is true. This could be that we're in a
		// codeblock.
		if paused || inCode {
			continue
		}

		// Only count if surrounding runes are letters (...a[bcd]e...):
		if checkRunes(readAheadRunes(runes, i-1, 3), unicode.IsLetter) {
			// Determine if it's time to insert using j:
			if j%Frequency == 0 {
				// Insert a null rune at the end:
				runes = append(runes, 0)
				// Shift the entire rune slice from this point to the right once:
				copy(runes[i+1:], runes[i:])
				// Set the current rune to the zero-width space:
				runes[i] = '\u200b'

				// Skip the inserted zero-width space:
				i++
			}

			// j is always behind.
			j++
		}
	}

	return string(runes)
}

// Delete removes all zero-width spaces from a string.
func Delete(from string) string {
	return strings.Replace(from, "\u200b", "", -1)
}

// peak the next rune in the slice, except -1 is returned if the next index is
// out of range.
func peakNext(runes []rune, i int) rune {
	if i < len(runes)-1 {
		return runes[i+1]
	}
	return -1
}
func peakPrev(runes []rune, i int) rune {
	if i > 0 {
		return runes[i-1]
	}
	return -1
}

// try and read runes starting from i and going up delta. string returned has a
// maximum length of delta.
func readAhead(runes []rune, i, delta int) string {
	return string(readAheadRunes(runes, i, delta))
}

func readAheadRunes(runes []rune, i, delta int) []rune {
	if i < 0 {
		i = 0
	}

	max := i + delta
	if max >= len(runes) {
		max = len(runes) - 1
	}

	return runes[i:max]
}

func checkRunes(runes []rune, checker func(r rune) bool) bool {
	for _, r := range runes {
		if !checker(r) {
			return false
		}
	}
	return true
}
