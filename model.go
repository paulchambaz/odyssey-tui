package main

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	WideThreshold = 100
	SideWidth     = 40
)

type Mode int

const (
	ModeMain Mode = iota
	ModeHelp
	ModeSearch
	ModeSearching
)

var speedSteps = []float64{0.5, 0.75, 1.0, 1.5, 2.0, 2.5, 3.0}

type tickMsg struct{}

type posQueryMsg struct {
	posMs  int64
	durMs  int64
	ended  bool
	err    error
}

type booksResultMsg struct {
	books []Audiobook
	err   error
}

type localBooksMsg struct {
	books  []Audiobook
	counts map[string]int
	times  map[string]int64
}

type cachedLocalBooksMsg struct {
	books []Audiobook
}

type downloadProgressMsg struct {
	hash string
	pct  float64
}

type downloadDoneMsg struct {
	hash string
	err  error
}

type syncDoneMsg struct{ err error }

type positionSavedMsg struct {
	hash string
	pos  Position
}

type posFetchResultMsg struct {
	pos     *Position
	offline bool
}

type PlayerState struct {
	mode       Mode
	returnMode Mode

	windowWidth  int
	windowHeight int
	program      *tea.Program

	api            ApiClient
	store          *Store
	cancelDownloads map[string]context.CancelFunc

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
	playerPaused bool
	playerSpeed  float64
	playerVolume int
	mpv          *mpvPlayer

	// search
	searchQuery      string
	searchMatches    []int
	searchMatchIdx   int
	searchSavedAlbum  int
	searchSavedOffset int

	// download metadata
	downloadTimes    map[string]int64
	localByHash      map[string]Audiobook

	// position conflict dialog
	conflictHash   string
	conflictServer *Position
	conflictLocal  *Position

	// playback position (updated from posQueryMsg)
	positionMs int64
	durationMs int64
	tickCount  int

	// status bar
	statusMsg    string
	statusErr    bool
	statusClearAt int

	// speed overlay
	showSpeed bool
	speedSel  int

	// help context
	helpForMode Mode
}

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
	books := ps.sortedLocalBooks()
	if len(books) == 0 || ps.albumSelected >= len(books) {
		return nil
	}
	return books[ps.albumSelected]
}

func (ps *PlayerState) sortedLocalBooks() []*Audiobook {
	books := ps.localBooks()

	rank := func(b *Audiobook) int {
		p := ps.bookProgress(b)
		if p >= 1.0 {
			return 2
		}
		if p <= 0.0 {
			return 1
		}
		return 0
	}

	sort.Slice(books, func(i, j int) bool {
		ri, rj := rank(books[i]), rank(books[j])
		if ri != rj {
			return ri < rj
		}
		var ti, tj int64
		if books[i].Position != nil {
			ti = books[i].Position.Timestamp
		}
		if books[j].Position != nil {
			tj = books[j].Position.Timestamp
		}
		if ti != tj {
			return ti > tj
		}
		dti := ps.downloadTimes[books[i].Hash]
		dtj := ps.downloadTimes[books[j].Hash]
		if dti != dtj {
			return dti > dtj
		}
		return books[i].Title < books[j].Title
	})

	return books
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

func (ps *PlayerState) Init() tea.Cmd {
	cmds := []tea.Cmd{ps.tickCmd()}
	if ps.api != nil {
		cmds = append(cmds, fetchBooksCmd(ps.api))
	}
	if ps.store != nil {
		cmds = append(cmds, loadCachedLocalBooksCmd(ps.store), loadLocalBooksCmd(ps.store))
	}
	return tea.Batch(cmds...)
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
		books := ps.sortedLocalBooks()
		if len(books) > 0 {
			ps.albumOffset = clampOffset(ps.albumOffset, ps.albumSelected, ps.abPanelInnerH(), 2, len(books))
			if b := ps.selectedLocalBook(); b != nil {
				ps.trackOffset = clampOffset(ps.trackOffset, ps.trackSelected, ps.abPanelInnerH(), 2, len(b.Chapters))
			}
		}
		if ps.lib != nil {
			ps.libOffset = clampOffset(ps.libOffset, ps.libSel, ps.libListH(), 2, len(ps.lib.Books))
		}
		if b := ps.selectedLocalBook(); b != nil {
			maxA := ps.maxInfoOff(b, ps.albumInner(), ps.infoDetailH())
			if ps.albumInfoOff > maxA {
				ps.albumInfoOff = maxA
			}
		}
		if ps.lib != nil && len(ps.lib.Books) > 0 {
			lb := &ps.lib.Books[ps.libSel]
			maxL := ps.maxInfoOff(lb, ps.libInner(), ps.infoDetailH())
			if ps.libInfoOff > maxL {
				ps.libInfoOff = maxL
			}
		}
		return ps, nil
	case tickMsg:
		ps.tickCount++
		if ps.statusMsg != "" && ps.tickCount >= ps.statusClearAt {
			ps.statusMsg = ""
			ps.statusErr = false
		}
		cmds := []tea.Cmd{ps.tickCmd()}
		if ps.albumPlaying != nil && !ps.playerPaused && ps.mpv != nil {
			cmds = append(cmds, queryPositionCmd(ps.mpv))
		}
		return ps, tea.Batch(cmds...)
	case tea.KeyMsg:
		return ps.handleKey(msg)
	case booksResultMsg:
		if msg.err != nil {
			ps.statusMsg = msg.err.Error()
			ps.statusErr = true
			ps.statusClearAt = ps.tickCount + 4
			return ps, nil
		}
		ps.lib.Books = msg.books
		ps.statusErr = false
		for i := range ps.lib.Books {
			if local, ok := ps.localByHash[ps.lib.Books[i].Hash]; ok {
				ps.lib.Books[i].State = local.State
				ps.lib.Books[i].Chapters = local.Chapters
				if local.Position != nil {
					pos := *local.Position
					ps.lib.Books[i].Position = &pos
				}
			}
		}
		return ps, nil
	case cachedLocalBooksMsg:
		if ps.localByHash != nil {
			return ps, nil
		}
		ps.localByHash = make(map[string]Audiobook)
		for _, b := range msg.books {
			ps.localByHash[b.Hash] = b
		}
		for i := range ps.lib.Books {
			if local, ok := ps.localByHash[ps.lib.Books[i].Hash]; ok {
				ps.lib.Books[i].State = local.State
				ps.lib.Books[i].Chapters = local.Chapters
			}
		}
		return ps, nil
	case localBooksMsg:
		ps.localByHash = make(map[string]Audiobook)
		for _, b := range msg.books {
			ps.localByHash[b.Hash] = b
		}
		for i := range ps.lib.Books {
			if local, ok := ps.localByHash[ps.lib.Books[i].Hash]; ok {
				ps.lib.Books[i].State = local.State
				ps.lib.Books[i].Chapters = local.Chapters
				if local.Position != nil {
					pos := *local.Position
					ps.lib.Books[i].Position = &pos
				}
			}
		}
		if msg.times != nil {
			ps.downloadTimes = msg.times
		}
		// Update cursor to the current book's saved chapter (if not playing).
		if ps.albumPlaying == nil {
			sorted := ps.sortedLocalBooks()
			if ps.albumSelected < len(sorted) {
				if b := sorted[ps.albumSelected]; b.Position != nil {
					ps.trackSelected = b.Position.ChapterIndex
					ps.trackOffset = clampOffset(ps.trackOffset, ps.trackSelected, ps.abPanelInnerH(), 2, len(b.Chapters))
				}
			}
		}
		if ps.store != nil {
			return ps, saveLocalBooksCacheCmd(ps.store, msg.books, msg.times)
		}
		return ps, nil
	case downloadProgressMsg:
		for i := range ps.lib.Books {
			if ps.lib.Books[i].Hash == msg.hash {
				ps.lib.Books[i].DownloadProgress = msg.pct
				break
			}
		}
		return ps, nil
	case downloadDoneMsg:
		if ps.cancelDownloads != nil {
			delete(ps.cancelDownloads, msg.hash)
		}
		var reload bool
		for i := range ps.lib.Books {
			if ps.lib.Books[i].Hash == msg.hash {
				if msg.err != nil {
					ps.lib.Books[i].State = DownloadRemote
					if !errors.Is(msg.err, context.Canceled) {
						ps.statusMsg = "download failed: " + msg.err.Error()
						ps.statusErr = true
						ps.statusClearAt = ps.tickCount + 4
					}
				} else {
					ps.lib.Books[i].State = DownloadReady
					reload = true
				}
				ps.lib.Books[i].DownloadProgress = 0
				break
			}
		}
		if reload && ps.store != nil {
			cmds := []tea.Cmd{loadLocalBooksCmd(ps.store)}
			if ps.api != nil {
				cmds = append(cmds, fetchAndSavePositionCmd(ps.api, ps.store, msg.hash))
			}
			return ps, tea.Batch(cmds...)
		}
		return ps, nil

	case posFetchResultMsg:
		if ps.albumPlaying == nil {
			return ps, nil
		}
		book := ps.findPlayingBook()
		if book == nil {
			return ps, nil
		}
		hash := ps.lib.PlayingHash

		localPos := Position{}
		if book.Position != nil {
			localPos = *book.Position
		}

		if msg.offline {
			chIdx := localPos.ChapterIndex
			if chIdx < 0 || chIdx >= len(book.Chapters) {
				chIdx = 0
			}
			chPath := ps.chapterPath(book, chIdx)
			if ps.mpv != nil && chPath != "" {
				ps.mpv.loadFile(chPath, localPos.ChapterPosition)
				ps.mpv.play()
			}
			ps.playerPaused = false
			idx := chIdx
			ps.trackPlaying = &idx
			ps.trackSelected = chIdx
			return ps, nil
		}

		serverPos := Position{}
		if msg.pos != nil {
			serverPos = *msg.pos
		}
		lastSrvPos := Position{}
		if ps.store != nil {
			if p := ps.store.LoadServerPosition(hash); p != nil {
				lastSrvPos = *p
			}
		}

		resolved, conflict := resolvePosition(serverPos, localPos, lastSrvPos)

		if conflict {
			ps.conflictHash = hash
			srv := serverPos
			loc := localPos
			ps.conflictServer = &srv
			ps.conflictLocal = &loc
			return ps, nil
		}

		// No conflict: persist and start playing
		if ps.store != nil {
			ps.store.SavePosition(hash, resolved)
			ps.store.SaveServerPosition(hash, resolved)
		}
		chIdx := resolved.ChapterIndex
		if chIdx < 0 || chIdx >= len(book.Chapters) {
			chIdx = 0
		}
		for i := range ps.lib.Books {
			if ps.lib.Books[i].Hash == hash {
				r := resolved
				ps.lib.Books[i].Position = &r
				break
			}
		}
		chPath := ps.chapterPath(book, chIdx)
		if ps.mpv != nil && chPath != "" {
			ps.mpv.loadFile(chPath, resolved.ChapterPosition)
			ps.mpv.play()
		}
		ps.playerPaused = false
		idx := chIdx
		ps.trackPlaying = &idx
		ps.trackSelected = chIdx
		return ps, nil

	case posQueryMsg:
		if msg.err != nil || ps.albumPlaying == nil {
			return ps, nil
		}
		ps.positionMs = msg.posMs
		ps.durationMs = msg.durMs

		hash := ps.lib.PlayingHash
		chapter := 0
		if ps.trackPlaying != nil {
			chapter = *ps.trackPlaying
		}
		// Cursor follows the playing chapter so the panel always shows where we are.
		ps.trackSelected = chapter
		now := time.Now().Unix()
		for i := range ps.lib.Books {
			if ps.lib.Books[i].Hash == hash {
				if ps.lib.Books[i].Position == nil {
					p := Position{}
					ps.lib.Books[i].Position = &p
				}
				ps.lib.Books[i].Position.ChapterIndex = chapter
				ps.lib.Books[i].Position.ChapterPosition = msg.posMs
				ps.lib.Books[i].Position.Timestamp = now
				break
			}
		}

		var cmds []tea.Cmd
		book := ps.findPlayingBook()
		if book != nil {
			ps.trackOffset = clampOffset(ps.trackOffset, ps.trackSelected, ps.abPanelInnerH(), 2, len(book.Chapters))
		}

		if msg.ended && book != nil && ps.trackPlaying != nil {
			next := *ps.trackPlaying + 1
			if next < len(book.Chapters) {
				ps.trackPlaying = &next
				ps.trackSelected = next
				chPath := ps.chapterPath(book, next)
				if ps.mpv != nil && chPath != "" {
					ps.mpv.loadFile(chPath, 0)
				}
			} else {
				// last chapter ended
				ps.albumPlaying = nil
				ps.trackPlaying = nil
				ps.playerPaused = true
			}
		}

		if ps.tickCount%60 == 0 && ps.api != nil && ps.store != nil && hash != "" {
			pos := Position{
				ChapterIndex:    chapter,
				ChapterPosition: ps.positionMs,
				Timestamp:       now,
			}
			cmds = append(cmds, syncPositionCmd(ps.api, ps.store, hash, pos))
		}
		return ps, tea.Batch(cmds...)

	case positionSavedMsg:
		for i := range ps.lib.Books {
			if ps.lib.Books[i].Hash == msg.hash {
				p := msg.pos
				ps.lib.Books[i].Position = &p
				break
			}
		}
		return ps, nil

	case syncDoneMsg:
		if msg.err != nil {
			ps.statusMsg = "sync failed: " + msg.err.Error()
			ps.statusErr = true
			ps.statusClearAt = ps.tickCount + 4
		}
		return ps, nil
	}
	return ps, nil
}

func fetchBooksCmd(api ApiClient) tea.Cmd {
	return func() tea.Msg {
		books, err := api.GetAudiobooks()
		return booksResultMsg{books: books, err: err}
	}
}

func loadLocalBooksCmd(store *Store) tea.Cmd {
	return func() tea.Msg {
		books, counts, times, err := store.LoadLocalAudiobooks()
		if err != nil {
			return localBooksMsg{}
		}
		for i := range books {
			if pos := store.LoadPosition(books[i].Hash); pos != nil {
				books[i].Position = pos
			}
		}
		return localBooksMsg{books: books, counts: counts, times: times}
	}
}

func loadCachedLocalBooksCmd(store *Store) tea.Cmd {
	return func() tea.Msg {
		books, _ := store.LoadLocalBooksCache()
		if books == nil {
			return nil
		}
		return cachedLocalBooksMsg{books: books}
	}
}

func saveLocalBooksCacheCmd(store *Store, books []Audiobook, times map[string]int64) tea.Cmd {
	return func() tea.Msg {
		store.SaveLocalBooksCache(books, times)
		return nil
	}
}

// resolvePosition implements the five-case merge algorithm from MainActivity.kt:788-852.
func resolvePosition(server, local, lastServer Position) (Position, bool) {
	// Case 1: nearly identical — same chapter, < 30 s apart → pick earlier
	if server.ChapterIndex == local.ChapterIndex {
		diff := server.ChapterPosition - local.ChapterPosition
		if diff < 0 {
			diff = -diff
		}
		if diff < 30_000 {
			if local.ChapterPosition <= server.ChapterPosition {
				return local, false
			}
			return server, false
		}
	}
	// Case 2: server unchanged since last sync AND local newer → local wins
	if server == lastServer && local.Timestamp > server.Timestamp {
		return local, false
	}
	// Case 3: server newer → server wins
	if server.Timestamp > local.Timestamp {
		return server, false
	}
	// Case 4: local newer → conflict
	if local.Timestamp > server.Timestamp {
		return local, true
	}
	// Case 5: fallback → server wins
	return server, false
}

func syncPositionCmd(api ApiClient, store *Store, hash string, pos Position) tea.Cmd {
	return func() tea.Msg {
		if store != nil {
			store.SavePosition(hash, pos)
		}
		if api != nil {
			if err := api.PutPosition(hash, pos); err != nil {
				return syncDoneMsg{err: err}
			}
			if store != nil {
				store.SaveServerPosition(hash, pos)
			}
		}
		return syncDoneMsg{}
	}
}

func fetchAndSavePositionCmd(api ApiClient, store *Store, hash string) tea.Cmd {
	return func() tea.Msg {
		pos, err := api.GetPosition(hash)
		if err != nil {
			return nil
		}
		if store != nil {
			store.SavePosition(hash, pos)
			store.SaveServerPosition(hash, pos)
		}
		return positionSavedMsg{hash: hash, pos: pos}
	}
}

func fetchPositionAndPlayCmd(api ApiClient, hash string) tea.Cmd {
	return func() tea.Msg {
		if api == nil {
			return posFetchResultMsg{offline: true}
		}
		pos, err := api.GetPosition(hash)
		if err != nil {
			return posFetchResultMsg{offline: true}
		}
		return posFetchResultMsg{pos: &pos}
	}
}

func (ps *PlayerState) chapterPath(book *Audiobook, idx int) string {
	if ps.store == nil || book == nil || idx < 0 || idx >= len(book.Chapters) {
		return ""
	}
	return filepath.Join(ps.store.LibraryDir(book.Hash), book.Chapters[idx].Path)
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
