package wordle

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---- board.go tests ----

func TestEvaluate_allCorrect(t *testing.T) {
	hints := evaluate("crane", "crane")
	for i, h := range hints {
		if h != hintCorrect {
			t.Errorf("hints[%d] = %v; want hintCorrect", i, h)
		}
	}
}

func TestEvaluate_allAbsent(t *testing.T) {
	hints := evaluate("zzzzz", "abcde")
	for i, h := range hints {
		if h != hintAbsent {
			t.Errorf("hints[%d] = %v; want hintAbsent", i, h)
		}
	}
}

func TestEvaluate_present(t *testing.T) {
	// 'a' is in "alpha" but not at index 0 of "crane"
	hints := evaluate("crane", "alpha")
	// c→absent, r→absent, a→present (a is at index 0,2,4 in alpha → present for position 2), n→absent, e→absent
	// Actually let's use a deterministic case: guess "abcde", answer "eabcd"
	// a(0)→present, b(1)→present, c(2)→present, d(3)→present, e(4)→present
	hints2 := evaluate("abcde", "eabcd")
	for i, h := range hints2 {
		if h != hintPresent {
			t.Errorf("hints2[%d] = %v; want hintPresent", i, h)
		}
	}
	_ = hints
}

func TestEvaluate_duplicateLetterInGuess(t *testing.T) {
	// guess "speed", answer "spell"
	// s(0)→correct, p(1)→correct, e(2)→correct, e(3)→absent (only one e left), d(4)→absent
	hints := evaluate("speed", "spell")
	want := [wordLen]hint{hintCorrect, hintCorrect, hintCorrect, hintAbsent, hintAbsent}
	if hints != want {
		t.Errorf("got %v; want %v", hints, want)
	}
}

func TestEvaluate_duplicateLetterInAnswer(t *testing.T) {
	// guess "llama", answer "hello": "hello" has l at positions 2 and 3.
	// Pass 1 (correct): no exact matches.
	// Pass 2 (present): l(0) consumes hello[2], l(1) consumes hello[3]; both present.
	hints := evaluate("llama", "hello")
	if hints[0] != hintPresent {
		t.Errorf("hints[0] = %v; want hintPresent", hints[0])
	}
	if hints[1] != hintPresent {
		t.Errorf("hints[1] = %v; want hintPresent", hints[1])
	}
	if hints[2] != hintAbsent {
		t.Errorf("hints[2] = %v; want hintAbsent", hints[2])
	}
}

func TestIsWin_true(t *testing.T) {
	hints := [wordLen]hint{hintCorrect, hintCorrect, hintCorrect, hintCorrect, hintCorrect}
	if !isWin(hints) {
		t.Error("expected win")
	}
}

func TestIsWin_false(t *testing.T) {
	hints := [wordLen]hint{hintCorrect, hintCorrect, hintCorrect, hintCorrect, hintAbsent}
	if isWin(hints) {
		t.Error("expected no win")
	}
}

// ---- words.go tests ----

func TestWordList_notEmpty(t *testing.T) {
	if len(words) == 0 {
		t.Fatal("word list is empty")
	}
}

func TestWordList_allFiveLetters(t *testing.T) {
	for _, w := range words {
		if len(w) != wordLen {
			t.Errorf("word %q has length %d; want %d", w, len(w), wordLen)
		}
	}
}

func TestIsValid_knownWord(t *testing.T) {
	if !isValid(words[0]) {
		t.Errorf("word %q should be valid", words[0])
	}
}

func TestIsValid_unknownWord(t *testing.T) {
	if isValid("zzzzz") {
		t.Error("zzzzz should not be valid")
	}
}

// ---- model tests ----

func newTestModel() *Model {
	m := newModel(80, 24)
	m.answer = "crane" // deterministic answer for tests
	return m
}

func TestModel_initialState(t *testing.T) {
	m := newTestModel()
	if m.state != statePlaying {
		t.Error("new model should be in playing state")
	}
	if m.current != 0 {
		t.Errorf("current = %d; want 0", m.current)
	}
	if m.input != "" {
		t.Errorf("input = %q; want empty", m.input)
	}
}

func TestModel_typingAddsToInput(t *testing.T) {
	m := newTestModel()
	m.handleKey(keyMsg("c"))
	m.handleKey(keyMsg("r"))
	if m.input != "cr" {
		t.Errorf("input = %q; want %q", m.input, "cr")
	}
}

func TestModel_backspaceRemovesLetter(t *testing.T) {
	m := newTestModel()
	m.handleKey(keyMsg("c"))
	m.handleKey(keyMsg("r"))
	m.handleKey(keyMsgSpecial("backspace"))
	if m.input != "c" {
		t.Errorf("input = %q; want %q", m.input, "c")
	}
}

func TestModel_tooShortGuessGivesNotice(t *testing.T) {
	m := newTestModel()
	m.handleKey(keyMsg("c"))
	m.handleKey(keyMsgSpecial("enter"))
	if m.notice == "" {
		t.Error("expected notice for too-short guess")
	}
	if m.current != 0 {
		t.Error("current should not advance on invalid guess")
	}
}

func TestModel_invalidWordGivesNotice(t *testing.T) {
	m := newTestModel()
	for _, ch := range "zzzzz" {
		m.handleKey(keyMsg(string(ch)))
	}
	m.handleKey(keyMsgSpecial("enter"))
	if m.notice == "" {
		t.Error("expected notice for invalid word")
	}
}

func TestModel_correctGuessWins(t *testing.T) {
	m := newTestModel()
	for _, ch := range "crane" {
		m.handleKey(keyMsg(string(ch)))
	}
	m.handleKey(keyMsgSpecial("enter"))
	if m.state != stateWon {
		t.Errorf("state = %v; want stateWon", m.state)
	}
}

func TestModel_sixWrongGuessesLoses(t *testing.T) {
	m := newTestModel()
	// "zzzzz" is not valid; use a real word that isn't "crane"
	// "storm" is in the word list and != "crane"
	for i := 0; i < maxGuess; i++ {
		m.input = ""
		for _, ch := range "storm" {
			m.handleKey(keyMsg(string(ch)))
		}
		m.handleKey(keyMsgSpecial("enter"))
	}
	if m.state != stateLost {
		t.Errorf("state = %v; want stateLost after %d wrong guesses", m.state, maxGuess)
	}
}

func TestModel_inputCappedAtWordLen(t *testing.T) {
	m := newTestModel()
	for _, ch := range "craneextra" {
		m.handleKey(keyMsg(string(ch)))
	}
	if len(m.input) != wordLen {
		t.Errorf("input length = %d; want %d", len(m.input), wordLen)
	}
}

func TestModel_restartResets(t *testing.T) {
	m := newTestModel()
	for _, ch := range "crane" {
		m.handleKey(keyMsg(string(ch)))
	}
	m.handleKey(keyMsgSpecial("enter"))
	if m.state != stateWon {
		t.Fatal("setup: expected win")
	}

	cmd := m.handleKey(keyMsg("r"))
	if cmd == nil {
		t.Fatal("expected restart command")
	}
	msg := cmd()
	rm, ok := msg.(replaceMsg)
	if !ok {
		t.Fatalf("expected replaceMsg, got %T", msg)
	}
	if rm.m.state != statePlaying {
		t.Error("fresh model should be in playing state")
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_viewRendersWon(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	_ = m.View()
}

func TestModel_viewRendersLost(t *testing.T) {
	m := newTestModel()
	m.state = stateLost
	_ = m.View()
}

// helpers

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyMsgSpecial(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
