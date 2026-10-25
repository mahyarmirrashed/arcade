package pasur

// captures returns all subsets of pool that card c can capture, following
// Pasur rules:
//
//   - A Jack captures all Jacks and all number cards (not Kings/Queens).
//   - A King captures one King; a Queen captures one Queen.
//   - Number cards (A=1 through 10) capture any subset that sums to 11.
//
// Returns a slice of capture options; each option is a slice of indices into
// pool. The caller chooses which option to play (the player must capture if
// any capture is possible).
func captures(c Card, pool []Card) [][]int {
	switch {
	case c.Rank == Jack:
		return jackCaptures(pool)
	case c.Rank == King:
		return faceCaptures(pool, King)
	case c.Rank == Queen:
		return faceCaptures(pool, Queen)
	default:
		return sumCaptures(c, pool)
	}
}

// canCapture reports whether card c can capture anything in pool.
func canCapture(c Card, pool []Card) bool {
	return len(captures(c, pool)) > 0
}

// jackCaptures returns the single capture set for a Jack: all Jacks and
// all number cards in pool. Returns nil if there is nothing to capture.
func jackCaptures(pool []Card) [][]int {
	var indices []int
	for i, p := range pool {
		if p.Rank == Jack || p.Rank < Jack {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return nil
	}
	return [][]int{indices}
}

// faceCaptures returns capture options for a King or Queen: one matching card.
func faceCaptures(pool []Card, rank Rank) [][]int {
	var opts [][]int
	for i, p := range pool {
		if p.Rank == rank {
			opts = append(opts, []int{i})
		}
	}
	return opts
}

// sumCaptures returns all subsets of pool whose pip values sum to
// (11 - c.pipValue()), i.e. together with c they sum to 11.
// Only number cards (Ace through 10) in pool count; face cards are skipped.
func sumCaptures(c Card, pool []Card) [][]int {
	target := 11 - c.Rank.pipValue()
	if target <= 0 {
		return nil
	}

	// Collect eligible (number) cards from pool with their indices.
	type entry struct {
		idx int
		val int
	}
	var eligible []entry
	for i, p := range pool {
		if v := p.Rank.pipValue(); v > 0 {
			eligible = append(eligible, entry{i, v})
		}
	}

	var opts [][]int
	// Enumerate all non-empty subsets.
	for mask := 1; mask < (1 << len(eligible)); mask++ {
		sum := 0
		var idxs []int
		for bit, e := range eligible {
			if mask&(1<<bit) != 0 {
				sum += e.val
				idxs = append(idxs, e.idx)
			}
		}
		if sum == target {
			opts = append(opts, idxs)
		}
	}
	return opts
}

// isSur reports whether a capture clears the entire pool (a Sur).
// A Sur is NOT counted when:
//   - it was caused by a Jack
//   - it happens on the last deal of the deck
func isSur(card Card, capturedCount, poolSize int, lastDeal bool) bool {
	if lastDeal {
		return false
	}
	if card.Rank == Jack {
		return false
	}
	return capturedCount == poolSize
}

// Score holds a player's accumulated points for one full game (multiple rounds).
type Score struct {
	Points int // total leaderboard points
	Surs   int // number of surs scored (each worth 5 points at end of round)
}

// roundScore computes the points earned from a round's captured cards.
// mostClubs is true if this player captured more clubs than their opponent.
// surCount is how many surs this player scored in the round.
func roundScore(captured []Card, mostClubs bool, surCount int) int {
	pts := 0
	if mostClubs {
		pts += 7
	}
	for _, c := range captured {
		pts += c.scoreValue()
	}
	pts += surCount * 5
	return pts
}

// clubCount counts the number of Club cards in a slice.
func clubCount(cards []Card) int {
	n := 0
	for _, c := range cards {
		if c.Suit == Clubs {
			n++
		}
	}
	return n
}
