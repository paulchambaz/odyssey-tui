package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (ps *PlayerState) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logf("KEY", "key=%q mode=%s focus=%s", msg.String(), modeName(ps.mode), ps.focusName())
	switch msg.String() {
	case "ctrl+c":
		logf("KEY", "force quit")
		return ps, tea.Quit
	case "?":
		if ps.mode != ModeHelp && ps.mode != ModeSearch && ps.mode != ModeSearching {
			logf("MODE", "%s -> help", modeName(ps.mode))
			ps.returnMode = ps.mode
			ps.helpForMode = ps.mode
			ps.mode = ModeHelp
			return ps, nil
		}
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
		logf("KEY", "quit")
		if ps.albumPlaying != nil && ps.lib.PlayingHash != "" {
			chapter := 0
			if ps.trackPlaying != nil {
				chapter = *ps.trackPlaying
			}
			pos := Position{
				ChapterIndex:    chapter,
				ChapterPosition: ps.positionMs,
				Timestamp:       time.Now().Unix(),
			}
			return ps, syncAndQuitCmd(ps.api, ps.store, ps.lib.PlayingHash, pos)
		}
		return ps, tea.Quit

	case "d":
		ps.showLibrary = !ps.showLibrary
		logf("KEY", "toggle library panel -> showLibrary=%v", ps.showLibrary)
		if ps.showLibrary {
			ps.libActive = true
			ps.onAlbum = false
		} else {
			ps.libActive = false
			ps.onAlbum = true
		}

	case "enter":
		if ps.libActive && len(ps.lib.Books) > 0 {
			b := &ps.lib.Books[ps.libSel]
			switch b.State {
			case DownloadInProgress:
				logf("KEY", "cancel download hash=%s title=%q", b.Hash, b.Title)
				if ps.cancelDownloads != nil {
					if cancel, ok := ps.cancelDownloads[b.Hash]; ok {
						cancel()
						delete(ps.cancelDownloads, b.Hash)
					}
				}
			case DownloadRemote:
				logf("KEY", "start download hash=%s title=%q", b.Hash, b.Title)
				ctx, cancel := context.WithCancel(context.Background())
				if ps.cancelDownloads == nil {
					ps.cancelDownloads = make(map[string]context.CancelFunc)
				}
				ps.cancelDownloads[b.Hash] = cancel
				b.State = DownloadInProgress
				return ps, startDownloadCmd(ps.api, ps.store, b, ps.program, ctx)
			}
		} else if ps.onAlbum && !ps.libActive {
			books := ps.sortedLocalBooks()
			if ps.albumSelected < 0 || ps.albumSelected >= len(books) {
				return ps, nil
			}
			book := books[ps.albumSelected]
			logf("KEY", "play book hash=%s title=%q", book.Hash, book.Title)
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
		} else if !ps.onAlbum && !ps.libActive {
			book := ps.selectedLocalBook()
			if book == nil || ps.trackSelected < 0 || ps.trackSelected >= len(book.Chapters) {
				return ps, nil
			}
			chIdx := ps.trackSelected
			logf("KEY", "play chapter hash=%s title=%q ch=%d", book.Hash, book.Title, chIdx)
			idx := ps.albumSelected
			ps.albumPlaying = &idx
			ps.lib.PlayingHash = book.Hash
			ps.trackPlaying = &chIdx
			chPath := ps.chapterPath(book, chIdx)
			if ps.mpv != nil && chPath != "" {
				ps.mpv.loadFile(chPath, 0)
				ps.mpv.play()
			}
			ps.playerPaused = false
			pos := Position{ChapterIndex: chIdx, ChapterPosition: 0, Timestamp: time.Now().Unix()}
			for i := range ps.lib.Books {
				if ps.lib.Books[i].Hash == book.Hash {
					p := pos
					ps.lib.Books[i].Position = &p
					break
				}
			}
			if ps.store != nil && ps.lib.PlayingHash != "" {
				return ps, syncPositionCmd(ps.api, ps.store, ps.lib.PlayingHash, pos)
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
				logf("KEY", "delete local book hash=%s title=%q", book.Hash, book.Title)
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
			logf("KEY", "enter search mode")
			ps.enterSearch()
		}

	case " ":
		if ps.albumPlaying != nil {
			if ps.playerPaused {
				logf("KEY", "resume playback hash=%s", ps.lib.PlayingHash)
				if ps.mpv != nil {
					ps.mpv.play()
				}
				ps.playerPaused = false
			} else {
				logf("KEY", "pause playback hash=%s pos=%dms", ps.lib.PlayingHash, ps.positionMs)
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
		next := 0
		for i, v := range speedSteps {
			if v == ps.playerSpeed {
				next = (i + 1) % len(speedSteps)
				break
			}
		}
		s := speedSteps[next]
		logf("KEY", "speed cycle %.2f", s)
		ps.playerSpeed = s
		if ps.mpv != nil {
			_ = ps.mpv.setSpeed(s)
		}
		if ps.store != nil {
			_ = ps.store.SaveSetting("playback_speed", s)
		}

	case "-":
		v := ps.playerVolume - 10
		if v < 0 {
			v = 0
		}
		ps.playerVolume = v
		logf("KEY", "volume down -> %d", v)
		if ps.mpv != nil {
			_ = ps.mpv.setVolume(float64(v) / 100.0)
		}

	case "=":
		v := ps.playerVolume + 10
		if v > 100 {
			v = 100
		}
		ps.playerVolume = v
		logf("KEY", "volume up -> %d", v)
		if ps.mpv != nil {
			_ = ps.mpv.setVolume(float64(v) / 100.0)
		}

	case ",":
		if ps.albumPlaying != nil {
			logf("KEY", "seek backward 10s")
			if ps.mpv != nil {
				_ = ps.mpv.seekRelative(-10000)
			}
			newPos := ps.positionMs - 10000
			if newPos < 0 {
				newPos = 0
			}
			ps.positionMs = newPos
			chapter := 0
			if ps.trackPlaying != nil {
				chapter = *ps.trackPlaying
			}
			pos := Position{ChapterIndex: chapter, ChapterPosition: newPos, Timestamp: time.Now().Unix()}
			var cmds []tea.Cmd
			if ps.mpv != nil {
				cmds = append(cmds, queryPositionCmd(ps.mpv))
			}
			if ps.store != nil && ps.lib.PlayingHash != "" {
				cmds = append(cmds, syncPositionCmd(ps.api, ps.store, ps.lib.PlayingHash, pos))
			}
			return ps, tea.Batch(cmds...)
		}

	case ".":
		if ps.albumPlaying != nil {
			logf("KEY", "seek forward 10s")
			if ps.mpv != nil {
				_ = ps.mpv.seekRelative(10000)
			}
			newPos := ps.positionMs + 10000
			if ps.durationMs > 0 && newPos > ps.durationMs {
				newPos = ps.durationMs
			}
			ps.positionMs = newPos
			chapter := 0
			if ps.trackPlaying != nil {
				chapter = *ps.trackPlaying
			}
			pos := Position{ChapterIndex: chapter, ChapterPosition: newPos, Timestamp: time.Now().Unix()}
			var cmds []tea.Cmd
			if ps.mpv != nil {
				cmds = append(cmds, queryPositionCmd(ps.mpv))
			}
			if ps.store != nil && ps.lib.PlayingHash != "" {
				cmds = append(cmds, syncPositionCmd(ps.api, ps.store, ps.lib.PlayingHash, pos))
			}
			return ps, tea.Batch(cmds...)
		}

	case "Q":
		logf("KEY", "logout+quit")
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
			next := (ps.searchMatchIdx + 1) % len(ps.searchMatches)
			logf("KEY", "search next match=%d/%d", next+1, len(ps.searchMatches))
			ps.jumpToMatch(next)
		}
	case "p":
		if len(ps.searchMatches) > 0 {
			prev := (ps.searchMatchIdx + len(ps.searchMatches) - 1) % len(ps.searchMatches)
			logf("KEY", "search prev match=%d/%d", prev+1, len(ps.searchMatches))
			ps.jumpToMatch(prev)
		}
	case "enter":
		logf("KEY", "search confirm query=%q match=%d/%d", ps.searchQuery, ps.searchMatchIdx+1, len(ps.searchMatches))
		ps.exitSearchKeep()
	case "esc":
		logf("KEY", "search cancel")
		ps.cancelSearch()
	case "/":
		logf("MODE", "searching -> search (re-edit)")
		ps.mode = ModeSearch
	}
	return ps, nil
}

func (ps *PlayerState) handleHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "q", "esc":
		logf("MODE", "help -> %s", modeName(ps.returnMode))
		ps.mode = ps.returnMode
	}
	return ps, nil
}

func (ps *PlayerState) handleConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		logf("SYNC", "conflict resolved: keep local hash=%s", ps.conflictHash)
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
		logf("SYNC", "conflict resolved: take server hash=%s", ps.conflictHash)
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

