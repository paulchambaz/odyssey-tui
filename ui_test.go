package main

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

//  padLine 

func TestPadLine_Short(t *testing.T) {
	got := padLine("hi", 5)
	if lipgloss.Width(got) != 5 {
		t.Errorf("width = %d, want 5 (got %q)", lipgloss.Width(got), got)
	}
	if !strings.HasPrefix(got, "hi") {
		t.Errorf("expected %q to start with 'hi'", got)
	}
}

func TestPadLine_ExactLength(t *testing.T) {
	got := padLine("hello", 5)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestPadLine_AlreadyLonger(t *testing.T) {
	got := padLine("hello world", 5)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestPadLine_EmptyString(t *testing.T) {
	got := padLine("", 3)
	if lipgloss.Width(got) != 3 {
		t.Errorf("width = %d, want 3", lipgloss.Width(got))
	}
}

//  wrapText 

func TestWrapText_NoWrap(t *testing.T) {
	got := wrapText("hello", 10)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %v, want [hello]", got)
	}
}

func TestWrapText_ExactFit(t *testing.T) {
	got := wrapText("hello", 5)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %v, want [hello]", got)
	}
}

func TestWrapText_WrapsAtWord(t *testing.T) {
	got := wrapText("hello world", 7)
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %v", len(got), got)
	}
	if got[0] != "hello" || got[1] != "world" {
		t.Errorf("got %v, want [hello world]", got)
	}
}

func TestWrapText_MultipleWraps(t *testing.T) {
	got := wrapText("one two three four", 7)
	// "one two" (7) fits, then "three" (5), then "four" (4)
	if len(got) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %v", len(got), got)
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	got := wrapText("hello world", 0)
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestWrapText_EmptyString(t *testing.T) {
	got := wrapText("", 10)
	if len(got) != 0 {
		t.Errorf("got %v, want empty slice", got)
	}
}

func TestWrapText_LongWord_NeverSplit(t *testing.T) {
	got := wrapText("superlongwordthatexceedswidth next", 10)
	if len(got) == 0 {
		t.Fatal("expected at least one line")
	}
	if got[0] != "superlongwordthatexceedswidth" {
		t.Errorf("long word should not be split, got[0] = %q", got[0])
	}
}

//  bookChip 

func TestBookChip_Ready(t *testing.T) {
	b := &Audiobook{State: DownloadReady}
	chip, _ := bookChip(b)
	if chip != "[ready]" {
		t.Errorf("got %q, want [ready]", chip)
	}
}

func TestBookChip_InProgress(t *testing.T) {
	b := &Audiobook{State: DownloadInProgress, DownloadProgress: 0.42}
	chip, _ := bookChip(b)
	if !strings.Contains(chip, "42") {
		t.Errorf("expected %% in chip label, got %q", chip)
	}
	if !strings.Contains(chip, "downloading") {
		t.Errorf("expected 'downloading' in chip label, got %q", chip)
	}
}

func TestBookChip_Preparing(t *testing.T) {
	b := &Audiobook{State: DownloadPreparing}
	chip, _ := bookChip(b)
	if chip != "[queue]" {
		t.Errorf("got %q, want [queue]", chip)
	}
}

func TestBookChip_RemoteArchiveReady(t *testing.T) {
	b := &Audiobook{State: DownloadRemote, ArchiveReady: true}
	chip, _ := bookChip(b)
	if chip != "[download]" {
		t.Errorf("got %q, want [download]", chip)
	}
}

func TestBookChip_RemoteArchiveNotReady(t *testing.T) {
	b := &Audiobook{State: DownloadRemote, ArchiveReady: false}
	chip, _ := bookChip(b)
	if chip != "[building]" {
		t.Errorf("got %q, want [building]", chip)
	}
}

//  fmtSize 

func TestFmtSize_Zero(t *testing.T) {
	got := fmtSize(0)
	if got != "0 KB" {
		t.Errorf("got %q, want %q", got, "0 KB")
	}
}

func TestFmtSize_Bytes(t *testing.T) {
	got := fmtSize(512)
	if got != "0 KB" {
		t.Errorf("got %q, want %q", got, "0 KB")
	}
}

func TestFmtSize_Kilobytes(t *testing.T) {
	got := fmtSize(5 * 1024)
	if got != "5 KB" {
		t.Errorf("got %q, want %q", got, "5 KB")
	}
}

func TestFmtSize_Megabytes(t *testing.T) {
	got := fmtSize(50 * 1024 * 1024)
	if got != "50 MB" {
		t.Errorf("got %q, want %q", got, "50 MB")
	}
}

func TestFmtSize_Gigabytes(t *testing.T) {
	got := fmtSize(2 * 1024 * 1024 * 1024)
	if got != "2.0 GB" {
		t.Errorf("got %q, want %q", got, "2.0 GB")
	}
}

func TestFmtSize_FractionalGB(t *testing.T) {
	got := fmtSize(int64(1.5 * 1024 * 1024 * 1024))
	if got != "1.5 GB" {
		t.Errorf("got %q, want %q", got, "1.5 GB")
	}
}

//  chapterLabel 

func TestChapterLabel_InBounds(t *testing.T) {
	b := &Audiobook{Chapters: []Chapter{{Title: "Prologue"}, {Title: "Chapter 1"}}}
	got := chapterLabel(b, 0)
	if got != "Prologue" {
		t.Errorf("got %q, want %q", got, "Prologue")
	}
}

func TestChapterLabel_OutOfBounds(t *testing.T) {
	b := &Audiobook{Chapters: []Chapter{{Title: "Only"}}}
	got := chapterLabel(b, 5)
	if got != "Chapter 6" {
		t.Errorf("got %q, want %q", got, "Chapter 6")
	}
}

func TestChapterLabel_EmptyChapters(t *testing.T) {
	b := &Audiobook{}
	got := chapterLabel(b, 0)
	if got != "Chapter 1" {
		t.Errorf("got %q, want %q", got, "Chapter 1")
	}
}

//  renderPanel 

func panelLines(title string, w, h int, content string, focused bool, titleX int) []string {
	out := renderPanel(title, w, h, content, focused, titleX)
	lines := strings.Split(out, "\n")
	// drop trailing empty string from a trailing newline, if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func TestRenderPanel_LineCount(t *testing.T) {
	lines := panelLines("Title", 40, 10, "", false, -1)
	if len(lines) != 10 {
		t.Errorf("line count = %d, want 10", len(lines))
	}
}

func TestRenderPanel_LineWidth(t *testing.T) {
	const w = 40
	lines := panelLines("Title", w, 8, "", false, -1)
	for i, line := range lines {
		if lipgloss.Width(line) != w {
			t.Errorf("line[%d] width = %d, want %d (line=%q)", i, lipgloss.Width(line), w, line)
		}
	}
}

// runeIndexOf returns the rune (column) index of sub in s, or -1.
func runeIndexOf(s, sub string) int {
	byteIdx := strings.Index(s, sub)
	if byteIdx < 0 {
		return -1
	}
	return len([]rune(s[:byteIdx]))
}

func TestRenderPanel_TitleCentered(t *testing.T) {
	lines := panelLines("MyTitle", 40, 5, "", false, -1)
	top := lines[0]
	if !strings.Contains(top, "MyTitle") {
		t.Errorf("top border %q does not contain title", top)
	}
	// Count  chars before and after " MyTitle "
	idx := runeIndexOf(top, "MyTitle")
	after := len([]rune(top)) - idx - len([]rune("MyTitle"))
	// title is padded by " " on each side, so subtract 1 from idx for the leading space
	before := idx - 1  // leading space before "MyTitle"
	after -= 2         // trailing space after "MyTitle" and the "┐"
	if before-after > 2 || after-before > 2 {
		t.Errorf("title not centered: dash-before=%d dash-after=%d in %q", before, after, top)
	}
}

func TestRenderPanel_TitleLeftAligned(t *testing.T) {
	lines := panelLines("MyTitle", 40, 5, "", false, 1)
	top := lines[0]
	if !strings.Contains(top, "MyTitle") {
		t.Errorf("top border %q does not contain title", top)
	}
	// titleX=1: ┌ MyTitle ...┐ → "M" at rune index 3 (┌, , space)
	col := runeIndexOf(top, "MyTitle")
	if col > 5 {
		t.Errorf("title at rune col %d, expected near left (≤5): %q", col, top)
	}
}

func TestRenderPanel_FocusedVsUnfocused_SameDimensions(t *testing.T) {
	const w, h = 40, 8
	focused := panelLines("T", w, h, "content", true, -1)
	unfocused := panelLines("T", w, h, "content", false, -1)
	if len(focused) != len(unfocused) {
		t.Errorf("focused lines=%d unfocused lines=%d", len(focused), len(unfocused))
	}
	for i := range focused {
		if lipgloss.Width(focused[i]) != lipgloss.Width(unfocused[i]) {
			t.Errorf("line[%d] width focused=%d unfocused=%d",
				i, lipgloss.Width(focused[i]), lipgloss.Width(unfocused[i]))
		}
	}
}

//  buildBookDetail 

func makeDetailBook() *Audiobook {
	return &Audiobook{
		Hash:        "xyz",
		Title:       "Great Book",
		Author:      "Some Author",
		Date:        2022,
		Duration:    3600000,
		Size:        50 * 1024 * 1024,
		Description: strings.Repeat("word ", 40),
		State:       DownloadReady,
		ArchiveReady: true,
	}
}

func makeDetailState() *PlayerState {
	ps := newPlayerState(makeLib("", nil))
	ps.windowWidth = 80
	ps.windowHeight = 24
	return ps
}

func TestBuildBookDetail_LineCount(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	const h = 12
	lines := ps.buildBookDetail(b, 40, h, false, 0, false)
	if len(lines) != h {
		t.Errorf("line count = %d, want %d", len(lines), h)
	}
}

func TestBuildBookDetail_LineWidths(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	const inner = 40
	lines := ps.buildBookDetail(b, inner, 10, false, 0, false)
	for i, line := range lines {
		if lipgloss.Width(line) != inner {
			t.Errorf("line[%d] width = %d, want %d (line=%q)", i, lipgloss.Width(line), inner, line)
		}
	}
}

func TestBuildBookDetail_ContainsTitle(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	lines := ps.buildBookDetail(b, 40, 10, false, 0, false)
	if !strings.Contains(lines[0], "Great Book") {
		t.Errorf("line[0] = %q, want title 'Great Book'", lines[0])
	}
}

func TestBuildBookDetail_ContainsMeta(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	lines := ps.buildBookDetail(b, 40, 10, false, 0, false)
	meta := lines[1]
	if !strings.Contains(meta, "Some Author") {
		t.Errorf("line[1] = %q, want author", meta)
	}
	if !strings.Contains(meta, "2022") {
		t.Errorf("line[1] = %q, want year", meta)
	}
}

func TestBuildBookDetail_ChipShownWhenShowChip(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	lines := ps.buildBookDetail(b, 40, 10, true, 0, false)
	line3 := lines[3]
	if !strings.Contains(line3, "[ready]") {
		t.Errorf("line[3] = %q, want chip '[ready]'", line3)
	}
}

func TestBuildBookDetail_NoChipWhenNotShowChip(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	lines := ps.buildBookDetail(b, 40, 10, false, 0, false)
	line3 := lines[3]
	if strings.Contains(line3, "[ready]") {
		t.Errorf("line[3] = %q, should not contain chip", line3)
	}
}

func TestBuildBookDetail_DescriptionOffset(t *testing.T) {
	ps := makeDetailState()
	b := makeDetailBook()
	const inner, h = 40, 12
	offset0 := ps.buildBookDetail(b, inner, h, false, 0, false)
	offset2 := ps.buildBookDetail(b, inner, h, false, 2, false)
	// Both have same length
	if len(offset0) != len(offset2) {
		t.Fatalf("line count mismatch: %d vs %d", len(offset0), len(offset2))
	}
	// With a long description, desc content at offset 0 and offset 2 should differ
	joined0 := strings.Join(offset0[5:], " ")
	joined2 := strings.Join(offset2[5:], " ")
	if joined0 == joined2 {
		t.Error("offset 0 and offset 2 description content should differ")
	}
}

//  buildLibList 

func makeLibState(books []Audiobook) *PlayerState {
	lib := makeLib("", books)
	ps := newPlayerState(lib)
	ps.windowWidth = 80
	ps.windowHeight = 24
	return ps
}

func TestBuildLibList_LineCount(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
		makeBook("c", DownloadReady, nil, nil),
	}
	ps := makeLibState(books)
	const h = 8
	lines := ps.buildLibList(40, h)
	if len(lines) != h {
		t.Errorf("line count = %d, want %d", len(lines), h)
	}
}

func TestBuildLibList_LineWidths(t *testing.T) {
	books := []Audiobook{
		makeBook("a", DownloadReady, nil, nil),
		makeBook("b", DownloadRemote, nil, nil),
	}
	ps := makeLibState(books)
	const inner = 40
	lines := ps.buildLibList(inner, 5)
	for i, line := range lines {
		if lipgloss.Width(line) != inner {
			t.Errorf("line[%d] width = %d, want %d", i, lipgloss.Width(line), inner)
		}
	}
}

func TestBuildLibList_InProgressShowsPercentage(t *testing.T) {
	b := makeBook("dl", DownloadInProgress, nil, nil)
	b.DownloadProgress = 0.37
	books := []Audiobook{b}
	ps := makeLibState(books)
	lines := ps.buildLibList(40, 3)
	if !strings.Contains(lines[0], "37") {
		t.Errorf("line[0] = %q, want percentage 37%%", lines[0])
	}
}

