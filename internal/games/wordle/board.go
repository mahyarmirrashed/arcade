package wordle

const (
	wordLen  = 5
	maxGuess = 6
)

// hint describes the result of a single letter in a guess.
type hint int

const (
	hintAbsent  hint = iota // letter not in word
	hintPresent             // letter in word but wrong position
	hintCorrect             // letter in correct position
)

// row holds a single completed guess and its evaluation.
type row struct {
	letters [wordLen]rune
	hints   [wordLen]hint
}

// evaluate returns the hints for guessing g against answer a.
// Implements the standard Wordle algorithm: correct positions first, then
// present-but-wrong-position, consuming each answer letter at most once.
func evaluate(g, a string) [wordLen]hint {
	var hints [wordLen]hint
	used := [wordLen]bool{}

	// Pass 1: mark correct positions.
	for i := 0; i < wordLen; i++ {
		if rune(g[i]) == rune(a[i]) {
			hints[i] = hintCorrect
			used[i] = true
		}
	}

	// Pass 2: mark present-but-wrong-position.
	for i := 0; i < wordLen; i++ {
		if hints[i] == hintCorrect {
			continue
		}
		for j := 0; j < wordLen; j++ {
			if !used[j] && rune(g[i]) == rune(a[j]) {
				hints[i] = hintPresent
				used[j] = true
				break
			}
		}
	}

	return hints
}

// isWin reports whether all hints are correct.
func isWin(hints [wordLen]hint) bool {
	for _, h := range hints {
		if h != hintCorrect {
			return false
		}
	}
	return true
}
