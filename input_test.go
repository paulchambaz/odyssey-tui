package main

import (
	"context"
	"os"
	"path/filepath"
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

	// q in ModeHelp → returnMode
	ps4 := newPlayerState(makeLib("", nil))
	ps4.mode = ModeHelp
	ps4.returnMode = ModeMain
	ps5, _ := pressKey(ps4, "q")
	if ps5.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps5.mode)
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
	// posA has higher timestamp so book "a" sorts first; posB lower timestamp sorts second.
	// Both are in-progress (rank 0) so only timestamp differentiates order.
	posA := &Position{ChapterIndex: 2, Timestamp: 100}
	posB := &Position{ChapterIndex: 4, Timestamp: 1}
	books := []Audiobook{
		makeBook("a", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), posA),
		makeBook("b", DownloadReady, makeChapters(1000, 1000, 1000, 1000, 1000), posB),
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
	ps.returnMode = ModeSearch
	ps2, _ := pressKey(ps, "esc")
	if ps2.mode != ModeSearch {
		t.Errorf("mode = %v, want ModeSearch", ps2.mode)
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

//  handleMain — enter download/cancel/play in library

func TestHandleMain_Enter_Remote_StartsDownload(t *testing.T) {
	book := makeBook("a", DownloadRemote, nil, nil)
	mock := &MockApiClient{
		GetAudiobooksFn: func() ([]Audiobook, error) { return nil, nil },
	}
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.api = mock
	ps.store = newTestStore(t)
	ps.libActive = true
	ps.libSel = 0

	ps2, cmd := pressKey(ps, "enter")
	if cmd == nil {
		t.Error("enter on DownloadRemote should return a non-nil download cmd")
	}
	if ps2.lib.Books[0].State != DownloadInProgress {
		t.Errorf("state = %v, want DownloadInProgress", ps2.lib.Books[0].State)
	}
}

func TestHandleMain_Enter_InProgress_Cancels(t *testing.T) {
	book := makeBook("a", DownloadInProgress, nil, nil)
	cancelled := false
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.cancelDownloads = map[string]context.CancelFunc{
		"a": func() { cancelled = true },
	}
	ps.libActive = true
	ps.libSel = 0

	pressKey(ps, "enter")
	if !cancelled {
		t.Error("enter on DownloadInProgress should call the cancel func")
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

//  handleMain — l library toggle, H/L panel navigation

func TestHandleMain_D_ToggleShowsLibrary(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.showLibrary = false
	ps2, _ := pressKey(ps, "d")
	if !ps2.showLibrary {
		t.Error("d should set showLibrary")
	}
}

func TestHandleMain_D_ToggleHidesLibrary(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.showLibrary = true
	ps.libActive = false
	ps2, _ := pressKey(ps, "d")
	if ps2.showLibrary {
		t.Error("d should clear showLibrary")
	}
}

func TestHandleMain_D_ToggleHidesLibraryAndResetsFocus(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.showLibrary = true
	ps.libActive = true
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "d")
	if ps2.showLibrary {
		t.Error("d hiding library should clear showLibrary")
	}
	if ps2.libActive {
		t.Error("d hiding library should clear libActive")
	}
	if !ps2.onAlbum {
		t.Error("d hiding library should set onAlbum")
	}
}

func TestHandleMain_D_OpenLibraryFocusesLibrary(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.showLibrary = false
	ps.libActive = false
	ps.onAlbum = true
	ps2, _ := pressKey(ps, "d")
	if !ps2.showLibrary {
		t.Error("d should set showLibrary")
	}
	if !ps2.libActive {
		t.Error("d opening library should set libActive")
	}
	if ps2.onAlbum {
		t.Error("d opening library should clear onAlbum")
	}
}

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

func TestHandleMain_ShiftI_LibActive_FiresFetchDetail(t *testing.T) {
	book := makeBook("abc", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = false
	ps.api = &MockApiClient{
		GetAudiobookFn: func(hash string) (Audiobook, error) {
			return Audiobook{Hash: hash, Description: "fetched desc"}, nil
		},
	}
	_, cmd := pressKey(ps, "I")
	if cmd == nil {
		t.Fatal("I in lib with api should return a fetch cmd")
	}
	msg := cmd()
	detail, ok := msg.(audiobookDetailMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want audiobookDetailMsg", msg)
	}
	if detail.err != nil {
		t.Fatalf("unexpected error: %v", detail.err)
	}
	if detail.book.Description != "fetched desc" {
		t.Errorf("description = %q, want %q", detail.book.Description, "fetched desc")
	}
}

func TestHandleMain_ShiftI_LibActive_NoApiNoCmd(t *testing.T) {
	book := makeBook("abc", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libInfoOpen = true
	ps.libInfoFocused = false
	_, cmd := pressKey(ps, "I")
	if cmd != nil {
		t.Error("I without api should return nil cmd")
	}
}

func TestUpdate_AudiobookDetailMsg_UpdatesDescription(t *testing.T) {
	book := makeBook("abc", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	msg := audiobookDetailMsg{book: Audiobook{Hash: "abc", Description: "new desc", Genres: []string{"sci-fi"}}}
	model, _ := ps.Update(msg)
	ps2 := model.(*PlayerState)
	var found bool
	for _, b := range ps2.lib.Books {
		if b.Hash == "abc" {
			found = true
			if b.Description != "new desc" {
				t.Errorf("description = %q, want %q", b.Description, "new desc")
			}
			if len(b.Genres) == 0 || b.Genres[0] != "sci-fi" {
				t.Errorf("genres = %v, want [sci-fi]", b.Genres)
			}
		}
	}
	if !found {
		t.Error("book not found in lib after update")
	}
}

func TestUpdate_AudiobookDetailMsg_ErrorIsNoop(t *testing.T) {
	book := makeBook("abc", DownloadRemote, nil, nil)
	book.Description = "original"
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	msg := audiobookDetailMsg{err: context.Canceled}
	model, _ := ps.Update(msg)
	ps2 := model.(*PlayerState)
	if ps2.lib.Books[0].Description != "original" {
		t.Error("error msg should leave description unchanged")
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

//  handleMain — x delete

func TestHandleMain_X_ReturnsCmdOnReadyBook(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = newTestStore(t)
	ps.onAlbum = true
	ps.albumSelected = 0
	_, cmd := pressKey(ps, "x")
	if cmd == nil {
		t.Error("x on ready book should return non-nil cmd")
	}
}

func TestHandleMain_X_ChangesStateImmediately(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = newTestStore(t)
	ps.onAlbum = true
	ps.albumSelected = 0
	ps2, _ := pressKey(ps, "x")
	if len(ps2.localBooks()) != 0 {
		t.Errorf("localBooks after x should be empty, got %d", len(ps2.localBooks()))
	}
}

func TestHandleMain_X_CmdDeletesDirectory(t *testing.T) {
	s := newTestStore(t)
	bookDir := filepath.Join(s.dataDir, "library", "a")
	os.MkdirAll(bookDir, 0755)
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = s
	ps.onAlbum = true
	ps.albumSelected = 0
	_, cmd := pressKey(ps, "x")
	if cmd == nil {
		t.Fatal("x should return non-nil cmd")
	}
	cmd()
	if _, err := os.Stat(bookDir); !os.IsNotExist(err) {
		t.Error("x cmd should delete the book directory")
	}
}

func TestHandleMain_X_CmdReturnsLocalBooksMsg(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = newTestStore(t)
	ps.onAlbum = true
	ps.albumSelected = 0
	_, cmd := pressKey(ps, "x")
	if cmd == nil {
		t.Fatal("x should return non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(localBooksMsg); !ok {
		t.Errorf("cmd() returned %T, want localBooksMsg", msg)
	}
}

func TestHandleMain_X_NoopWhenLibActive(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = newTestStore(t)
	ps.libActive = true
	ps.onAlbum = false
	ps2, cmd := pressKey(ps, "x")
	if cmd != nil {
		t.Error("x in library panel should not return cmd")
	}
	if len(ps2.localBooks()) != 1 {
		t.Error("x in library panel should not change local books")
	}
}

func TestHandleMain_X_NoopWhenNilStore(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = nil
	ps.onAlbum = true
	ps.albumSelected = 0
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("x with nil store should not panic: %v", r)
		}
	}()
	pressKey(ps, "x")
}

//  handleMain — / search entry

func TestHandleMain_Slash_EntersModeSearch(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.onAlbum = true
	ps.libActive = false
	ps2, _ := pressKey(ps, "/")
	if ps2.mode != ModeSearch {
		t.Errorf("mode = %v, want ModeSearch", ps2.mode)
	}
}

func TestHandleMain_Slash_SavesAlbumCursor(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps.albumOffset = 0
	ps2, _ := pressKey(ps, "/")
	if ps2.searchSavedAlbum != 0 {
		t.Errorf("searchSavedAlbum = %d, want 0", ps2.searchSavedAlbum)
	}
}

func TestHandleMain_Slash_ClearsQuery(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.onAlbum = true
	ps.searchQuery = "stale"
	ps2, _ := pressKey(ps, "/")
	if ps2.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty after /", ps2.searchQuery)
	}
}

func TestHandleMain_Slash_NoopWhenLibActive(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.libActive = true
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "/")
	if ps2.mode == ModeSearch {
		t.Error("/ in library panel should not enter ModeSearch")
	}
}

//  handleSearch

func TestHandleSearch_TypingRune_AppendsToQuery(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps2, _ := pressKey(ps, "d")
	if ps2.searchQuery != "d" {
		t.Errorf("searchQuery = %q, want %q", ps2.searchQuery, "d")
	}
}

func TestHandleSearch_TypingMultipleRunes(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps2, _ := pressKey(ps, "d")
	ps3, _ := pressKey(ps2, "u")
	if ps3.searchQuery != "du" {
		t.Errorf("searchQuery = %q, want %q", ps3.searchQuery, "du")
	}
}

func TestHandleSearch_TypingRune_RunsSearch(t *testing.T) {
	b := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b.Title = "Dune"
	ps := newPlayerState(makeLib("", []Audiobook{b}))
	ps.mode = ModeSearch
	ps2, _ := pressKey(ps, "d")
	if len(ps2.searchMatches) == 0 {
		t.Error("typing should populate searchMatches via runSearch")
	}
}

func TestHandleSearch_Backspace_RemovesLastChar(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = "ab"
	ps2, _ := pressKey(ps, "backspace")
	if ps2.searchQuery != "a" {
		t.Errorf("searchQuery = %q, want %q after backspace", ps2.searchQuery, "a")
	}
}

func TestHandleSearch_Backspace_Empty_RestoresCursor(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = ""
	ps.searchSavedAlbum = 2
	ps.searchSavedOffset = 1
	ps.albumSelected = 5
	ps.albumOffset = 3
	ps2, _ := pressKey(ps, "backspace")
	if ps2.albumSelected != 2 {
		t.Errorf("albumSelected = %d, want 2 (restored)", ps2.albumSelected)
	}
	if ps2.albumOffset != 1 {
		t.Errorf("albumOffset = %d, want 1 (restored)", ps2.albumOffset)
	}
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain after empty backspace", ps2.mode)
	}
}

func TestHandleSearch_Enter_NonEmpty_TransitionsToSearching(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.mode = ModeSearch
	ps.searchQuery = "foo"
	ps.searchMatches = []int{0}
	ps2, _ := pressKey(ps, "enter")
	if ps2.mode != ModeSearching {
		t.Errorf("mode = %v, want ModeSearching", ps2.mode)
	}
}

func TestHandleSearch_Enter_Empty_CancelsSearch(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = ""
	ps.searchSavedAlbum = 1
	ps.albumSelected = 3
	ps2, _ := pressKey(ps, "enter")
	if ps2.mode == ModeSearching {
		t.Error("enter with empty query should not enter ModeSearching")
	}
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (restored on cancel)", ps2.albumSelected)
	}
}

func TestHandleSearch_Escape_CancelsAndRestores(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearch
	ps.searchQuery = "foo"
	ps.searchSavedAlbum = 3
	ps.albumSelected = 7
	ps2, _ := pressKey(ps, "esc")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain after esc", ps2.mode)
	}
	if ps2.albumSelected != 3 {
		t.Errorf("albumSelected = %d, want 3 (restored)", ps2.albumSelected)
	}
}

//  handleSearching

func TestHandleSearching_N_AdvancesMatchIdx(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	b3 := makeBook("c", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2, b3}))
	ps.mode = ModeSearching
	ps.searchMatches = []int{0, 1, 2}
	ps.searchMatchIdx = 0
	ps2, _ := pressKey(ps, "n")
	if ps2.searchMatchIdx != 1 {
		t.Errorf("searchMatchIdx = %d, want 1", ps2.searchMatchIdx)
	}
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (searchMatches[1])", ps2.albumSelected)
	}
}

func TestHandleSearching_N_Wraps(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.mode = ModeSearching
	ps.searchMatches = []int{0, 1}
	ps.searchMatchIdx = 1
	ps2, _ := pressKey(ps, "n")
	if ps2.searchMatchIdx != 0 {
		t.Errorf("searchMatchIdx = %d, want 0 (wrapped)", ps2.searchMatchIdx)
	}
}

func TestHandleSearching_P_DecrementsIdx(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	b3 := makeBook("c", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2, b3}))
	ps.mode = ModeSearching
	ps.searchMatches = []int{0, 1, 2}
	ps.searchMatchIdx = 2
	ps2, _ := pressKey(ps, "p")
	if ps2.searchMatchIdx != 1 {
		t.Errorf("searchMatchIdx = %d, want 1", ps2.searchMatchIdx)
	}
}

func TestHandleSearching_P_WrapsBackward(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.mode = ModeSearching
	ps.searchMatches = []int{0, 1}
	ps.searchMatchIdx = 0
	ps2, _ := pressKey(ps, "p")
	if ps2.searchMatchIdx != 1 {
		t.Errorf("searchMatchIdx = %d, want 1 (wrapped backward)", ps2.searchMatchIdx)
	}
}

func TestHandleSearching_Enter_ExitsKeepPosition(t *testing.T) {
	b1 := makeBook("a", DownloadReady, makeChapters(1000), nil)
	b2 := makeBook("b", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{b1, b2}))
	ps.mode = ModeSearching
	ps.searchMatches = []int{1, 0}
	ps.searchMatchIdx = 0
	ps.albumSelected = 1
	ps.searchSavedAlbum = 0
	ps2, _ := pressKey(ps, "enter")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps2.mode)
	}
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (kept)", ps2.albumSelected)
	}
}

func TestHandleSearching_Escape_RestoresOriginal(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearching
	ps.albumSelected = 5
	ps.searchSavedAlbum = 1
	ps.searchSavedOffset = 0
	ps2, _ := pressKey(ps, "esc")
	if ps2.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", ps2.mode)
	}
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1 (restored)", ps2.albumSelected)
	}
}

func TestHandleSearching_Slash_ReturnToSearch(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeSearching
	ps2, _ := pressKey(ps, "/")
	if ps2.mode != ModeSearch {
		t.Errorf("mode = %v, want ModeSearch", ps2.mode)
	}
}

//  handleMain — enter play

func TestHandleMain_Enter_SetsAlbumPlayingAndReturnsCmd(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000), nil)
	mock := &MockApiClient{
		GetPositionFn: func(hash string) (Position, error) { return Position{}, nil },
	}
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.api = mock
	ps.onAlbum = true
	ps.albumSelected = 0

	ps2, cmd := pressKey(ps, "enter")
	if ps2.albumPlaying == nil || *ps2.albumPlaying != 0 {
		t.Errorf("albumPlaying = %v, want &0", ps2.albumPlaying)
	}
	if cmd == nil {
		t.Error("enter should return fetchPositionAndPlayCmd (non-nil cmd)")
	}
	if ps2.lib.PlayingHash != "h1" {
		t.Errorf("PlayingHash = %q, want %q", ps2.lib.PlayingHash, "h1")
	}
}

func TestHandleMain_Enter_NoopWhenNoBooks(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps2, cmd := pressKey(ps, "enter")
	if ps2.albumPlaying != nil {
		t.Error("enter with no books should not set albumPlaying")
	}
	if cmd != nil {
		t.Error("enter with no books should return nil cmd")
	}
}

func TestHandleMain_Enter_NoopPlayWhenLibActive(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps2, _ := pressKey(ps, "enter")
	if ps2.albumPlaying != nil {
		t.Error("enter in library panel should not set albumPlaying")
	}
}

func TestHandleMain_Enter_SetsTrackPlayingFromPosition(t *testing.T) {
	pos := &Position{ChapterIndex: 2, Timestamp: 1}
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), pos)
	mock := &MockApiClient{
		GetPositionFn: func(hash string) (Position, error) { return Position{}, nil },
	}
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.api = mock
	ps.onAlbum = true
	ps.albumSelected = 0

	ps2, _ := pressKey(ps, "enter")
	if ps2.trackPlaying == nil || *ps2.trackPlaying != 2 {
		t.Errorf("trackPlaying = %v, want &2 (from position)", ps2.trackPlaying)
	}
}

//  handleMain — enter chapter play

func TestHandleMain_Enter_ChapterPlaysSelectedChapter(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.onAlbum = false
	ps.albumSelected = 0
	ps.trackSelected = 2

	ps2, _ := pressKey(ps, "enter")
	if ps2.albumPlaying == nil || *ps2.albumPlaying != 0 {
		t.Errorf("albumPlaying = %v, want &0", ps2.albumPlaying)
	}
	if ps2.trackPlaying == nil || *ps2.trackPlaying != 2 {
		t.Errorf("trackPlaying = %v, want &2", ps2.trackPlaying)
	}
	if ps2.lib.PlayingHash != "h1" {
		t.Errorf("PlayingHash = %q, want %q", ps2.lib.PlayingHash, "h1")
	}
}

func TestHandleMain_Enter_ChapterUpdatesLocalPosition(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.onAlbum = false
	ps.albumSelected = 0
	ps.trackSelected = 1

	ps2, _ := pressKey(ps, "enter")
	b := ps2.findPlayingBook()
	if b == nil || b.Position == nil || b.Position.ChapterIndex != 1 {
		t.Errorf("book position not updated to chapter 1")
	}
}

func TestHandleMain_Enter_ChapterNoopWhenNoBook(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.onAlbum = false
	ps2, _ := pressKey(ps, "enter")
	if ps2.albumPlaying != nil {
		t.Error("enter with no book should not set albumPlaying")
	}
}

//  handleMain — volume

func TestHandleMain_Minus_DecreasesVolume(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerVolume = 50
	ps2, _ := pressKey(ps, "-")
	if ps2.playerVolume != 40 {
		t.Errorf("playerVolume = %d, want 40", ps2.playerVolume)
	}
}

func TestHandleMain_Minus_ClampsAtZero(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerVolume = 7
	ps2, _ := pressKey(ps, "-")
	if ps2.playerVolume != 0 {
		t.Errorf("playerVolume = %d, want 0", ps2.playerVolume)
	}
}

func TestHandleMain_Minus_WorksWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.playerVolume = 50
	ps2, _ := pressKey(ps, "-")
	if ps2.playerVolume != 40 {
		t.Errorf("playerVolume = %d, want 40", ps2.playerVolume)
	}
}

func TestHandleMain_Equal_IncreasesVolume(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerVolume = 50
	ps2, _ := pressKey(ps, "=")
	if ps2.playerVolume != 60 {
		t.Errorf("playerVolume = %d, want 60", ps2.playerVolume)
	}
}

func TestHandleMain_Equal_ClampsAt100(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerVolume = 95
	ps2, _ := pressKey(ps, "=")
	if ps2.playerVolume != 100 {
		t.Errorf("playerVolume = %d, want 100", ps2.playerVolume)
	}
}

func TestHandleMain_Equal_WorksWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.playerVolume = 50
	ps2, _ := pressKey(ps, "=")
	if ps2.playerVolume != 60 {
		t.Errorf("playerVolume = %d, want 60", ps2.playerVolume)
	}
}

//  handleMain — seek

func TestHandleMain_Comma_SeeksBackward(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.positionMs = 30000
	ps2, _ := pressKey(ps, ",")
	if ps2.positionMs != 20000 {
		t.Errorf("positionMs = %d, want 20000", ps2.positionMs)
	}
}

func TestHandleMain_Comma_ClampsAtZero(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.positionMs = 5000
	ps2, _ := pressKey(ps, ",")
	if ps2.positionMs != 0 {
		t.Errorf("positionMs = %d, want 0", ps2.positionMs)
	}
}

func TestHandleMain_Comma_NoopWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.positionMs = 30000
	ps2, _ := pressKey(ps, ",")
	if ps2.positionMs != 30000 {
		t.Errorf("positionMs = %d, want 30000 (unchanged)", ps2.positionMs)
	}
}

func TestHandleMain_Dot_SeeksForward(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.positionMs = 30000
	ps.durationMs = 120000
	ps2, _ := pressKey(ps, ".")
	if ps2.positionMs != 40000 {
		t.Errorf("positionMs = %d, want 40000", ps2.positionMs)
	}
}

func TestHandleMain_Dot_ClampsAtDuration(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	idx := 0
	ps.albumPlaying = &idx
	ps.positionMs = 115000
	ps.durationMs = 120000
	ps2, _ := pressKey(ps, ".")
	if ps2.positionMs != 120000 {
		t.Errorf("positionMs = %d, want 120000", ps2.positionMs)
	}
}

func TestHandleMain_Dot_NoopWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.positionMs = 30000
	ps2, _ := pressKey(ps, ".")
	if ps2.positionMs != 30000 {
		t.Errorf("positionMs = %d, want 30000 (unchanged)", ps2.positionMs)
	}
}

//  handleMain — space play/pause

func TestHandleMain_Space_PausesWhenPlaying(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerPaused = false

	ps2, _ := pressKey(ps, "space")
	if !ps2.playerPaused {
		t.Error("space should pause when playing")
	}
}

func TestHandleMain_Space_PlaysWhenPaused(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	idx := 0
	ps.albumPlaying = &idx
	ps.playerPaused = true

	ps2, _ := pressKey(ps, "space")
	if ps2.playerPaused {
		t.Error("space should unpause when paused")
	}
}

func TestHandleMain_Space_NoopWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.albumPlaying = nil
	ps.playerPaused = false

	ps2, _ := pressKey(ps, "space")
	if ps2.playerPaused {
		t.Error("space when not playing should not set playerPaused")
	}
}

//  handleKey — y/n conflict handlers

func TestHandleKey_Y_ClearsConflictAndReturnsSync(t *testing.T) {
	s := newTestStore(t)
	mock := &MockApiClient{
		PutPositionFn: func(hash string, pos Position) error { return nil },
	}
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.api = mock
	ps.store = s
	idx := 0
	ps.albumPlaying = &idx
	localPos := Position{ChapterIndex: 1, ChapterPosition: 5000, Timestamp: 200}
	srvPos := Position{ChapterIndex: 2, ChapterPosition: 3000, Timestamp: 150}
	ps.conflictHash = "h1"
	ps.conflictLocal = &localPos
	ps.conflictServer = &srvPos

	ps2, cmd := pressKey(ps, "y")
	if ps2.conflictHash != "" {
		t.Errorf("conflictHash = %q, want empty after y", ps2.conflictHash)
	}
	if ps2.conflictLocal != nil || ps2.conflictServer != nil {
		t.Error("conflict fields should be nil after y")
	}
	if cmd == nil {
		t.Error("y should return a sync cmd")
	}
	msg := cmd()
	if _, ok := msg.(syncDoneMsg); !ok {
		t.Errorf("cmd() = %T, want syncDoneMsg", msg)
	}
}

func TestHandleKey_Y_SetsTrackToLocalChapter(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	idx := 0
	ps.albumPlaying = &idx
	localPos := Position{ChapterIndex: 1, ChapterPosition: 5000, Timestamp: 200}
	srvPos := Position{ChapterIndex: 2, ChapterPosition: 3000, Timestamp: 150}
	ps.conflictHash = "h1"
	ps.conflictLocal = &localPos
	ps.conflictServer = &srvPos

	ps2, _ := pressKey(ps, "y")
	if ps2.trackSelected != 1 {
		t.Errorf("trackSelected = %d, want 1 (local chapter)", ps2.trackSelected)
	}
}

func TestHandleKey_N_ClearsConflictAndSavesServerPos(t *testing.T) {
	s := newTestStore(t)
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.store = s
	idx := 0
	ps.albumPlaying = &idx
	localPos := Position{ChapterIndex: 1, ChapterPosition: 5000, Timestamp: 200}
	srvPos := Position{ChapterIndex: 2, ChapterPosition: 3000, Timestamp: 150}
	ps.conflictHash = "h1"
	ps.conflictLocal = &localPos
	ps.conflictServer = &srvPos

	ps2, _ := pressKey(ps, "n")
	if ps2.conflictHash != "" {
		t.Errorf("conflictHash = %q, want empty after n", ps2.conflictHash)
	}
	if ps2.conflictLocal != nil || ps2.conflictServer != nil {
		t.Error("conflict fields should be nil after n")
	}
	saved := s.LoadPosition("h1")
	if saved == nil || *saved != srvPos {
		t.Errorf("LoadPosition = %v, want %v (server pos saved)", saved, srvPos)
	}
}

func TestHandleKey_N_SetsTrackToServerChapter(t *testing.T) {
	book := makeBook("h1", DownloadReady, makeChapters(1000, 1000, 1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	idx := 0
	ps.albumPlaying = &idx
	localPos := Position{ChapterIndex: 1, ChapterPosition: 5000, Timestamp: 200}
	srvPos := Position{ChapterIndex: 2, ChapterPosition: 3000, Timestamp: 150}
	ps.conflictHash = "h1"
	ps.conflictLocal = &localPos
	ps.conflictServer = &srvPos

	ps2, _ := pressKey(ps, "n")
	if ps2.trackSelected != 2 {
		t.Errorf("trackSelected = %d, want 2 (server chapter)", ps2.trackSelected)
	}
}

func TestHandleKey_ConflictActive_OtherKeysBlocked(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.conflictHash = "h1"
	ps.conflictLocal = &Position{}
	ps.conflictServer = &Position{}

	// j should not scroll when conflict is active
	ps2, _ := pressKey(ps, "j")
	if ps2.conflictHash == "" {
		t.Error("conflict should not be cleared by j")
	}
	// q should not quit when conflict is active
	_, cmd := pressKey(ps, "q")
	if isQuitCmd(cmd) {
		t.Error("q should not quit while conflict dialog is active")
	}
}

//  handleMain — Q logout

func TestHandleMain_Q_ClearsCredentialsAndQuits(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveCredentials(Credentials{BaseURL: "http://x", Username: "u", Password: "p", Token: "t"}); err != nil {
		t.Fatal(err)
	}
	ps := newPlayerState(makeLib("", nil))
	ps.store = s

	_, cmd := pressKey(ps, "Q")
	if !isQuitCmd(cmd) {
		t.Error("Q should produce quit cmd")
	}
	if creds := s.LoadCredentials(); creds != nil {
		t.Error("Q should clear credentials")
	}
}

func TestHandleMain_Q_NilStore_StillQuits(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.store = nil
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Q with nil store should not panic: %v", r)
		}
	}()
	_, cmd := pressKey(ps, "Q")
	if !isQuitCmd(cmd) {
		t.Error("Q should produce quit cmd even with nil store")
	}
}

//  handleMain — speed cycle

func TestHandleMain_S_CyclesToNextSpeed(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.playerSpeed = 1.0 // index 2 → next is 1.5 (index 3)
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed != 1.5 {
		t.Errorf("playerSpeed = %v, want 1.5", ps2.playerSpeed)
	}
}

func TestHandleMain_S_WorksWhenNotPlaying(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.playerSpeed = 1.0
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed == 1.0 {
		t.Error("playerSpeed unchanged, want cycle even when nothing playing")
	}
}

func TestHandleMain_S_WrapsAroundAfterMax(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.playerSpeed = speedSteps[len(speedSteps)-1]
	ps2, _ := pressKey(ps, "s")
	if ps2.playerSpeed != speedSteps[0] {
		t.Errorf("playerSpeed = %v, want %v (wrapped)", ps2.playerSpeed, speedSteps[0])
	}
}

//  handleKey — helpForMode

func TestHandleKey_QuestionMark_SetsHelpForMode(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	ps.mode = ModeMain
	ps2, _ := pressKey(ps, "?")
	if ps2.helpForMode != ModeMain {
		t.Errorf("helpForMode = %v, want ModeMain", ps2.helpForMode)
	}
}

//  handleMain — g/G navigation

func TestHandleMain_G_LibJumpsToTop(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
		makeBook("c", DownloadRemote, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 2
	ps.libOffset = 1
	ps2, _ := pressKey(ps, "g")
	if ps2.libSel != 0 {
		t.Errorf("libSel = %d, want 0", ps2.libSel)
	}
	if ps2.libOffset != 0 {
		t.Errorf("libOffset = %d, want 0", ps2.libOffset)
	}
}

func TestHandleMain_ShiftG_LibJumpsToBottom(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadRemote, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
		makeBook("c", DownloadRemote, nil, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.libActive = true
	ps.libSel = 0
	ps.windowHeight = 40
	ps2, _ := pressKey(ps, "G")
	if ps2.libSel != 2 {
		t.Errorf("libSel = %d, want 2", ps2.libSel)
	}
}

func TestHandleMain_G_AlbumsJumpsToTop(t *testing.T) {
	chs := makeChapters(1000, 2000)
	books := []Audiobook{
		makeBook("a", DownloadReady, chs, nil),
		makeBook("b", DownloadReady, chs, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 1
	ps.albumOffset = 1
	ps2, _ := pressKey(ps, "g")
	if ps2.albumSelected != 0 {
		t.Errorf("albumSelected = %d, want 0", ps2.albumSelected)
	}
	if ps2.albumOffset != 0 {
		t.Errorf("albumOffset = %d, want 0", ps2.albumOffset)
	}
}

func TestHandleMain_ShiftG_AlbumsJumpsToBottom(t *testing.T) {
	chs := makeChapters(1000, 2000)
	books := []Audiobook{
		makeBook("a", DownloadReady, chs, nil),
		makeBook("b", DownloadReady, chs, nil),
	}
	ps := newPlayerState(makeLib("", books))
	ps.onAlbum = true
	ps.albumSelected = 0
	ps.windowHeight = 40
	ps2, _ := pressKey(ps, "G")
	if ps2.albumSelected != 1 {
		t.Errorf("albumSelected = %d, want 1", ps2.albumSelected)
	}
}

func TestHandleMain_G_ChaptersJumpsToTop(t *testing.T) {
	chs := makeChapters(1000, 2000, 3000)
	book := makeBook("a", DownloadReady, chs, &Position{ChapterIndex: 2})
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.onAlbum = false
	ps.trackSelected = 2
	ps.trackOffset = 1
	ps2, _ := pressKey(ps, "g")
	if ps2.trackSelected != 0 {
		t.Errorf("trackSelected = %d, want 0", ps2.trackSelected)
	}
	if ps2.trackOffset != 0 {
		t.Errorf("trackOffset = %d, want 0", ps2.trackOffset)
	}
}

func TestHandleMain_ShiftG_ChaptersJumpsToBottom(t *testing.T) {
	chs := makeChapters(1000, 2000, 3000)
	book := makeBook("a", DownloadReady, chs, &Position{ChapterIndex: 0})
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.onAlbum = false
	ps.trackSelected = 0
	ps.windowHeight = 40
	ps2, _ := pressKey(ps, "G")
	if ps2.trackSelected != 2 {
		t.Errorf("trackSelected = %d, want 2", ps2.trackSelected)
	}
}

//  handleMain — D delete from library

func TestHandleMain_ShiftD_DeletesReadyBookFromLib(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0
	ps.store = newTestStore(t)
	ps2, cmd := pressKey(ps, "D")
	if ps2.lib.Books[0].State != DownloadRemote {
		t.Error("D should immediately set state to DownloadRemote")
	}
	if cmd == nil {
		t.Error("D on ready book should return non-nil cmd")
	}
}

func TestHandleMain_ShiftD_NoopWhenRemote(t *testing.T) {
	book := makeBook("a", DownloadRemote, nil, nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0
	ps.store = newTestStore(t)
	_, cmd := pressKey(ps, "D")
	if cmd != nil {
		t.Error("D on non-ready book should return nil cmd")
	}
}

func TestHandleMain_ShiftD_NoopWhenNotLibActive(t *testing.T) {
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = false
	ps.onAlbum = true
	ps.store = newTestStore(t)
	_, cmd := pressKey(ps, "D")
	if cmd != nil {
		t.Error("D outside library should return nil cmd")
	}
}

func TestHandleMain_ShiftD_CmdDeletesDirectory(t *testing.T) {
	s := newTestStore(t)
	bookDir := filepath.Join(s.dataDir, "library", "a")
	os.MkdirAll(bookDir, 0755)
	book := makeBook("a", DownloadReady, makeChapters(1000), nil)
	ps := newPlayerState(makeLib("", []Audiobook{book}))
	ps.libActive = true
	ps.libSel = 0
	ps.store = s
	_, cmd := pressKey(ps, "D")
	if cmd == nil {
		t.Fatal("D should return non-nil cmd")
	}
	cmd()
	if _, err := os.Stat(bookDir); !os.IsNotExist(err) {
		t.Error("D cmd should delete the book directory")
	}
}

//  sortLibraryBooks

func TestSortLibraryBooks_ReadyLast(t *testing.T) {
	books := []Audiobook{
		{Hash: "ready", Author: "Z", State: DownloadReady},
		{Hash: "remote", Author: "A", State: DownloadRemote},
		{Hash: "inprog", Author: "B", State: DownloadInProgress},
	}
	sortLibraryBooks(books)
	if books[0].Hash != "inprog" {
		t.Errorf("[0] = %s, want inprog", books[0].Hash)
	}
	if books[1].Hash != "remote" {
		t.Errorf("[1] = %s, want remote", books[1].Hash)
	}
	if books[2].Hash != "ready" {
		t.Errorf("[2] = %s, want ready", books[2].Hash)
	}
}

func TestSortLibraryBooks_WithinGroupByAuthorDateTitle(t *testing.T) {
	books := []Audiobook{
		{Hash: "c", Author: "B", Date: 2000, Title: "Z", State: DownloadRemote},
		{Hash: "a", Author: "A", Date: 2000, Title: "Z", State: DownloadRemote},
		{Hash: "b", Author: "B", Date: 1990, Title: "Z", State: DownloadRemote},
		{Hash: "d", Author: "B", Date: 2000, Title: "A", State: DownloadRemote},
	}
	sortLibraryBooks(books)
	order := make([]string, len(books))
	for i, b := range books {
		order[i] = b.Hash
	}
	// A first, then B/1990, then B/2000/A, then B/2000/Z
	want := []string{"a", "b", "d", "c"}
	for i, h := range want {
		if order[i] != h {
			t.Errorf("position %d = %s, want %s (order=%v)", i, order[i], h, order)
			break
		}
	}
}

func TestUpdate_BooksResultMsg_SortsReadyLast(t *testing.T) {
	ps := newPlayerState(makeLib("", nil))
	books := []Audiobook{
		{Hash: "ready", Author: "A", State: DownloadReady},
		{Hash: "remote", Author: "B", State: DownloadRemote},
	}
	msg := booksResultMsg{books: books}
	model, _ := ps.Update(msg)
	ps2 := model.(*PlayerState)
	if ps2.lib.Books[0].Hash != "remote" {
		t.Errorf("[0] = %s, want remote", ps2.lib.Books[0].Hash)
	}
	if ps2.lib.Books[1].Hash != "ready" {
		t.Errorf("[1] = %s, want ready", ps2.lib.Books[1].Hash)
	}
}
