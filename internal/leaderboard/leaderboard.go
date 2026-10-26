// Package leaderboard keeps the arcade's high scores: a display name for each
// player's SSH key, and each player's best result in every game. The board is
// saved to a JSON file after every change, so it survives server restarts.
package leaderboard

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mahyarmirrashed/arcade/internal/game"
)

// Errors returned by Session.SetName.
var (
	ErrNameInvalid = errors.New("names are 3-12 letters, digits, - or _")
	ErrNameTaken   = errors.New("that name is taken")
	ErrNoKey       = errors.New("connect with an ssh key to choose a name")
)

// validName keeps names to characters that are safe to print on every
// player's terminal; anything else could smuggle in escape codes.
var validName = regexp.MustCompile(`^[A-Za-z0-9_-]{3,12}$`)

// reserved names can't be chosen by anyone.
var reserved = map[string]bool{"guest": true}

const version = 1

// data is the saved form of the board.
type data struct {
	Version int                          `json:"version"`
	Players map[string]player            `json:"players"` // by key fingerprint
	Games   map[string]map[string]record `json:"games"`   // by game ID, then fingerprint
}

type player struct {
	Name   string    `json:"name"`
	Joined time.Time `json:"joined"`
}

// record is one player's results in one game. Which fields are used depends on
// how the game is ranked.
type record struct {
	Best       int           `json:"best,omitempty"`       // ByScore
	Streak     int           `json:"streak,omitempty"`     // ByStreak: current run of wins
	BestStreak int           `json:"bestStreak,omitempty"` // ByStreak
	Wins       int           `json:"wins,omitempty"`       // ByStreak
	Guesses    int           `json:"guesses,omitempty"`    // ByStreak: total over all wins
	BestTime   time.Duration `json:"bestTime,omitempty"`   // ByTime
	At         time.Time     `json:"at"`                   // when the best was set
}

// Board is the arcade's leaderboard. It is safe for concurrent use by many
// sessions. Only one process should open a given file.
type Board struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
	data data
}

// Open loads the board saved at path, or starts an empty one if there is no
// file yet. A file that can't be read is an error rather than a fresh start,
// so a damaged file is never overwritten with an empty board.
func Open(path string) (*Board, error) {
	b := &Board{path: path, now: time.Now, data: data{Version: version}}

	buf, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("reading leaderboard: %w", err)
	default:
		if err := json.Unmarshal(buf, &b.data); err != nil {
			return nil, fmt.Errorf("leaderboard %s is unreadable; fix or move it aside: %w", path, err)
		}
		if b.data.Version != version {
			return nil, fmt.Errorf("leaderboard %s is version %d; this server reads version %d",
				path, b.data.Version, version)
		}
	}

	if b.data.Players == nil {
		b.data.Players = map[string]player{}
	}
	if b.data.Games == nil {
		b.data.Games = map[string]map[string]record{}
	}
	return b, nil
}

// save writes the board to a temporary file and renames it over the old one,
// so a crash mid-write leaves the previous board intact. Callers hold b.mu.
func (b *Board) save() error {
	buf, err := json.MarshalIndent(b.data, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".leaderboard-*.tmp") // created with mode 0600
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }() // fails harmlessly once renamed

	if _, err := f.Write(buf); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}

// Session is one player's view of the board. The zero Session has no board: it
// records nothing and lists nothing.
type Session struct {
	board *Board
	id    string // SSH key fingerprint; "" for a guest without a key
}

// Session returns the view for the player with key fingerprint id, or for a
// guest if id is "".
func (b *Board) Session(id string) Session { return Session{board: b, id: id} }

// Enabled reports whether the session has a leaderboard to record on and
// list. Guests on a board are enabled; they just aren't ranked.
func (s Session) Enabled() bool { return s.board != nil }

// Name returns the player's display name, or "" if they have none.
func (s Session) Name() string {
	if s.board == nil || s.id == "" {
		return ""
	}
	s.board.mu.Lock()
	defer s.board.mu.Unlock()
	return s.board.data.Players[s.id].Name
}

// NeedsName reports whether the player has a key but hasn't chosen a name.
func (s Session) NeedsName() bool {
	return s.board != nil && s.id != "" && s.Name() == ""
}

// SetName gives the player a display name. Names are unique, ignoring case.
func (s Session) SetName(name string) error {
	if s.board == nil || s.id == "" {
		return ErrNoKey
	}
	if !validName.MatchString(name) {
		return ErrNameInvalid
	}
	if reserved[strings.ToLower(name)] {
		return ErrNameTaken
	}

	b := s.board
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, p := range b.data.Players {
		if id != s.id && strings.EqualFold(p.Name, name) {
			return ErrNameTaken
		}
	}
	p, ok := b.data.Players[s.id]
	if !ok {
		p.Joined = b.now()
	}
	p.Name = name
	b.data.Players[s.id] = p
	return b.save()
}

// Record adds a finished game's result and returns where the player now
// stands. Guests and players without a name are unranked and nothing is kept.
func (s Session) Record(gameID string, r game.Ranking, res game.Result) (game.Standing, error) {
	if s.board == nil || s.id == "" {
		return game.Standing{}, nil
	}
	b := s.board
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.data.Players[s.id].Name == "" {
		return game.Standing{}, nil
	}

	recs := b.data.Games[gameID]
	if recs == nil {
		recs = map[string]record{}
		b.data.Games[gameID] = recs
	}
	rec, existed := recs[s.id]
	newBest := apply(&rec, existed, r, res, b.now())
	recs[s.id] = rec

	rank, of := b.rank(gameID, r, s.id)
	return game.Standing{Ranked: true, NewBest: newBest, Rank: rank, Of: of, Best: rec.Best}, b.save()
}

// apply folds a result into rec and reports whether it set a new best.
func apply(rec *record, existed bool, r game.Ranking, res game.Result, now time.Time) bool {
	switch r {
	case game.ByStreak:
		if !res.Won {
			rec.Streak = 0
			return false
		}
		rec.Streak++
		rec.Wins++
		rec.Guesses += res.Guesses
		if rec.Streak <= rec.BestStreak {
			return false
		}
		rec.BestStreak = rec.Streak
	case game.ByTime:
		if !res.Won || (rec.BestTime > 0 && res.Duration >= rec.BestTime) {
			return false
		}
		rec.BestTime = res.Duration
	default:
		if existed && res.Score <= rec.Best {
			return false
		}
		rec.Best = res.Score
	}
	rec.At = now
	return true
}

// Entry is one row of a leaderboard.
type Entry struct {
	Rank  int
	Name  string
	Value string // the ranked result, formatted for display
	You   bool   // the row is the session's own player
}

// Top returns the best n players for a game.
func (s Session) Top(gameID string, r game.Ranking, n int) []Entry {
	if s.board == nil {
		return nil
	}
	b := s.board
	b.mu.Lock()
	defer b.mu.Unlock()

	order := b.ordered(gameID, r)
	if len(order) > n {
		order = order[:n]
	}
	entries := make([]Entry, len(order))
	for i, p := range order {
		entries[i] = Entry{
			Rank:  i + 1,
			Name:  b.data.Players[p.id].Name,
			Value: format(r, p.rec),
			You:   s.id != "" && p.id == s.id,
		}
	}
	return entries
}

// Rank returns the player's position on a game's board and how many players
// are on it. rank is 0 if the player isn't on the board.
func (s Session) Rank(gameID string, r game.Ranking) (rank, of int) {
	if s.board == nil {
		return 0, 0
	}
	s.board.mu.Lock()
	defer s.board.mu.Unlock()
	return s.board.rank(gameID, r, s.id)
}

type ranked struct {
	id  string
	rec record
}

// ordered returns the named players on a game's board, best first. Callers
// hold b.mu.
func (b *Board) ordered(gameID string, r game.Ranking) []ranked {
	var out []ranked
	for id, rec := range b.data.Games[gameID] {
		if qualifies(r, rec) && b.data.Players[id].Name != "" {
			out = append(out, ranked{id, rec})
		}
	}
	slices.SortFunc(out, func(a, c ranked) int {
		if n := compare(r, a.rec, c.rec); n != 0 {
			return n
		}
		return cmp.Compare(a.id, c.id) // keep the order stable
	})
	return out
}

// rank finds id in a game's ordering. Callers hold b.mu.
func (b *Board) rank(gameID string, r game.Ranking, id string) (rank, of int) {
	order := b.ordered(gameID, r)
	for i, p := range order {
		if p.id == id {
			return i + 1, len(order)
		}
	}
	return 0, len(order)
}

// qualifies reports whether a record belongs on the board: score games list
// everyone who has played; streak and time games need a win.
func qualifies(r game.Ranking, rec record) bool {
	switch r {
	case game.ByStreak:
		return rec.BestStreak > 0
	case game.ByTime:
		return rec.BestTime > 0
	default:
		return true
	}
}

// compare orders a before b when a ranks higher. Ties go to whoever set their
// best first.
func compare(r game.Ranking, a, b record) int {
	switch r {
	case game.ByStreak:
		if n := cmp.Compare(b.BestStreak, a.BestStreak); n != 0 {
			return n
		}
		// Fewer guesses per win ranks higher: a.Guesses/a.Wins against
		// b.Guesses/b.Wins, cross-multiplied to stay in integers.
		if n := cmp.Compare(a.Guesses*b.Wins, b.Guesses*a.Wins); n != 0 {
			return n
		}
	case game.ByTime:
		if n := cmp.Compare(a.BestTime, b.BestTime); n != 0 {
			return n
		}
	default:
		if n := cmp.Compare(b.Best, a.Best); n != 0 {
			return n
		}
	}
	return a.At.Compare(b.At)
}

func format(r game.Ranking, rec record) string {
	switch r {
	case game.ByStreak:
		return strconv.Itoa(rec.BestStreak)
	case game.ByTime:
		d := rec.BestTime.Round(time.Second)
		return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return strconv.Itoa(rec.Best)
	}
}

// Label names what a game's board ranks, for column headings.
func Label(r game.Ranking) string {
	switch r {
	case game.ByStreak:
		return "best streak"
	case game.ByTime:
		return "fastest win"
	default:
		return "best score"
	}
}
