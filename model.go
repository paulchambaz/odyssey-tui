package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	WideThreshold = 100
	SideWidth     = 40
)

type Mode int

const (
	ModeMain   Mode = iota
	ModePlayer
	ModeHelp
)

type tickMsg struct{}

type posQueryMsg struct {
	posMs  int64
	durMs  int64
	ended  bool
	err    error
}

type PlayerState struct {
	mode       Mode
	returnMode Mode

	windowWidth  int
	windowHeight int
	program      *tea.Program

	lib *Library

	showLibrary      bool
	onAlbum          bool
	libActive        bool
	albumInfoOpen    bool
	libInfoOpen      bool
	albumInfoFocused bool
	libInfoFocused   bool

	// library panel
	libSel    int
	libOffset int
	libInfoOff int

	// audiobooks panel
	albumSelected  int
	albumOffset    int
	albumInfoOff   int

	// chapters panel (main view)
	trackSelected int
	trackOffset   int

	// playing state
	albumPlaying *int
	trackPlaying *int

	// player
	playerBook   *Audiobook
	playerChSel  int
	playerChOff  int
	playerPaused bool
	playerSpeed  float64
	playerVolume int
	mpv          *mpvPlayer

	// status bar
	statusMsg string
	statusErr bool
}

var playerSpeeds = []float64{0.50, 0.75, 1.00, 1.50, 2.00, 3.00}

func newPlayerState(lib *Library) *PlayerState {
	ps := &PlayerState{
		lib:          lib,
		mode:         ModeMain,
		onAlbum:      true,
		libInfoOpen:  true,
		playerSpeed:  1.00,
		playerVolume: 100,
	}
	for i, b := range ps.localBooks() {
		if b.Hash == lib.PlayingHash {
			ps.albumSelected = i
			idx := i
			ps.albumPlaying = &idx
			if b.Position != nil {
				ps.trackSelected = b.Position.ChapterIndex
				ps.playerChSel = b.Position.ChapterIndex
				chIdx := b.Position.ChapterIndex
				ps.trackPlaying = &chIdx
			}
			break
		}
	}
	return ps
}

func (ps *PlayerState) localBooks() []*Audiobook {
	var out []*Audiobook
	for i := range ps.lib.Books {
		if ps.lib.Books[i].State == DownloadReady {
			out = append(out, &ps.lib.Books[i])
		}
	}
	return out
}

func (ps *PlayerState) selectedLocalBook() *Audiobook {
	books := ps.localBooks()
	if len(books) == 0 || ps.albumSelected >= len(books) {
		return nil
	}
	return books[ps.albumSelected]
}

func (ps *PlayerState) bookProgress(b *Audiobook) float64 {
	if b.Position == nil || b.Position.Timestamp == 0 || b.Duration == 0 {
		return 0.0
	}
	var elapsed int64
	for i := 0; i < b.Position.ChapterIndex && i < len(b.Chapters); i++ {
		elapsed += b.Chapters[i].Duration
	}
	elapsed += b.Position.ChapterPosition
	pct := float64(elapsed) / float64(b.Duration)
	if pct > 1.0 {
		return 1.0
	}
	return pct
}

// bookElapsed returns the elapsed milliseconds for a book.
func (ps *PlayerState) bookElapsed(b *Audiobook) int64 {
	if b.Position == nil {
		return 0
	}
	var elapsed int64
	for i := 0; i < b.Position.ChapterIndex && i < len(b.Chapters); i++ {
		elapsed += b.Chapters[i].Duration
	}
	elapsed += b.Position.ChapterPosition
	return elapsed
}

func (ps *PlayerState) abPanelInnerH() int {
	h := ps.windowHeight - 2 - 2
	if h < 1 {
		return 1
	}
	if ps.albumInfoOpen {
		return h*6/10 + 4
	}
	return h
}

func (ps *PlayerState) libListH() int {
	h := ps.windowHeight - 2 - 2
	if h < 1 {
		return 1
	}
	if ps.libInfoOpen {
		return h*6/10 + 4
	}
	return h
}

func (ps *PlayerState) infoDetailH() int {
	h := ps.windowHeight - 2 - 2
	listH := h*6/10 + 4
	d := h - listH - 1
	if d < 0 {
		return 0
	}
	return d
}

func (ps *PlayerState) libInner() int {
	libW := ps.windowWidth * 30 / 100
	return libW - 2
}

func (ps *PlayerState) albumInner() int {
	abW := SideWidth
	if ps.windowWidth <= WideThreshold {
		abW = 2 * ps.windowWidth / 5
	}
	return abW - 2
}

func (ps *PlayerState) maxInfoOff(b *Audiobook, inner, detailH int) int {
	if b == nil || b.Description == "" {
		return 0
	}
	const fixedLines = 5
	visible := detailH - fixedLines
	if visible < 1 {
		return 0
	}
	n := len(wrapText(b.Description, inner)) - visible
	if n < 0 {
		return 0
	}
	return n
}

func (ps *PlayerState) playerInnerH() int {
	h := ps.windowHeight - 2 - 2
	if h < 1 {
		return 1
	}
	return h
}

func (ps *PlayerState) Init() tea.Cmd {
	return ps.tickCmd()
}

func (ps *PlayerState) tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (ps *PlayerState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ps.windowWidth = msg.Width
		ps.windowHeight = msg.Height
		return ps, nil
	case tickMsg:
		return ps, ps.tickCmd()
	case tea.KeyMsg:
		return ps.handleKey(msg)
	}
	return ps, nil
}

func clampOffset(offset, selected, panelH, padding, total int) int {
	if panelH <= 0 {
		return 0
	}
	if selected < offset+padding {
		offset = selected - padding
	}
	if selected >= offset+panelH-padding {
		offset = selected - panelH + 1 + padding
	}
	if offset < 0 {
		offset = 0
	}
	maxOff := total - panelH
	if maxOff < 0 {
		maxOff = 0
	}
	if offset > maxOff {
		offset = maxOff
	}
	return offset
}
