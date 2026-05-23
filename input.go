package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func (ps *PlayerState) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return ps, tea.Quit
	case "?":
		if ps.mode != ModeHelp {
			ps.returnMode = ps.mode
			ps.mode = ModeHelp
			return ps, nil
		}
	}

	switch ps.mode {
	case ModeMain:
		return ps.handleMain(msg)
	case ModePlayer:
		return ps.handlePlayer(msg)
	case ModeHelp:
		return ps.handleHelp(msg)
}
	return ps, nil
}

func (ps *PlayerState) handleMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	books := ps.localBooks()

	switch msg.String() {
	case "q":
		return ps, tea.Quit

	case "d":
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

	case "p", "enter":
		var book *Audiobook
		if ps.libActive && len(ps.lib.Books) > 0 {
			book = &ps.lib.Books[ps.libSel]
		} else if ps.onAlbum {
			book = ps.selectedLocalBook()
		}
		if book == nil || book.State != DownloadReady {
			ps.statusMsg = "not ready: download first"
			ps.statusErr = true
		} else {
			ps.playerBook = book
			ps.mode = ModePlayer
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

	}

	return ps, nil
}

func (ps *PlayerState) handlePlayer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		ps.mode = ModeMain

	case "s":
		for i, s := range playerSpeeds {
			if s == ps.playerSpeed {
				ps.playerSpeed = playerSpeeds[(i+1)%len(playerSpeeds)]
				break
			}
		}

	case "j", "down":
		if ps.playerBook != nil && ps.playerChSel < len(ps.playerBook.Chapters)-1 {
			ps.playerChSel++
			listH := ps.playerInnerH() - 9
			ps.playerChOff = clampOffset(ps.playerChOff, ps.playerChSel, listH, 2, len(ps.playerBook.Chapters))
		}

	case "k", "up":
		if ps.playerChSel > 0 {
			ps.playerChSel--
			listH := ps.playerInnerH() - 9
			if ps.playerBook != nil {
				ps.playerChOff = clampOffset(ps.playerChOff, ps.playerChSel, listH, 2, len(ps.playerBook.Chapters))
			}
		}
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

