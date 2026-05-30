package main

import (
	"testing"
)

func makeLibBook(hash, title, author string, date int, genres []string) Audiobook {
	b := makeBook(hash, DownloadReady, nil, nil)
	b.Title = title
	b.Author = author
	b.Date = date
	b.Genres = genres
	return b
}

// parseLibQuery

func TestParseLibQuery_EmptyRaw_ReturnsEmpty(t *testing.T) {
	q := parseLibQuery("")
	if q.Text != "" || len(q.Tags) != 0 || q.YearLow != nil || q.YearHigh != nil {
		t.Errorf("parseLibQuery(%q) = %+v, want zero value", "", q)
	}
}

func TestParseLibQuery_PlainText_ReturnsText(t *testing.T) {
	q := parseLibQuery("dune herbert")
	if q.Text != "dune herbert" {
		t.Errorf("Text = %q, want %q", q.Text, "dune herbert")
	}
	if len(q.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", q.Tags)
	}
}

func TestParseLibQuery_SingleTag_ExtractedAndRemovedFromText(t *testing.T) {
	q := parseLibQuery("dune +scifi")
	if q.Text != "dune" {
		t.Errorf("Text = %q, want %q", q.Text, "dune")
	}
	if len(q.Tags) != 1 || q.Tags[0] != "scifi" {
		t.Errorf("Tags = %v, want [scifi]", q.Tags)
	}
}

func TestParseLibQuery_MultipleTags(t *testing.T) {
	q := parseLibQuery("+history +sociology")
	if len(q.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(q.Tags))
	}
	if q.Tags[0] != "history" || q.Tags[1] != "sociology" {
		t.Errorf("Tags = %v, want [history sociology]", q.Tags)
	}
	if q.Text != "" {
		t.Errorf("Text = %q, want empty", q.Text)
	}
}

func TestParseLibQuery_YearLow_TakesMax(t *testing.T) {
	q := parseLibQuery(">1850 >1900")
	if q.YearLow == nil || *q.YearLow != 1900 {
		t.Errorf("YearLow = %v, want 1900", q.YearLow)
	}
}

func TestParseLibQuery_YearHigh_TakesMin(t *testing.T) {
	q := parseLibQuery("<2000 <1990")
	if q.YearHigh == nil || *q.YearHigh != 1990 {
		t.Errorf("YearHigh = %v, want 1990", q.YearHigh)
	}
}

func TestParseLibQuery_DateRange_NarrowsInterval(t *testing.T) {
	q := parseLibQuery(">1900 <2000 <1990 >1850")
	if q.YearLow == nil || *q.YearLow != 1900 {
		t.Errorf("YearLow = %v, want 1900", q.YearLow)
	}
	if q.YearHigh == nil || *q.YearHigh != 1990 {
		t.Errorf("YearHigh = %v, want 1990", q.YearHigh)
	}
}

func TestParseLibQuery_LoneFilterChars_NotInText(t *testing.T) {
	for _, q := range []string{"<", ">", "+"} {
		got := parseLibQuery(q)
		if got.Text != "" {
			t.Errorf("parseLibQuery(%q).Text = %q, want empty (lone filter char stripped)", q, got.Text)
		}
	}
}

func TestParseLibQuery_PartialFilterToken_NotInText(t *testing.T) {
	// "<20" is a partial year filter being typed — should not appear in text
	q := parseLibQuery("<20")
	if q.Text != "" {
		t.Errorf("Text = %q, want empty (<20 partial filter stripped)", q.Text)
	}
}

func TestParseLibQuery_FilterCharMiddleOfWord_Kept(t *testing.T) {
	// "abc<def" is in the middle of a word — literal text, not a filter
	q := parseLibQuery("abc<def")
	if q.Text != "abc<def" {
		t.Errorf("Text = %q, want %q (< in middle of word kept)", q.Text, "abc<def")
	}
}

func TestParseLibQuery_FilterCharWithSpaceAfter_Stripped(t *testing.T) {
	// "< dune" — the "<" is a lone word at a boundary; the text "dune" remains
	q := parseLibQuery("< dune")
	if q.Text != "dune" {
		t.Errorf("Text = %q, want %q (lone < stripped, dune remains)", q.Text, "dune")
	}
}

func TestRunLibSearch_LoneFilterChar_DoesNotEmptyList(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libQuery = "<"
	ps.runLibSearch()
	if len(ps.libMatches) == 0 {
		t.Error("lone '<' should not filter out all books")
	}
}

func TestParseLibQuery_Combined_AllParsed(t *testing.T) {
	q := parseLibQuery("dune +scifi >1960 <1970")
	if q.Text != "dune" {
		t.Errorf("Text = %q, want %q", q.Text, "dune")
	}
	if len(q.Tags) != 1 || q.Tags[0] != "scifi" {
		t.Errorf("Tags = %v, want [scifi]", q.Tags)
	}
	if q.YearLow == nil || *q.YearLow != 1960 {
		t.Errorf("YearLow = %v, want 1960", q.YearLow)
	}
	if q.YearHigh == nil || *q.YearHigh != 1970 {
		t.Errorf("YearHigh = %v, want 1970", q.YearHigh)
	}
}

// libTextScore

func TestLibTextScore_EmptyText_ReturnsOne(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("", b); got != 1 {
		t.Errorf("libTextScore(%q) = %d, want 1", "", got)
	}
}

func TestLibTextScore_NoSep_MatchesTitle(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("dune", b); got == 0 {
		t.Error("libTextScore should match title 'Dune'")
	}
}

func TestLibTextScore_NoSep_MatchesAuthor(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("herb", b); got == 0 {
		t.Error("libTextScore should match author 'Herbert'")
	}
}

func TestLibTextScore_NoSep_BidirectionalOrderBothWork(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	s1 := libTextScore("dune herbert", b)
	s2 := libTextScore("herbert dune", b)
	if s1 == 0 || s2 == 0 {
		t.Errorf("both orderings should score > 0: s1=%d s2=%d", s1, s2)
	}
}

func TestLibTextScore_NoSep_NoMatch_ReturnsZero(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("zzzzz", b); got != 0 {
		t.Errorf("libTextScore no match = %d, want 0", got)
	}
}

func TestLibTextScore_WithSep_TitleAuthorOrder(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("dune - herbert", b); got == 0 {
		t.Error("libTextScore 'title - author' should score > 0")
	}
}

func TestLibTextScore_WithSep_AuthorTitleOrder(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("herbert - dune", b); got == 0 {
		t.Error("libTextScore 'author - title' should score > 0")
	}
}

func TestLibTextScore_WithSep_LeftNoMatch_ReturnsZero(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("zzz - herbert", b); got != 0 {
		t.Errorf("libTextScore left no match = %d, want 0", got)
	}
}

func TestLibTextScore_WithSep_RightNoMatch_ReturnsZero(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	if got := libTextScore("dune - zzz", b); got != 0 {
		t.Errorf("libTextScore right no match = %d, want 0", got)
	}
}

// buildTagCandidates

func TestBuildTagCandidates_ExactMatch_Returned(t *testing.T) {
	genres := []string{"History", "Philosophy", "Science"}
	got := buildTagCandidates("History", genres)
	found := false
	for _, g := range got {
		if g == "History" {
			found = true
		}
	}
	if !found {
		t.Errorf("buildTagCandidates exact match: got %v, want to contain 'History'", got)
	}
}

func TestBuildTagCandidates_FuzzyMatch_Returned(t *testing.T) {
	genres := []string{"History", "Philosophy", "Science"}
	got := buildTagCandidates("his", genres)
	found := false
	for _, g := range got {
		if g == "History" {
			found = true
		}
	}
	if !found {
		t.Errorf("buildTagCandidates fuzzy 'his': got %v, want to contain 'History'", got)
	}
}

func TestBuildTagCandidates_NoMatch_EmptyResult(t *testing.T) {
	genres := []string{"History", "Philosophy", "Science"}
	got := buildTagCandidates("zzz", genres)
	if len(got) != 0 {
		t.Errorf("buildTagCandidates no match: got %v, want empty", got)
	}
}

func TestBuildTagCandidates_CaseInsensitive(t *testing.T) {
	genres := []string{"History"}
	got := buildTagCandidates("history", genres)
	if len(got) == 0 {
		t.Error("buildTagCandidates should be case-insensitive")
	}
}

// scoreBookTags

func TestScoreBookTags_EmptyTagLists_ReturnsZero(t *testing.T) {
	got := scoreBookTags([]string{"History"}, [][]string{})
	if got != 0 {
		t.Errorf("scoreBookTags empty tagLists = %d, want 0", got)
	}
}

func TestScoreBookTags_OneList_BookHasMatch_ReturnsOne(t *testing.T) {
	tagLists := [][]string{{"History", "Philosophy"}}
	got := scoreBookTags([]string{"History", "Music"}, tagLists)
	if got != 1 {
		t.Errorf("scoreBookTags single match = %d, want 1", got)
	}
}

func TestScoreBookTags_OneList_BookNoMatch_ReturnsZero(t *testing.T) {
	tagLists := [][]string{{"History", "Philosophy"}}
	got := scoreBookTags([]string{"Music", "Science"}, tagLists)
	if got != 0 {
		t.Errorf("scoreBookTags no match = %d, want 0", got)
	}
}

func TestScoreBookTags_TwoLists_BothMatched_ReturnsTwo(t *testing.T) {
	tagLists := [][]string{{"History"}, {"Philosophy"}}
	got := scoreBookTags([]string{"History", "Philosophy"}, tagLists)
	if got != 2 {
		t.Errorf("scoreBookTags both matched = %d, want 2", got)
	}
}

func TestScoreBookTags_TwoSameLists_BookHasOneGenre_ReturnsOne(t *testing.T) {
	// +oy matches [history, sociology], +oy again same list; book has only history
	tagLists := [][]string{{"History", "Sociology"}, {"History", "Sociology"}}
	got := scoreBookTags([]string{"History"}, tagLists)
	if got != 1 {
		t.Errorf("scoreBookTags one genre, two overlapping lists = %d, want 1", got)
	}
}

func TestScoreBookTags_TwoSameLists_BookHasBothGenres_ReturnsTwo(t *testing.T) {
	tagLists := [][]string{{"History", "Sociology"}, {"History", "Sociology"}}
	got := scoreBookTags([]string{"History", "Sociology"}, tagLists)
	if got != 2 {
		t.Errorf("scoreBookTags both genres match both lists = %d, want 2", got)
	}
}

func TestScoreBookTags_NoReuseOfBookGenre(t *testing.T) {
	// Book has only "History"; two lists both contain "History"
	// History can only satisfy one constraint → score = 1, not 2
	tagLists := [][]string{{"History"}, {"History"}}
	got := scoreBookTags([]string{"History"}, tagLists)
	if got != 1 {
		t.Errorf("scoreBookTags no-reuse: history used twice = %d, want 1", got)
	}
}

// runLibSearch

func TestRunLibSearch_EmptyQuery_NoMatches(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, []string{"Science Fiction"})
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libQuery = ""
	ps.runLibSearch()
	if len(ps.libMatches) != 0 {
		t.Errorf("empty query: libMatches = %v, want nil/empty", ps.libMatches)
	}
}

func TestRunLibSearch_TextMatch_IncludesBook(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libQuery = "dune"
	ps.runLibSearch()
	if len(ps.libMatches) == 0 {
		t.Error("'dune' should match book with title Dune")
	}
}

func TestRunLibSearch_TextNoMatch_ExcludesBook(t *testing.T) {
	b := makeLibBook("a", "Dune", "Herbert", 1965, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libQuery = "zzzzz"
	ps.runLibSearch()
	if len(ps.libMatches) != 0 {
		t.Errorf("no match: libMatches = %v, want empty", ps.libMatches)
	}
}

func TestRunLibSearch_DateLowerBound_FiltersOldBooks(t *testing.T) {
	old := makeLibBook("a", "Old Book", "Auth", 1800, nil)
	new := makeLibBook("b", "New Book", "Auth", 1950, nil)
	ps := newPlayerState(makeLib("", []Audiobook{old, new}))
	ps.libQuery = ">1900"
	ps.runLibSearch()
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "a" {
			t.Error("old book (1800) should be filtered by >1900")
		}
	}
	found := false
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "b" {
			found = true
		}
	}
	if !found {
		t.Error("new book (1950) should pass >1900 filter")
	}
}

func TestRunLibSearch_DateUpperBound_FiltersNewBooks(t *testing.T) {
	old := makeLibBook("a", "Old Book", "Auth", 1800, nil)
	new := makeLibBook("b", "New Book", "Auth", 1950, nil)
	ps := newPlayerState(makeLib("", []Audiobook{old, new}))
	ps.libQuery = "<1900"
	ps.runLibSearch()
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "b" {
			t.Error("new book (1950) should be filtered by <1900")
		}
	}
}

func TestRunLibSearch_DateBoundsInclusive(t *testing.T) {
	b := makeLibBook("a", "Book", "Auth", 1900, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libQuery = ">1900 <1900"
	ps.runLibSearch()
	if len(ps.libMatches) == 0 {
		t.Error("book at exact boundary 1900 should pass >1900 <1900 (inclusive)")
	}
}

func TestRunLibSearch_TagFilter_FiltersNonMatchingBooks(t *testing.T) {
	hist := makeLibBook("a", "History Book", "Auth", 1900, []string{"History"})
	sci := makeLibBook("b", "Science Book", "Auth", 1900, []string{"Science"})
	ps := newPlayerState(makeLib("", []Audiobook{hist, sci}))
	ps.libQuery = "+History"
	ps.runLibSearch()
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "b" {
			t.Error("science book should be filtered by +History")
		}
	}
	found := false
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "a" {
			found = true
		}
	}
	if !found {
		t.Error("history book should pass +History filter")
	}
}

func TestRunLibSearch_TagMultiplier_BothBooksIncluded(t *testing.T) {
	one := makeLibBook("a", "One Tag", "Auth", 1900, []string{"History"})
	two := makeLibBook("b", "Two Tags", "Auth", 1900, []string{"History", "Philosophy"})
	ps := newPlayerState(makeLib("", []Audiobook{one, two}))
	ps.libQuery = "+History +Philosophy"
	ps.runLibSearch()
	if len(ps.libMatches) < 2 {
		t.Fatalf("expected 2 matches (both books have at least one matching tag), got %d", len(ps.libMatches))
	}
}

func TestRunLibSearch_PreservesOriginalOrder(t *testing.T) {
	// "Bicarbonate" scores lower for "ca" than "Calcium" (no word boundary for 'c' in Bicarbonate),
	// so score-order would be [1, 0] but original-order should be [0, 1].
	a := makeLibBook("a", "Bicarbonate", "Auth", 1900, nil)
	b := makeLibBook("b", "Calcium", "Auth", 1900, nil)
	ps := newPlayerState(makeLib("", []Audiobook{a, b}))
	ps.libQuery = "ca"
	ps.runLibSearch()
	if len(ps.libMatches) < 2 {
		t.Fatalf("expected 2 matches, got %d", len(ps.libMatches))
	}
	if ps.libMatches[0] != 0 {
		t.Errorf("libMatches[0] = %d, want 0 (original order preserved, not score order)", ps.libMatches[0])
	}
	if ps.libMatches[1] != 1 {
		t.Errorf("libMatches[1] = %d, want 1 (original order preserved)", ps.libMatches[1])
	}
}

func TestRunLibSearch_Combined_TextAndTag(t *testing.T) {
	match := makeLibBook("a", "Dune", "Herbert", 1965, []string{"Science Fiction"})
	noTag := makeLibBook("b", "Dune Again", "Herbert", 1965, []string{"History"})
	ps := newPlayerState(makeLib("", []Audiobook{match, noTag}))
	ps.libQuery = "dune +Science"
	ps.runLibSearch()
	for _, idx := range ps.libMatches {
		if ps.lib.Books[idx].Hash == "b" {
			t.Error("book without science tag should be filtered out by +Science")
		}
	}
}

// enterLibSearch / cancelLibSearch / confirmLibSearch / exitLibSearchKeep

func TestEnterLibSearch_SetsModeLibSearch(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.enterLibSearch()
	if ps.mode != ModeLibSearch {
		t.Errorf("mode = %v, want ModeLibSearch", ps.mode)
	}
}

func TestEnterLibSearch_SavesCursor(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libSel = 3
	ps.libOffset = 2
	ps.enterLibSearch()
	if ps.libSavedSel != 3 {
		t.Errorf("libSavedSel = %d, want 3", ps.libSavedSel)
	}
	if ps.libSavedOffset != 2 {
		t.Errorf("libSavedOffset = %d, want 2", ps.libSavedOffset)
	}
}

func TestEnterLibSearch_ClearsState(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libQuery = "old"
	ps.libMatches = []int{0}
	ps.enterLibSearch()
	if ps.libQuery != "" {
		t.Errorf("libQuery = %q, want empty", ps.libQuery)
	}
	if len(ps.libMatches) != 0 {
		t.Errorf("libMatches = %v, want empty", ps.libMatches)
	}
}

func TestCancelLibSearch_RestoresCursor(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libSavedSel = 3
	ps.libSavedOffset = 2
	ps.libSel = 7
	ps.libOffset = 5
	ps.cancelLibSearch()
	if ps.libSel != 3 {
		t.Errorf("libSel = %d, want 3", ps.libSel)
	}
	if ps.libOffset != 2 {
		t.Errorf("libOffset = %d, want 2", ps.libOffset)
	}
}

func TestCancelLibSearch_ReturnsToMain(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeLibSearch
	ps.cancelLibSearch()
	if ps.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps.mode)
	}
}

func TestCancelLibSearch_ClearsState(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libQuery = "foo"
	ps.libMatches = []int{0}
	ps.cancelLibSearch()
	if ps.libQuery != "" || len(ps.libMatches) != 0 {
		t.Errorf("libQuery=%q libMatches=%v, want both empty", ps.libQuery, ps.libMatches)
	}
}

func TestConfirmLibSearch_NonEmpty_TransitionsToLibSearching(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeLibSearch
	ps.libQuery = "dune"
	ps.confirmLibSearch()
	if ps.mode != ModeLibSearching {
		t.Errorf("mode = %v, want ModeLibSearching", ps.mode)
	}
}

func TestConfirmLibSearch_Empty_Cancels(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeLibSearch
	ps.libQuery = ""
	ps.confirmLibSearch()
	if ps.mode == ModeLibSearching {
		t.Error("empty query should not enter ModeLibSearching")
	}
}

func TestExitLibSearchKeep_KeepsLibSel(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeLibSearching
	ps.libSel = 4
	ps.libSavedSel = 1
	ps.exitLibSearchKeep()
	if ps.libSel != 4 {
		t.Errorf("libSel = %d, want 4 (kept)", ps.libSel)
	}
}

func TestExitLibSearchKeep_ReturnsToMain(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeLibSearching
	ps.exitLibSearchKeep()
	if ps.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps.mode)
	}
}

func TestExitLibSearchKeep_ClearsState(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libQuery = "foo"
	ps.libMatches = []int{0}
	ps.exitLibSearchKeep()
	if ps.libQuery != "" || len(ps.libMatches) != 0 {
		t.Errorf("libQuery=%q libMatches=%v, want both empty", ps.libQuery, ps.libMatches)
	}
}

// jumpToLibMatch

func TestJumpToLibMatch_SetsLibSel(t *testing.T) {
	b1 := makeLibBook("a", "Book A", "Auth", 1900, nil)
	b2 := makeLibBook("b", "Book B", "Auth", 1900, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.libMatches = []int{1, 0}
	ps.jumpToLibMatch(0)
	if ps.libSel != 1 {
		t.Errorf("libSel = %d, want 1", ps.libSel)
	}
}

func TestJumpToLibMatch_SetsMatchIdx(t *testing.T) {
	b1 := makeLibBook("a", "Book A", "Auth", 1900, nil)
	b2 := makeLibBook("b", "Book B", "Auth", 1900, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.libMatches = []int{0, 1}
	ps.jumpToLibMatch(1)
	if ps.libMatchIdx != 1 {
		t.Errorf("libMatchIdx = %d, want 1", ps.libMatchIdx)
	}
}

func TestJumpToLibMatch_ResetsLibInfoOff(t *testing.T) {
	b := makeLibBook("a", "Book A", "Auth", 1900, nil)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.libInfoOff = 5
	ps.libMatches = []int{0}
	ps.jumpToLibMatch(0)
	if ps.libInfoOff != 0 {
		t.Errorf("libInfoOff = %d, want 0", ps.libInfoOff)
	}
}

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
