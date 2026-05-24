package main

import "testing"

//  fuzzyScore

func TestFuzzyScore_EmptyQuery_ReturnsZero(t *testing.T) {
	if got := fuzzyScore("", "anything"); got != 0 {
		t.Errorf("fuzzyScore(%q, %q) = %d, want 0", "", "anything", got)
	}
}

func TestFuzzyScore_EmptyTarget_ReturnsZero(t *testing.T) {
	if got := fuzzyScore("abc", ""); got != 0 {
		t.Errorf("fuzzyScore(%q, %q) = %d, want 0", "abc", "", got)
	}
}

func TestFuzzyScore_NoMatchInTarget_ReturnsZero(t *testing.T) {
	if got := fuzzyScore("xyz", "abc"); got != 0 {
		t.Errorf("fuzzyScore(%q, %q) = %d, want 0", "xyz", "abc", got)
	}
}

func TestFuzzyScore_MatchScoresHigher(t *testing.T) {
	match := fuzzyScore("ab", "ab")
	noMatch := fuzzyScore("xy", "ab")
	if match <= noMatch {
		t.Errorf("match score %d should be > no-match score %d", match, noMatch)
	}
}

func TestFuzzyScore_ConsecutiveBonusHigherThanNonConsecutive(t *testing.T) {
	// "ab" in "xaby" — a at pos 1, b at pos 2 (consecutive)
	consec := fuzzyScore("ab", "xaby")
	// "ab" in "xaxby" — a at pos 1, b at pos 3 (non-consecutive)
	nonConsec := fuzzyScore("ab", "xaxby")
	if consec <= nonConsec {
		t.Errorf("consecutive score %d should be > non-consecutive score %d", consec, nonConsec)
	}
}

func TestFuzzyScore_WordBoundaryHigherThanMidWord(t *testing.T) {
	// 'b' at word boundary ("bar" in "foo bar")
	boundary := fuzzyScore("b", "foo bar")
	// 'b' mid-word in "fob"
	midWord := fuzzyScore("b", "fob")
	if boundary <= midWord {
		t.Errorf("word-boundary score %d should be > mid-word score %d", boundary, midWord)
	}
}

func TestFuzzyScore_CaseInsensitive(t *testing.T) {
	lower := fuzzyScore("ab", "ab")
	upper := fuzzyScore("AB", "ab")
	mixed := fuzzyScore("Ab", "AB")
	if lower != upper || lower != mixed {
		t.Errorf("case should not affect score: lower=%d upper=%d mixed=%d", lower, upper, mixed)
	}
}

func TestFuzzyScore_PartialMatchPenalty(t *testing.T) {
	// "xy" fully matched
	full := fuzzyScore("xy", "xy")
	// "xyz" partially matched (z not in target)
	partial := fuzzyScore("xyz", "xy")
	if partial >= full {
		t.Errorf("partial score %d should be < full match score %d", partial, full)
	}
}

//  runSearch

func TestRunSearch_EmptyQuery_NoMatches(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.searchQuery = ""
	ps.runSearch()
	if len(ps.searchMatches) != 0 {
		t.Errorf("empty query: searchMatches = %v, want nil/empty", ps.searchMatches)
	}
}

func TestRunSearch_QueryWithMatch_PopulatesMatches(t *testing.T) {
	b := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b.Title = "Dune"
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.searchQuery = "dune"
	ps.runSearch()
	if len(ps.searchMatches) == 0 {
		t.Error("query 'dune' should produce at least one match")
	}
}

func TestRunSearch_BestMatchFirst(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b1.Title = "Dune"
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	b2.Title = "Other Book"
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.searchQuery = "dune"
	ps.runSearch()
	if len(ps.searchMatches) == 0 {
		t.Fatal("expected matches for 'dune'")
	}
	books := ps.localBooks()
	first := ps.searchMatches[0]
	if first >= len(books) || books[first].Hash != "a" {
		t.Errorf("first match should be 'Dune' (hash a), got index %d", first)
	}
}

//  jumpToMatch

func TestJumpToMatch_SetsAlbumSelected(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.searchMatches = []int{1, 0}
	ps.jumpToMatch(0)
	if ps.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (searchMatches[0])", ps.albumSelected)
	}
}

func TestJumpToMatch_ResetsTrackCursor(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.trackSelected = 2
	ps.searchMatches = []int{0}
	ps.jumpToMatch(0)
	if ps.trackSelected != 0 {
		t.Errorf("trackSelected = %d, want 0 after jumpToMatch", ps.trackSelected)
	}
}

func TestJumpToMatch_SetsMatchIdx(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.searchMatches = []int{0, 1}
	ps.jumpToMatch(1)
	if ps.searchMatchIdx != 1 {
		t.Errorf("searchMatchIdx = %d, want 1", ps.searchMatchIdx)
	}
}

//  enterSearch

func TestEnterSearch_SetsModeSearch(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.enterSearch()
	if ps.mode != ModeSearch {
		t.Errorf("mode = %v, want ModeSearch", ps.mode)
	}
}

func TestEnterSearch_SavesCursorSnapshot(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.albumSelected = 3
	ps.albumOffset = 1
	ps.enterSearch()
	if ps.searchSavedAlbum != 3 {
		t.Errorf("searchSavedAlbum = %d, want 3", ps.searchSavedAlbum)
	}
	if ps.searchSavedOffset != 1 {
		t.Errorf("searchSavedOffset = %d, want 1", ps.searchSavedOffset)
	}
}

func TestEnterSearch_ClearsQuery(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.searchQuery = "old"
	ps.enterSearch()
	if ps.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty", ps.searchQuery)
	}
}

func TestEnterSearch_ClearsMatches(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.searchMatches = []int{0, 1}
	ps.enterSearch()
	if len(ps.searchMatches) != 0 {
		t.Errorf("searchMatches = %v, want nil/empty", ps.searchMatches)
	}
}

//  cancelSearch

func TestCancelSearch_RestoresCursor(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.searchSavedAlbum = 2
	ps.searchSavedOffset = 1
	ps.albumSelected = 5
	ps.albumOffset = 4
	ps.cancelSearch()
	if ps.albumSelected != 2 {
		t.Errorf("albumSelected = %d, want 2", ps.albumSelected)
	}
	if ps.albumOffset != 1 {
		t.Errorf("albumOffset = %d, want 1", ps.albumOffset)
	}
}

func TestCancelSearch_ClearsQueryAndMatches(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.searchQuery = "foo"
	ps.searchMatches = []int{0, 1}
	ps.cancelSearch()
	if ps.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty", ps.searchQuery)
	}
	if len(ps.searchMatches) != 0 {
		t.Errorf("searchMatches = %v, want nil/empty", ps.searchMatches)
	}
}

func TestCancelSearch_ReturnsToMain(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.cancelSearch()
	if ps.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps.mode)
	}
}

//  confirmSearch

func TestConfirmSearch_WithMatches_TransitionsToSearching(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = "foo"
	ps.searchMatches = []int{0}
	ps.confirmSearch()
	if ps.mode != ModeSearching {
		t.Errorf("mode = %v, want ModeSearching", ps.mode)
	}
}

func TestConfirmSearch_EmptyQuery_DoesNotEnterSearching(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = ""
	ps.confirmSearch()
	if ps.mode == ModeSearching {
		t.Error("empty query should not transition to ModeSearching")
	}
}

func TestConfirmSearch_NoMatches_StillEntersSearching(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = "noresults"
	ps.searchMatches = nil
	ps.confirmSearch()
	if ps.mode != ModeSearching {
		t.Errorf("mode = %v, want ModeSearching (show [0/0] until esc)", ps.mode)
	}
}

//  exitSearchKeep

func TestExitSearchKeep_KeepsCurrentPosition(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearching
	ps.albumSelected = 3
	ps.searchSavedAlbum = 1
	ps.exitSearchKeep()
	if ps.albumSelected != 3 {
		t.Errorf("albumSelected = %d, want 3 (kept)", ps.albumSelected)
	}
}

func TestExitSearchKeep_ReturnsToMain(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearching
	ps.exitSearchKeep()
	if ps.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps.mode)
	}
}

func TestExitSearchKeep_ClearsQueryAndMatches(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearching
	ps.searchQuery = "foo"
	ps.searchMatches = []int{0, 1}
	ps.exitSearchKeep()
	if ps.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty", ps.searchQuery)
	}
	if len(ps.searchMatches) != 0 {
		t.Errorf("searchMatches = %v, want nil/empty", ps.searchMatches)
	}
}
