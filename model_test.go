package main

import (
	"errors"
	"math"
	"testing"
)

//  newPlayerState 

func TestNewPlayerState_Defaults(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	if ps.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps.mode)
	}
	if !ps.onAlbum {
		t.Error("onAlbum = false, want true")
	}
	if !ps.libInfoOpen {
		t.Error("libInfoOpen = false, want true")
	}
	if ps.playerSpeed != 1.0 {
		t.Errorf("playerSpeed = %v, want 1.0", ps.playerSpeed)
	}
	if ps.playerVolume != 100 {
		t.Errorf("playerVolume = %v, want 100", ps.playerVolume)
	}
}

func TestNewPlayerState_PlayingHashMatch(t *testing.T) {
	books := []Audiobook{
		makeBook("h1", DownloadReady, nil, nil),
		makeBook("h2", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("h2", books))
	if ps.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1", ps.albumSelected)
	}
	if ps.albumPlaying == nil || *ps.albumPlaying != 1 {
		t.Errorf("albumPlaying = %v, want &1", ps.albumPlaying)
	}
}

func TestNewPlayerState_PlayingHashWithPosition(t *testing.T) {
	pos := &Position{ChapterIndex: 3, Timestamp: 1}
	books := []Audiobook{
		makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), pos),
	}
	ps := newPlayerState(makeLib("h1", books))
	if ps.trackSelected != 3 {
		t.Errorf("trackSelected = %d, want 3", ps.trackSelected)
	}
	if ps.playerChSel != 3 {
		t.Errorf("playerChSel = %d, want 3", ps.playerChSel)
	}
	if ps.trackPlaying == nil || *ps.trackPlaying != 3 {
		t.Errorf("trackPlaying = %v, want &3", ps.trackPlaying)
	}
}

func TestNewPlayerState_PlayingHashNoMatch(t *testing.T) {
	books := []Audiobook{makeBook("h1", DownloadReady, nil, nil)}
	ps := newPlayerState(makeLib("nope", books))
	if ps.albumPlaying != nil {
		t.Errorf("albumPlaying = %v, want nil", ps.albumPlaying)
	}
}

func TestNewPlayerState_PlayingHashNonLocalBook(t *testing.T) {
	// Remote book: not in localBooks(), so playing hash should not match
	books := []Audiobook{makeBook("h1", DownloadRemote, nil, nil)}
	ps := newPlayerState(makeLib("h1", books))
	if ps.albumPlaying != nil {
		t.Errorf("albumPlaying = %v, want nil (remote book not local)", ps.albumPlaying)
	}
}

//  readyBooks

func TestLocalBooks_Empty(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadPreparing, nil, nil),
		makeBook("c", DownloadInProgress, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	if got := ps.localBooks(); len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestLocalBooks_Mixed(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
		makeBook("c", DownloadInProgress, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	got := ps.localBooks()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Hash != "b" {
		t.Errorf("books[0].Hash = %q, want %q", got[0].Hash, "b")
	}
}

func TestLocalBooks_AllReady(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
		makeBook("c", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	if got := ps.localBooks(); len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestLocalBooks_ReturnsPointerToOriginal(t *testing.T) {
	books := []Audiobook{makeBook("a", DownloadReady, nil, nil)}
	ps := newPlayerState(makeLib("", books))
	local := ps.localBooks()
	local[0].Title = "mutated"
	if ps.lib.Books[0].Title != "mutated" {
		t.Error("mutation not reflected: readyBooks does not return pointers into lib.Books")
	}
}

//  selectedLocalBook 

func TestSelectedLocalBook_NilWhenEmpty(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	if got := ps.selectedLocalBook(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestSelectedLocalBook_IndexInRange(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
		makeBook("c", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.albumSelected = 1
	got := ps.selectedLocalBook()
	if got == nil || got.Hash != "b" {
		t.Errorf("got %v, want book with hash %q", got, "b")
	}
}

func TestSelectedLocalBook_IndexOutOfRange(t *testing.T) {
	books := []Audiobook{makeBook("a", DownloadReady, nil, nil)}
	ps := newPlayerState(makeLib("", books))
	ps.albumSelected = 99
	if got := ps.selectedLocalBook(); got != nil {
		t.Errorf("got %v, want nil for out-of-range index", got)
	}
}

//  bookProgress 

func TestBookProgress_NilPosition(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 1000), nil)
	if got := ps.bookProgress(&b); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestBookProgress_ZeroTimestamp(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 1000), &Position{Timestamp: 0})
	if got := ps.bookProgress(&b); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestBookProgress_ZeroDuration(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, nil, &Position{ChapterPosition: 100, Timestamp: 1})
	b.Duration = 0
	if got := ps.bookProgress(&b); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestBookProgress_AtStart(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 1000, 1000), &Position{Timestamp: 1})
	if got := ps.bookProgress(&b); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestBookProgress_MidChapter(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	// 3 chapters of 1000 ms each; at chapter 1, position 500 ms
	// elapsed = 1000 + 500 = 1500, total = 3000, pct = 0.5
	b := makeBook("x", DownloadReady, makeChapters(1000, 1000, 1000),
		&Position{ChapterIndex: 1, ChapterPosition: 500, Timestamp: 1})
	got := ps.bookProgress(&b)
	want := 0.5
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBookProgress_LastChapterEnd(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	// at chapter 2, position = chapter[2].Duration → elapsed == total
	b := makeBook("x", DownloadReady, makeChapters(1000, 1000, 1000),
		&Position{ChapterIndex: 2, ChapterPosition: 1000, Timestamp: 1})
	got := ps.bookProgress(&b)
	if got != 1.0 {
		t.Errorf("got %v, want 1.0", got)
	}
}

func TestBookProgress_ClampedAtOne(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000),
		&Position{ChapterIndex: 0, ChapterPosition: 9999, Timestamp: 1})
	// elapsed (9999) > duration (1000)
	got := ps.bookProgress(&b)
	if got != 1.0 {
		t.Errorf("got %v, want 1.0 (clamped)", got)
	}
}

func TestBookProgress_ChapterIndexOutOfBounds(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000),
		&Position{ChapterIndex: 99, ChapterPosition: 0, Timestamp: 1})
	// should not panic; returns some value
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked: %v", r)
		}
	}()
	ps.bookProgress(&b)
}

//  bookElapsed 

func TestBookElapsed_NilPosition(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000), nil)
	if got := ps.bookElapsed(&b); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestBookElapsed_FirstChapter(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 2000),
		&Position{ChapterIndex: 0, ChapterPosition: 500})
	if got := ps.bookElapsed(&b); got != 500 {
		t.Errorf("got %d, want 500", got)
	}
}

func TestBookElapsed_ThirdChapter(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 2000, 3000),
		&Position{ChapterIndex: 2, ChapterPosition: 100})
	// elapsed = 1000 + 2000 + 100 = 3100
	if got := ps.bookElapsed(&b); got != 3100 {
		t.Errorf("got %d, want 3100", got)
	}
}

func TestBookElapsed_ChapterIndexExceedsLen(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b := makeBook("x", DownloadReady, makeChapters(1000, 2000),
		&Position{ChapterIndex: 99, ChapterPosition: 50})
	// should not panic; sums all chapters + position
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked: %v", r)
		}
	}()
	ps.bookElapsed(&b)
}

//  clampOffset 

func TestClampOffset_BelowPadding(t *testing.T) {
	// selected=0, offset=5: must scroll up so selected is visible
	got := clampOffset(5, 0, 10, 2, 20)
	if got < 0 {
		t.Errorf("got %d, want >= 0", got)
	}
	if 0 < got+2 { // selected(0) >= offset + padding(2)
	} else {
		t.Errorf("selected not in padding zone: offset=%d", got)
	}
}

func TestClampOffset_AbovePanelBottom(t *testing.T) {
	// selected=18, offset=0, panelH=10: must scroll down
	got := clampOffset(0, 18, 10, 2, 20)
	// selected must be visible: offset <= selected < offset+panelH
	if !(got <= 18 && 18 < got+10) {
		t.Errorf("selected not visible: offset=%d, selected=18, panelH=10", got)
	}
}

func TestClampOffset_ClampToZero(t *testing.T) {
	got := clampOffset(-5, 0, 10, 2, 20)
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestClampOffset_ClampToMax(t *testing.T) {
	// total=5, panelH=10: all items fit → max offset = 0
	got := clampOffset(3, 4, 10, 2, 5)
	if got != 0 {
		t.Errorf("got %d, want 0 (all items fit)", got)
	}
}

func TestClampOffset_ZeroPanelH(t *testing.T) {
	got := clampOffset(5, 3, 0, 2, 20)
	if got != 0 {
		t.Errorf("got %d, want 0 for zero panelH", got)
	}
}

func TestClampOffset_Stable(t *testing.T) {
	// selected=10, offset=8, panelH=10, padding=2, total=30
	// selected is within [offset+padding, offset+panelH-padding) = [10, 16)
	got := clampOffset(8, 10, 10, 2, 30)
	if got != 8 {
		t.Errorf("got %d, want 8 (no change needed)", got)
	}
}

//  dimension helpers 

func TestAbPanelInnerH_WithInfoOpen(t *testing.T) {
	ps := &PlayerState{windowHeight: 30, albumInfoOpen: true}
	h := 30 - 4 // = 26
	want := h*6/10 + 4
	if got := ps.abPanelInnerH(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestAbPanelInnerH_WithoutInfoOpen(t *testing.T) {
	ps := &PlayerState{windowHeight: 30, albumInfoOpen: false}
	want := 30 - 4
	if got := ps.abPanelInnerH(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestAbPanelInnerH_MinimumOne(t *testing.T) {
	ps := &PlayerState{windowHeight: 0}
	if got := ps.abPanelInnerH(); got != 1 {
		t.Errorf("got %d, want 1 for zero windowHeight", got)
	}
}

func TestLibListH_WithInfoOpen(t *testing.T) {
	ps := &PlayerState{windowHeight: 30, libInfoOpen: true}
	h := 30 - 4
	want := h*6/10 + 4
	if got := ps.libListH(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestInfoDetailH_Computed(t *testing.T) {
	ps := &PlayerState{windowHeight: 30}
	h := 30 - 4
	listH := h*6/10 + 4
	want := h - listH - 1
	if got := ps.infoDetailH(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestAlbumInner_Wide(t *testing.T) {
	ps := &PlayerState{windowWidth: 120} // > WideThreshold(100)
	want := SideWidth - 2
	if got := ps.albumInner(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestAlbumInner_Narrow(t *testing.T) {
	ps := &PlayerState{windowWidth: 80} // <= WideThreshold
	want := 2*80/5 - 2
	if got := ps.albumInner(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestLibInner_Computed(t *testing.T) {
	ps := &PlayerState{windowWidth: 100}
	want := 100*30/100 - 2
	if got := ps.libInner(); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestMaxInfoOff_NilBook(t *testing.T) {
	ps := &PlayerState{}
	if got := ps.maxInfoOff(nil, 30, 10); got != 0 {
		t.Errorf("got %d, want 0 for nil book", got)
	}
}

func TestMaxInfoOff_EmptyDescription(t *testing.T) {
	ps := &PlayerState{}
	b := makeBook("x", DownloadReady, nil, nil)
	if got := ps.maxInfoOff(&b, 30, 10); got != 0 {
		t.Errorf("got %d, want 0 for empty description", got)
	}
}

func TestMaxInfoOff_ShortDescription(t *testing.T) {
	ps := &PlayerState{}
	b := makeBook("x", DownloadReady, nil, nil)
	b.Description = "Short desc"
	// wrapText("Short desc", 30) = 1 line; visible = detailH-5; if 1 <= visible → maxOff=0
	detailH := 10 // visible = 10-5 = 5; 1 line < 5 visible → 0
	if got := ps.maxInfoOff(&b, 30, detailH); got != 0 {
		t.Errorf("got %d, want 0 for short description", got)
	}
}

func TestMaxInfoOff_LongDescription(t *testing.T) {
	ps := &PlayerState{}
	b := makeBook("x", DownloadReady, nil, nil)
	// 30 words at width 10 → many lines
	desc := ""
	for i := 0; i < 30; i++ {
		desc += "word "
	}
	b.Description = desc
	wrapped := wrapText(b.Description, 10)
	visible := 8 - 5 // detailH=8, fixedLines=5
	want := len(wrapped) - visible
	if want < 0 {
		want = 0
	}
	if got := ps.maxInfoOff(&b, 10, 8); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

//  fetchBooksCmd

func TestFetchBooksCmd_Success(t *testing.T) {
	b1 := makeBook("a", DownloadRemote, nil, nil)
	b2 := makeBook("b", DownloadReady, nil, nil)
	mock := &MockApiClient{
		GetAudiobooksFn: func() ([]Audiobook, error) {
			return []Audiobook{b1, b2}, nil
		},
	}
	cmd := fetchBooksCmd(mock)
	if cmd == nil {
		t.Fatal("fetchBooksCmd returned nil")
	}
	msg := cmd()
	result, ok := msg.(booksResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want booksResultMsg", msg)
	}
	if result.err != nil {
		t.Errorf("err = %v, want nil", result.err)
	}
	if len(result.books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(result.books))
	}
}

func TestFetchBooksCmd_Error(t *testing.T) {
	mock := &MockApiClient{
		GetAudiobooksFn: func() ([]Audiobook, error) {
			return nil, errors.New("network error")
		},
	}
	cmd := fetchBooksCmd(mock)
	if cmd == nil {
		t.Fatal("fetchBooksCmd returned nil")
	}
	msg := cmd()
	result, ok := msg.(booksResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want booksResultMsg", msg)
	}
	if result.err == nil {
		t.Error("err should be non-nil on API error")
	}
	if result.books != nil {
		t.Error("books should be nil on error")
	}
}

//  Update — booksResultMsg

func TestUpdate_BooksResult_Success(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	b1 := makeBook("a", DownloadRemote, nil, nil)
	b2 := makeBook("b", DownloadReady, nil, nil)
	result, _ := ps.Update(booksResultMsg{books: []Audiobook{b1, b2}})
	ps2 := result.(*PlayerState)
	if len(ps2.lib.Books) != 2 {
		t.Errorf("lib.Books len = %d, want 2", len(ps2.lib.Books))
	}
	if ps2.statusErr {
		t.Error("statusErr should be false on success")
	}
}

func TestUpdate_BooksResult_Error(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	result, _ := ps.Update(booksResultMsg{err: errors.New("timeout")})
	ps2 := result.(*PlayerState)
	if ps2.statusMsg == "" {
		t.Error("statusMsg should be set on error")
	}
	if !ps2.statusErr {
		t.Error("statusErr should be true on error")
	}
}

//  loadLocalBooksCmd

func TestLoadLocalBooksCmd_ReturnsLocalBooksMsg(t *testing.T) {
	s := newTestStore(t)
	cmd := loadLocalBooksCmd(s)
	if cmd == nil {
		t.Fatal("loadLocalBooksCmd returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(localBooksMsg); !ok {
		t.Fatalf("msg type = %T, want localBooksMsg", msg)
	}
}

//  Update — localBooksMsg

func TestUpdate_LocalBooksMsg_SetsReadyState(t *testing.T) {
	serverBook := makeBook("a", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{serverBook}))

	localBook := makeBook("a", DownloadReady, makeChapters(1000, 2000), nil)
	result, _ := ps.Update(localBooksMsg{books: []Audiobook{localBook}})
	ps2 := result.(*PlayerState)

	if len(ps2.lib.Books) == 0 {
		t.Fatal("lib.Books should not be empty")
	}
	if ps2.lib.Books[0].State != DownloadReady {
		t.Errorf("book state = %v, want DownloadReady after local scan", ps2.lib.Books[0].State)
	}
	if len(ps2.lib.Books[0].Chapters) != 2 {
		t.Errorf("chapters = %d, want 2 (merged from local scan)", len(ps2.lib.Books[0].Chapters))
	}
}
