package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (ps *PlayerState) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return ps, tea.Quit
	case "?":
		if ps.mode != ModeHelp && ps.mode != ModeSearch && ps.mode != ModeSearching {
			ps.returnMode = ps.mode
			ps.helpForMode = ps.mode
			ps.mode = ModeHelp
			return ps, nil
		}
	}

	// Speed overlay intercepts all keys while open.
	if ps.showSpeed {
		return ps.handleSpeed(msg)
	}

	// Conflict dialog intercepts all keys until resolved.
	if ps.conflictHash != "" {
		return ps.handleConflict(msg)
	}

	switch ps.mode {
	case ModeMain:
		return ps.handleMain(msg)
	case ModeHelp:
		return ps.handleHelp(msg)
	case ModeSearch:
		return ps.handleSearch(msg)
	case ModeSearching:
		return ps.handleSearching(msg)
	}
	return ps, nil
}

func (ps *PlayerState) handleMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	books := ps.sortedLocalBooks()

	switch msg.String() {
	case "q":
		return ps, tea.Quit

	case "d":
		ps.showLibrary = !ps.showLibrary
		if !ps.showLibrary && ps.libActive {
			ps.libActive = false
			ps.onAlbum = true
		}

	case "enter":
		if ps.libActive && len(ps.lib.Books) > 0 {
			b := &ps.lib.Books[ps.libSel]
			switch b.State {
			case DownloadInProgress:
				if ps.cancelDownloads != nil {
					if cancel, ok := ps.cancelDownloads[b.Hash]; ok {
						cancel()
						delete(ps.cancelDownloads, b.Hash)
					}
				}
			case DownloadRemote:
				ctx, cancel := context.WithCancel(context.Background())
				if ps.cancelDownloads == nil {
					ps.cancelDownloads = make(map[string]context.CancelFunc)
				}
				ps.cancelDownloads[b.Hash] = cancel
				b.State = DownloadInProgress
				return ps, startDownloadCmd(ps.api, ps.store, b, ps.program, ctx)
			}
		}

	case "r":
		if ps.api != nil {
			return ps, fetchBooksCmd(ps.api)
		}

	case "h":
		switch {
		case !ps.libActive && !ps.onAlbum:
			ps.onAlbum = true
		case !ps.libActive && ps.onAlbum && ps.showLibrary:
			ps.libActive = true
			ps.onAlbum = false
		}

	case "l":
		switch {
		case ps.libActive:
			ps.libActive = false
			ps.onAlbum = true
		case ps.onAlbum:
			ps.onAlbum = false
		}

	case "j", "down":
		switch {
		case ps.libActive && ps.libInfoFocused:
			b := &ps.lib.Books[ps.libSel]
			if ps.libInfoOff < ps.maxInfoOff(b, ps.libInner(), ps.infoDetailH()) {
				ps.libInfoOff++
			}

		case ps.onAlbum && !ps.libActive && ps.albumInfoFocused:
			b := ps.selectedLocalBook()
			if ps.albumInfoOff < ps.maxInfoOff(b, ps.albumInner(), ps.infoDetailH()) {
				ps.albumInfoOff++
			}

		case ps.onAlbum && !ps.libActive:
			if ps.albumSelected < len(books)-1 {
				ps.albumSelected++
				ps.trackSelected = 0
				ps.trackOffset = 0
				ps.albumInfoOff = 0
				if b := books[ps.albumSelected]; b.Position != nil {
					ps.trackSelected = b.Position.ChapterIndex
				}
			}
			ps.albumOffset = clampOffset(ps.albumOffset, ps.albumSelected, ps.abPanelInnerH(), 2, len(books))

		case !ps.onAlbum && !ps.libActive:
			if b := ps.selectedLocalBook(); b != nil {
				if ps.trackSelected < len(b.Chapters)-1 {
					ps.trackSelected++
				}
				ps.trackOffset = clampOffset(ps.trackOffset, ps.trackSelected, ps.abPanelInnerH(), 2, len(b.Chapters))
			}

		case ps.libActive:
			if ps.libSel < len(ps.lib.Books)-1 {
				ps.libSel++
				ps.libInfoOff = 0
			}
			ps.libOffset = clampOffset(ps.libOffset, ps.libSel, ps.libListH(), 2, len(ps.lib.Books))
		}

	case "k", "up":
		switch {
		case ps.libActive && ps.libInfoFocused:
			if ps.libInfoOff > 0 {
				ps.libInfoOff--
			}

		case ps.onAlbum && !ps.libActive && ps.albumInfoFocused:
			if ps.albumInfoOff > 0 {
				ps.albumInfoOff--
			}

		case ps.onAlbum && !ps.libActive:
			if ps.albumSelected > 0 {
				ps.albumSelected--
				ps.trackSelected = 0
				ps.trackOffset = 0
				ps.albumInfoOff = 0
				if b := books[ps.albumSelected]; b.Position != nil {
					ps.trackSelected = b.Position.ChapterIndex
				}
			}
			ps.albumOffset = clampOffset(ps.albumOffset, ps.albumSelected, ps.abPanelInnerH(), 2, len(books))

		case !ps.onAlbum && !ps.libActive:
			if ps.trackSelected > 0 {
				ps.trackSelected--
			}
			if b := ps.selectedLocalBook(); b != nil {
				ps.trackOffset = clampOffset(ps.trackOffset, ps.trackSelected, ps.abPanelInnerH(), 2, len(b.Chapters))
			}

		case ps.libActive:
			if ps.libSel > 0 {
				ps.libSel--
				ps.libInfoOff = 0
			}
			ps.libOffset = clampOffset(ps.libOffset, ps.libSel, ps.libListH(), 2, len(ps.lib.Books))
		}

	case "I":
		if ps.libActive && ps.libInfoOpen {
			ps.libInfoFocused = !ps.libInfoFocused
		} else if ps.onAlbum && !ps.libActive && ps.albumInfoOpen {
			ps.albumInfoFocused = !ps.albumInfoFocused
		}

	case "i":
		if ps.libActive {
			ps.libInfoOpen = !ps.libInfoOpen
			if !ps.libInfoOpen {
				ps.libInfoFocused = false
			}
		} else {
			ps.albumInfoOpen = !ps.albumInfoOpen
			if !ps.albumInfoOpen {
				ps.albumInfoFocused = false
			}
		}

	case "x":
		if ps.onAlbum && !ps.libActive && ps.store != nil {
			if book := ps.selectedLocalBook(); book != nil {
				hash := book.Hash
				for i := range ps.lib.Books {
					if ps.lib.Books[i].Hash == hash {
						ps.lib.Books[i].State = DownloadRemote
						ps.lib.Books[i].Chapters = nil
						break
					}
				}
				return ps, deleteLocalBookCmd(ps.store, hash)
			}
		}

	case "/":
		if ps.onAlbum && !ps.libActive {
			ps.enterSearch()
		}

	case "p":
		if ps.onAlbum && !ps.libActive {
			books := ps.sortedLocalBooks()
			if ps.albumSelected < 0 || ps.albumSelected >= len(books) {
				return ps, nil
			}
			book := books[ps.albumSelected]
			idx := ps.albumSelected
			ps.albumPlaying = &idx
			ps.lib.PlayingHash = book.Hash
			chIdx := 0
			if book.Position != nil {
				chIdx = book.Position.ChapterIndex
			}
			ps.trackPlaying = &chIdx
			ps.trackSelected = chIdx
			if ps.api != nil {
				return ps, fetchPositionAndPlayCmd(ps.api, book.Hash)
			}
			// No API: play with local position directly
			chPath := ps.chapterPath(book, chIdx)
			if ps.mpv != nil && chPath != "" {
				seekMs := int64(0)
				if book.Position != nil {
					seekMs = book.Position.ChapterPosition
				}
				ps.mpv.loadFile(chPath, seekMs)
				ps.mpv.play()
			}
			ps.playerPaused = false
		}

	case " ":
		if ps.albumPlaying != nil {
			if ps.playerPaused {
				if ps.mpv != nil {
					ps.mpv.play()
				}
				ps.playerPaused = false
			} else {
				if ps.mpv != nil {
					ps.mpv.pause()
				}
				ps.playerPaused = true
				// Sync on pause
				if ps.store != nil && ps.lib.PlayingHash != "" {
					chapter := 0
					if ps.trackPlaying != nil {
						chapter = *ps.trackPlaying
					}
					pos := Position{
						ChapterIndex:    chapter,
						ChapterPosition: ps.positionMs,
						Timestamp:       time.Now().Unix(),
					}
					return ps, syncPositionCmd(ps.api, ps.store, ps.lib.PlayingHash, pos)
				}
			}
		}

	case "s":
		if ps.albumPlaying != nil {
			ps.speedSel = 2 // default 1.0×
			for i, v := range speedSteps {
				if v == ps.playerSpeed {
					ps.speedSel = i
					break
				}
			}
			ps.showSpeed = true
		}

	case "Q":
		if ps.store != nil {
			ps.store.ClearCredentials()
			ps.store.ClearAll()
		}
		if ps.mpv != nil {
			ps.mpv.quit()
		}
		return ps, tea.Quit

	}

	return ps, nil
}

func (ps *PlayerState) handleSpeed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		if ps.speedSel > 0 {
			ps.speedSel--
		}
	case "l", "right":
		if ps.speedSel < len(speedSteps)-1 {
			ps.speedSel++
		}
	case "enter":
		s := speedSteps[ps.speedSel]
		ps.playerSpeed = s
		if ps.mpv != nil {
			_ = ps.mpv.setSpeed(s)
		}
		if ps.store != nil {
			_ = ps.store.SaveSetting("speed", s)
		}
		ps.showSpeed = false
	case "esc", "s":
		ps.showSpeed = false
	}
	return ps, nil
}

func (ps *PlayerState) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		ps.cancelSearch()
	case tea.KeyEnter:
		ps.confirmSearch()
	case tea.KeyBackspace:
		if ps.searchQuery == "" {
			ps.cancelSearch()
		} else {
			ps.searchQuery = ps.searchQuery[:len(ps.searchQuery)-1]
			ps.runSearch()
			if len(ps.searchMatches) > 0 {
				ps.jumpToMatch(0)
			}
		}
	case tea.KeySpace:
		ps.searchQuery += " "
		ps.runSearch()
		if len(ps.searchMatches) > 0 {
			ps.jumpToMatch(0)
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			ps.searchQuery += string(r)
		}
		ps.runSearch()
		if len(ps.searchMatches) > 0 {
			ps.jumpToMatch(0)
		}
	}
	return ps, nil
}

func (ps *PlayerState) handleSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return ps, tea.Quit
	case "n":
		if len(ps.searchMatches) > 0 {
			ps.jumpToMatch((ps.searchMatchIdx + 1) % len(ps.searchMatches))
		}
	case "p":
		if len(ps.searchMatches) > 0 {
			ps.jumpToMatch((ps.searchMatchIdx + len(ps.searchMatches) - 1) % len(ps.searchMatches))
		}
	case "enter":
		ps.exitSearchKeep()
	case "esc":
		ps.cancelSearch()
	case "/":
		ps.mode = ModeSearch
	}
	return ps, nil
}

func (ps *PlayerState) handleHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "q", "esc":
		ps.mode = ps.returnMode
	}
	return ps, nil
}

func (ps *PlayerState) handleConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if ps.conflictLocal == nil {
			ps.conflictHash = ""
			ps.conflictServer = nil
			ps.conflictLocal = nil
			return ps, nil
		}
		loc := *ps.conflictLocal
		hash := ps.conflictHash
		book := ps.findPlayingBook()
		chIdx := loc.ChapterIndex
		if book != nil {
			if chIdx < 0 || chIdx >= len(book.Chapters) {
				chIdx = 0
			}
			chPath := ps.chapterPath(book, chIdx)
			if ps.mpv != nil && chPath != "" {
				ps.mpv.loadFile(chPath, loc.ChapterPosition)
				ps.mpv.play()
			}
			ps.playerPaused = false
			for i := range ps.lib.Books {
				if ps.lib.Books[i].Hash == hash {
					l := loc
					ps.lib.Books[i].Position = &l
					break
				}
			}
		}
		ps.trackSelected = chIdx
		if ps.trackPlaying != nil {
			*ps.trackPlaying = chIdx
		}
		ps.conflictHash = ""
		ps.conflictServer = nil
		ps.conflictLocal = nil
		if ps.api != nil && ps.store != nil {
			return ps, syncPositionCmd(ps.api, ps.store, hash, loc)
		}
		return ps, nil

	case "n":
		if ps.conflictServer == nil {
			ps.conflictHash = ""
			ps.conflictServer = nil
			ps.conflictLocal = nil
			return ps, nil
		}
		srv := *ps.conflictServer
		hash := ps.conflictHash
		book := ps.findPlayingBook()
		chIdx := srv.ChapterIndex
		if book != nil {
			if chIdx < 0 || chIdx >= len(book.Chapters) {
				chIdx = 0
			}
			chPath := ps.chapterPath(book, chIdx)
			if ps.mpv != nil && chPath != "" {
				ps.mpv.loadFile(chPath, srv.ChapterPosition)
				ps.mpv.play()
			}
			ps.playerPaused = false
			for i := range ps.lib.Books {
				if ps.lib.Books[i].Hash == hash {
					s := srv
					ps.lib.Books[i].Position = &s
					break
				}
			}
		}
		ps.trackSelected = chIdx
		if ps.trackPlaying != nil {
			*ps.trackPlaying = chIdx
		}
		if ps.store != nil {
			ps.store.SavePosition(hash, srv)
			ps.store.SaveServerPosition(hash, srv)
		}
		ps.conflictHash = ""
		ps.conflictServer = nil
		ps.conflictLocal = nil
		return ps, nil
	}
	// All other keys blocked while conflict is active.
	return ps, nil
}

