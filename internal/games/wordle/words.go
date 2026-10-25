package wordle

import (
	_ "embed"
	"math/rand"
	"strings"
)

//go:embed words.txt
var wordsTxt string

// words holds all valid 5-letter words, loaded once at startup.
// The same list is used for both answer selection and guess validation.
var words []string

// wordSet is the O(1) lookup set for guess validation.
var wordSet map[string]struct{}

func init() {
	lines := strings.Split(strings.TrimSpace(wordsTxt), "\n")
	words = make([]string, 0, len(lines))
	wordSet = make(map[string]struct{}, len(lines))
	for _, w := range lines {
		w = strings.TrimSpace(w)
		if len(w) == wordLen {
			words = append(words, w)
			wordSet[w] = struct{}{}
		}
	}
}

// randomAnswer picks a random answer from the word list.
func randomAnswer(rng *rand.Rand) string {
	return words[rng.Intn(len(words))]
}

// isValid reports whether w is an accepted guess.
func isValid(w string) bool {
	_, ok := wordSet[w]
	return ok
}
