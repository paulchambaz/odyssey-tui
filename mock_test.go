package main

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

//  Mock API client 

type MockApiClient struct {
	LoginFn             func(baseURL, user, pass string) (string, error)
	RegisterFn          func(baseURL, user, pass string) (string, error)
	GetAudiobooksFn     func() ([]Audiobook, error)
	GetAudiobookFn      func(hash string) (Audiobook, error)
	DownloadAudiobookFn func(hash, dest string, startByte int64, onProgress func(int64, int64)) error
	CancelDownloadFn    func()
	GetPositionFn       func(hash string) (Position, error)
	PutPositionFn       func(hash string, pos Position) error
}

var _ ApiClient = (*MockApiClient)(nil)

func (m *MockApiClient) Login(baseURL, user, pass string) (string, error) {
	return m.LoginFn(baseURL, user, pass)
}
func (m *MockApiClient) Register(baseURL, user, pass string) (string, error) {
	return m.RegisterFn(baseURL, user, pass)
}
func (m *MockApiClient) GetAudiobooks() ([]Audiobook, error) {
	return m.GetAudiobooksFn()
}
func (m *MockApiClient) GetAudiobook(hash string) (Audiobook, error) {
	return m.GetAudiobookFn(hash)
}
func (m *MockApiClient) DownloadAudiobook(hash, dest string, startByte int64, onProgress func(int64, int64)) error {
	return m.DownloadAudiobookFn(hash, dest, startByte, onProgress)
}
func (m *MockApiClient) CancelDownload() { m.CancelDownloadFn() }
func (m *MockApiClient) GetPosition(hash string) (Position, error) {
	return m.GetPositionFn(hash)
}
func (m *MockApiClient) PutPosition(hash string, pos Position) error {
	return m.PutPositionFn(hash, pos)
}

//  Test helpers 

func makeChapters(durations ...int64) []Chapter {
	chs := make([]Chapter, len(durations))
	for i, d := range durations {
		chs[i] = Chapter{Title: "Chapter " + string(rune('1'+i)), Duration: d}
	}
	return chs
}

func makeBook(hash string, state DownloadState, chapters []Chapter, pos *Position) Audiobook {
	var dur int64
	for _, ch := range chapters {
		dur += ch.Duration
	}
	return Audiobook{
		Hash:     hash,
		Title:    "Title " + hash,
		Author:   "Author " + hash,
		Duration: dur,
		Chapters: chapters,
		Position: pos,
		State:    state,
	}
}

func makeLib(playingHash string, books []Audiobook) *Library {
	return &Library{PlayingHash: playingHash, Books: books}
}

// pressKey sends a key to the player state and returns the updated state and cmd.
func pressKey(ps *PlayerState, key string) (*PlayerState, tea.Cmd) {
	var msg tea.KeyMsg
	switch key {
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	model, cmd := ps.handleKey(msg)
	return model.(*PlayerState), cmd
}

// isQuitCmd reports whether cmd produces a QuitMsg when called.
func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// stripANSI removes ANSI escape sequences from s.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[mKHJA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
