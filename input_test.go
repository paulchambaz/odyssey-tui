package main

import (
	"context"
	"strings"
	"testing"
)

//  Global handleKey 

func TestHandleKey_CtrlC_Quits(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	_, cmd := pressKey(ps, "ctrl+c")
	if !isQuitCmd(cmd) {
		t.Error("ctrl+c should produce quit cmd")
	}
}

func TestHandleKey_QuestionMark_EntersHelp(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeMain
	ps2, _ := pressKey(ps, "?")
	if ps2.mode != ModeHelp {
		t.Errorf("mode = %v, want ModeHelp", ps2.mode)
	}
	if ps2.returnMode != ModeMain {
		t.Errorf("returnMode = %v, want ModeMain", ps2.returnMode)
	}
}

func TestHandleKey_QuestionMark_WhenAlreadyHelp_ExitsHelp(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeHelp
	ps.returnMode = ModeMain
	ps2, _ := pressKey(ps, "?")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain (? exits help)", ps2.mode)
	}
}

func TestHandleKey_DispatchesToMode(t *testing.T) {
	// q in ModeMain → quit
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeMain
	_, cmd := pressKey(ps, "q")
	if !isQuitCmd(cmd) {
		t.Error("q in ModeMain should quit")
	}

	// q in ModePlayer → back to main (not quit)
	ps2 := newPlayerState(makeLib("", nil))
	ps2.mode = ModePlayer
	ps3, cmd2 := pressKey(ps2, "q")
	if isQuitCmd(cmd2) {
		t.Error("q in ModePlayer should not quit")
	}
	if ps3.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps3.mode)
	}

	// q in ModeHelp → returnMode
	ps4 := newPlayerState(makeLib("", nil))
	ps4.mode = ModeHelp
	ps4.returnMode = ModePlayer
	ps5, _ := pressKey(ps4, "q")
	if ps5.mode != ModePlayer {
		t.Errorf("mode = %v, want ModePlayer", ps5.mode)
	}
}

//  handleMain — quit 

func TestHandleMain_Q_Quits(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	_, cmd := pressKey(ps, "q")
	if !isQuitCmd(cmd) {
		t.Error("q should produce quit cmd")
	}
}

//  handleMain — j scroll

func TestHandleMain_J_ScrollsAlbums(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
		makeBook("c", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1", ps2.albumSelected)
	}
	if ps2.albumInfoOff != 0 {
		t.Errorf("albumInfoOff = %d, want 0 (reset)", ps2.albumInfoOff)
	}
}

func TestHandleMain_J_ScrollsAlbums_WithPosition(t *testing.T) {
	pos := &Position{ChapterIndex: 4, Timestamp: 1}
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), nil),
		makeBook("b", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), pos),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.trackSelected != 4 {
		t.Errorf("trackSelected = %d, want 4 (restored from position)", ps2.trackSelected)
	}
}

func TestHandleMain_J_ScrollsAlbums_AtEnd_NoChange(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 1 // last
	ps2, _ := pressKey(ps, "j")
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (no change at end)", ps2.albumSelected)
	}
}

func TestHandleMain_J_ScrollsChapters(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = false
	ps.libActive = false
	ps.trackSelected = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.trackSelected != 1 {
		t.Errorf("trackSelected = %d, want 1", ps2.trackSelected)
	}
}

func TestHandleMain_J_ChaptersAtEnd(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000), nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = false
	ps.libActive = false
	ps.trackSelected = 1 // last
	ps2, _ := pressKey(ps, "j")
	if ps2.trackSelected != 1 {
		t.Errorf("trackSelected = %d, want 1 (no change at end)", ps2.trackSelected)
	}
}

//  handleMain — k scroll

func TestHandleMain_K_ScrollsAlbums(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadReady, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 1
	ps2, _ := pressKey(ps, "k")
	if ps2.albumSelected != 0 {
		t.Errorf("albumSelected = %d, want 0", ps2.albumSelected)
	}
}

func TestHandleMain_K_AlbumsAtStart_NoChange(t *testing.T) {
	books := []Audiobook{makeBook("a", DownloadReady, nil, nil)}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps2, _ := pressKey(ps, "k")
	if ps2.albumSelected != 0 {
		t.Errorf("albumSelected = %d, want 0 (no change at start)", ps2.albumSelected)
	}
}

func TestHandleMain_K_ScrollsChapters(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000, 1000), nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = false
	ps.libActive = false
	ps.trackSelected = 2
	ps2, _ := pressKey(ps, "k")
	if ps2.trackSelected != 1 {
		t.Errorf("trackSelected = %d, want 1", ps2.trackSelected)
	}
}

func TestHandleMain_K_ChaptersAtStart_NoChange(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000), nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = false
	ps.libActive = false
	ps.trackSelected = 0
	ps2, _ := pressKey(ps, "k")
	if ps2.trackSelected != 0 {
		t.Errorf("trackSelected = %d, want 0 (no change at start)", ps2.trackSelected)
	}
}

//  handlePlayer 

func makePlayerState(book Audiobook) *PlayerState {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModePlayer
	ps.playerBook = &book
	return ps
}

func TestHandlePlayer_Q_ReturnToMain(t *testing.T) {
	ps := makePlayerState(makeBook("x", DownloadReady, makeChapters(1000), nil))
	ps2, _ := pressKey(ps, "q")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps2.mode)
	}
}

func TestHandlePlayer_Esc_ReturnToMain(t *testing.T) {
	ps := makePlayerState(makeBook("x", DownloadReady, makeChapters(1000), nil))
	ps2, _ := pressKey(ps, "esc")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps2.mode)
	}
}

func TestHandlePlayer_S_CyclesSpeed(t *testing.T) {
	ps := makePlayerState(makeBook("x", DownloadReady, nil, nil))
	ps.playerSpeed = 1.00
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed != 1.50 {
		t.Errorf("playerSpeed = %v, want 1.50", ps2.playerSpeed)
	}
}

func TestHandlePlayer_S_CyclesSpeedWrapsAround(t *testing.T) {
	ps := makePlayerState(makeBook("x", DownloadReady, nil, nil))
	ps.playerSpeed = 3.00 // last in playerSpeeds
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed != 0.50 {
		t.Errorf("playerSpeed = %v, want 0.50 (wrapped)", ps2.playerSpeed)
	}
}

func TestHandlePlayer_S_UnknownSpeed_NoChange(t *testing.T) {
	ps := makePlayerState(makeBook("x", DownloadReady, nil, nil))
	ps.playerSpeed = 99.0
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed != 99.0 {
		t.Errorf("playerSpeed = %v, want 99.0 (no match in list)", ps2.playerSpeed)
	}
}

func TestHandlePlayer_J_ScrollsChapters(t *testing.T) {
	book := makeBook("x", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), nil)
	ps := makePlayerState(book)
	ps.playerChSel = 2
	ps2, _ := pressKey(ps, "j")
	if ps2.playerChSel != 3 {
		t.Errorf("playerChSel = %d, want 3", ps2.playerChSel)
	}
}

func TestHandlePlayer_J_AtEnd_NoChange(t *testing.T) {
	book := makeBook("x", DownloadReady, makeChapters(1000, 1000), nil)
	ps := makePlayerState(book)
	ps.playerChSel = 1 // last
	ps2, _ := pressKey(ps, "j")
	if ps2.playerChSel != 1 {
		t.Errorf("playerChSel = %d, want 1 (no change)", ps2.playerChSel)
	}
}

func TestHandlePlayer_J_NilBook_NoChange(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModePlayer
	ps.playerBook = nil
	ps.playerChSel = 2
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked with nil playerBook: %v", r)
		}
	}()
	ps2, _ := pressKey(ps, "j")
	if ps2.playerChSel != 2 {
		t.Errorf("playerChSel = %d, want 2 (no change with nil book)", ps2.playerChSel)
	}
}

func TestHandlePlayer_K_ScrollsChapters(t *testing.T) {
	book := makeBook("x", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := makePlayerState(book)
	ps.playerChSel = 2
	ps2, _ := pressKey(ps, "k")
	if ps2.playerChSel != 1 {
		t.Errorf("playerChSel = %d, want 1", ps2.playerChSel)
	}
}

func TestHandlePlayer_K_AtStart_NoChange(t *testing.T) {
	book := makeBook("x", DownloadReady, makeChapters(1000), nil)
	ps := makePlayerState(book)
	ps.playerChSel = 0
	ps2, _ := pressKey(ps, "k")
	if ps2.playerChSel != 0 {
		t.Errorf("playerChSel = %d, want 0 (no change)", ps2.playerChSel)
	}
}

//  handleHelp 

func TestHandleHelp_Q_ReturnsToReturnMode(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeHelp
	ps.returnMode = ModeMain
	ps2, _ := pressKey(ps, "q")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps2.mode)
	}
}

func TestHandleHelp_Esc_ReturnsToReturnMode(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeHelp
	ps.returnMode = ModePlayer
	ps2, _ := pressKey(ps, "esc")
	if ps2.mode != ModePlayer {
		t.Errorf("mode = %v, want ModePlayer", ps2.mode)
	}
}

func TestHandleHelp_QuestionMark_ReturnsToReturnMode(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeHelp
	ps.returnMode = ModeMain
	// Note: handleKey intercepts "?" before reaching handleHelp when mode==ModeHelp
	// The guard `if ps.mode != ModeHelp` prevents entering help again.
	// So pressing "?" in ModeHelp does nothing (mode stays ModeHelp).
	ps2, _ := pressKey(ps, "?")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain (? exits help)", ps2.mode)
	}
}

func TestHandleHelp_OtherKey_NoChange(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeHelp
	ps2, _ := pressKey(ps, "j")
	if ps2.mode != ModeHelp {
		t.Errorf("mode = %v, want ModeHelp", ps2.mode)
	}
}

//  handleMain — d download/cancel

func TestHandleMain_D_Remote_StartsDownload(t *testing.T) {
	book := makeBook("a", DownloadRemote, nil, nil)
	mock := &MockApiClient{
		GetAudiobooksFn: func() ([]Audiobook, error) { return nil, nil },
	}
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.api = mock
	ps.store = newTestStore(t)
	ps.libActive = true
	ps.libSel = 0

	ps2, cmd := pressKey(ps, "d")
	if cmd == nil {
		t.Error("d on DownloadRemote should return a non-nil download cmd")
	}
	if ps2.lib.Books[0].State != DownloadInProgress {
		t.Errorf("state = %v, want DownloadInProgress", ps2.lib.Books[0].State)
	}
}

func TestHandleMain_D_InProgress_Cancels(t *testing.T) {
	book := makeBook("a", DownloadInProgress, nil, nil)
	cancelled := false
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.cancelDownloads = map[string]context.CancelFunc{
		"a": func() { cancelled = true },
	}
	ps.libActive = true
	ps.libSel = 0

	pressKey(ps, "d")
	if !cancelled {
		t.Error("d on DownloadInProgress should call the cancel func")
	}
}

//  handleMain — p / enter play

func TestHandleMain_P_Ready(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000, 2000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0

	ps2, _ := pressKey(ps, "p")
	if ps2.mode != ModePlayer {
		t.Errorf("mode = %v, want ModePlayer", ps2.mode)
	}
}

func TestHandleMain_Enter_Ready(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000, 2000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0

	ps2, _ := pressKey(ps, "enter")
	if ps2.mode != ModePlayer {
		t.Errorf("mode = %v, want ModePlayer", ps2.mode)
	}
}

func TestHandleMain_P_NotReady_Remote(t *testing.T) {
	book := makeBook("a", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0

	ps2, _ := pressKey(ps, "p")
	if ps2.statusMsg == "" {
		t.Error("p on remote book should set statusMsg")
	}
	if !ps2.statusErr {
		t.Error("p on remote book should set statusErr")
	}
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain (no play on non-ready)", ps2.mode)
	}
}

func TestHandleMain_P_NotReady_Preparing(t *testing.T) {
	book := makeBook("a", DownloadPreparing, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0

	ps2, _ := pressKey(ps, "p")
	if ps2.statusMsg == "" {
		t.Error("p on preparing book should set statusMsg")
	}
	if !ps2.statusErr {
		t.Error("p on preparing book should set statusErr")
	}
}

//  handleMain — r refresh

func TestHandleMain_R_FetchesBooks(t *testing.T) {
	mock := &MockApiClient{
		GetAudiobooksFn: func() ([]Audiobook, error) { return nil, nil },
	}
	ps := newPlayerState(makeLib("", nil))
	ps.api = mock

	_, cmd := pressKey(ps, "r")
	if cmd == nil {
		t.Error("r should return a non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(booksResultMsg); !ok {
		t.Errorf("cmd() returned %T, want booksResultMsg", msg)
	}
}

//  handleMain — h/l panel navigation

func TestHandleMain_H_ChaptersToAlbums(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = false
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "h")
	if !ps2.onAlbum {
		t.Error("h from chapters should set onAlbum")
	}
}

func TestHandleMain_H_AlbumsToLibrary(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.showLibrary = true
	ps.libActive = false
	ps.onAlbum = true
	ps2, _ := pressKey(ps, "h")
	if !ps2.libActive {
		t.Error("h from albums with library visible should set libActive")
	}
	if ps2.onAlbum {
		t.Error("h from albums to library should clear onAlbum")
	}
}

func TestHandleMain_H_NoOpWhenLibActive(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "h")
	if !ps2.libActive {
		t.Error("h when already at leftmost panel should not change libActive")
	}
}

func TestHandleMain_L_LibraryToAlbums(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "l")
	if ps2.libActive {
		t.Error("l from library should clear libActive")
	}
	if !ps2.onAlbum {
		t.Error("l from library should set onAlbum")
	}
}

func TestHandleMain_L_AlbumsToChapters(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = false
	ps.onAlbum = true
	ps2, _ := pressKey(ps, "l")
	if ps2.onAlbum {
		t.Error("l from albums should clear onAlbum")
	}
}

func TestHandleMain_L_NoOpWhenChapters(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = false
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "l")
	if ps2.libActive || ps2.onAlbum {
		t.Error("l from chapters (rightmost panel) should not change focus")
	}
}

//  handleMain — j/k library scroll

func TestHandleMain_J_ScrollsLib(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
		makeBook("c", DownloadRemote, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.libSel != 1 {
		t.Errorf("libSel = %d, want 1", ps2.libSel)
	}
	if ps2.libInfoOff != 0 {
		t.Errorf("libInfoOff = %d, want 0 (reset)", ps2.libInfoOff)
	}
}

func TestHandleMain_J_LibAtEnd_NoChange(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 1
	ps2, _ := pressKey(ps, "j")
	if ps2.libSel != 1 {
		t.Errorf("libSel = %d, want 1 (no change at end)", ps2.libSel)
	}
}

func TestHandleMain_K_ScrollsLib(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 1
	ps2, _ := pressKey(ps, "k")
	if ps2.libSel != 0 {
		t.Errorf("libSel = %d, want 0", ps2.libSel)
	}
	if ps2.libInfoOff != 0 {
		t.Errorf("libInfoOff = %d, want 0 (reset)", ps2.libInfoOff)
	}
}

func TestHandleMain_K_LibAtStart_NoChange(t *testing.T) {
	books := []Audiobook{makeBook("a", DownloadRemote, nil, nil)}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 0
	ps2, _ := pressKey(ps, "k")
	if ps2.libSel != 0 {
		t.Errorf("libSel = %d, want 0 (no change at start)", ps2.libSel)
	}
}

//  handleMain — j/k with info focused

func TestHandleMain_J_LibInfoFocused_Scrolls(t *testing.T) {
	b := makeBook("a", DownloadReady, nil, nil)
	b.Description = strings.Repeat("word ", 200)
	lib := makeLib("", []Audiobook{b})
	ps := newPlayerState(lib)
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = true
	ps.windowWidth = 80
	ps.windowHeight = 40
	ps.libInfoOff = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.libInfoOff != 1 {
		t.Errorf("libInfoOff = %d, want 1", ps2.libInfoOff)
	}
}

func TestHandleMain_J_LibInfoFocused_ClampedAtMax(t *testing.T) {
	b := makeBook("a", DownloadReady, nil, nil)
	lib := makeLib("", []Audiobook{b})
	ps := newPlayerState(lib)
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = true
	ps.windowWidth = 80
	ps.windowHeight = 24
	ps.libInfoOff = ps.maxInfoOff(&b, ps.libInner(), ps.infoDetailH())
	before := ps.libInfoOff
	ps2, _ := pressKey(ps, "j")
	if ps2.libInfoOff != before {
		t.Errorf("libInfoOff = %d, want %d (clamped at max)", ps2.libInfoOff, before)
	}
}

func TestHandleMain_K_LibInfoFocused_Scrolls(t *testing.T) {
	b := makeBook("a", DownloadReady, nil, nil)
	b.Description = strings.Repeat("word ", 200)
	lib := makeLib("", []Audiobook{b})
	ps := newPlayerState(lib)
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = true
	ps.windowWidth = 80
	ps.windowHeight = 40
	ps.libInfoOff = 3
	ps2, _ := pressKey(ps, "k")
	if ps2.libInfoOff != 2 {
		t.Errorf("libInfoOff = %d, want 2", ps2.libInfoOff)
	}
}

func TestHandleMain_K_LibInfoFocused_ClampedAtZero(t *testing.T) {
	b := makeBook("a", DownloadReady, nil, nil)
	lib := makeLib("", []Audiobook{b})
	ps := newPlayerState(lib)
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = true
	ps.libInfoOff = 0
	ps2, _ := pressKey(ps, "k")
	if ps2.libInfoOff != 0 {
		t.Errorf("libInfoOff = %d, want 0 (clamped at zero)", ps2.libInfoOff)
	}
}

func TestHandleMain_J_AlbumInfoFocused_Scrolls(t *testing.T) {
	b := makeBook("a", DownloadReady, nil, nil)
	b.Description = strings.Repeat("word ", 200)
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.onAlbum = true
	ps.libActive = false
	ps.albumInfoOpen = true
	ps.albumInfoFocused = true
	ps.windowWidth = 80
	ps.windowHeight = 40
	ps.albumInfoOff = 0
	ps2, _ := pressKey(ps, "j")
	if ps2.albumInfoOff != 1 {
		t.Errorf("albumInfoOff = %d, want 1", ps2.albumInfoOff)
	}
}

//  handleMain — i/I info toggle

func TestHandleMain_ShiftI_TogglesLibInfoFocused(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = false
	ps2, _ := pressKey(ps, "I")
	if !ps2.libInfoFocused {
		t.Error("I should toggle libInfoFocused to true")
	}
}

func TestHandleMain_ShiftI_TogglesAlbumInfoFocused(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.onAlbum = true
	ps.libActive = false
	ps.albumInfoOpen = true
	ps.albumInfoFocused = false
	ps2, _ := pressKey(ps, "I")
	if !ps2.albumInfoFocused {
		t.Error("I should toggle albumInfoFocused to true")
	}
}

func TestHandleMain_LowI_TogglesLibInfoOpen(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.libInfoOpen = true
	ps2, _ := pressKey(ps, "i")
	if ps2.libInfoOpen {
		t.Error("i should close libInfoOpen")
	}
}

func TestHandleMain_LowI_ClosingLibClearsFocused(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = true
	ps2, _ := pressKey(ps, "i")
	if ps2.libInfoFocused {
		t.Error("closing info should clear libInfoFocused")
	}
}

func TestHandleMain_LowI_TogglesAlbumInfoOpen(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = false
	ps.albumInfoOpen = false
	ps2, _ := pressKey(ps, "i")
	if !ps2.albumInfoOpen {
		t.Error("i should open albumInfoOpen")
	}
}

func TestHandleMain_LowI_ClosingAlbumClearsFocused(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = false
	ps.albumInfoOpen = true
	ps.albumInfoFocused = true
	ps2, _ := pressKey(ps, "i")
	if ps2.albumInfoFocused {
		t.Error("closing album info should clear albumInfoFocused")
	}
}
