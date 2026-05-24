package main

import (
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func fuzzyScore(query, target string) int {
	query = strings.ToLower(query)
	target = strings.ToLower(target)
	if query == "" || target == "" {
		return 0
	}

	qi := 0
	score := 0
	prevIdx := -1

	for i := 0; i < len(target) && qi < len(query); i++ {
		if target[i] == query[qi] {
			score++
			if prevIdx >= 0 && i == prevIdx+1 {
				score += 3
			}
			if i == 0 || target[i-1] == ' ' {
				score += 5
			}
			prevIdx = i
			qi++
		}
	}

	if qi < len(query) {
		return 0
	}
	return score
}

func (ps *PlayerState) runSearch() {
	ps.searchMatches = nil
	ps.searchMatchIdx = 0
	if ps.searchQuery == "" {
		return
	}

	type result struct {
		idx   int
		score int
	}
	var results []result

	books := ps.localBooks()
	for i, b := range books {
		s1 := fuzzyScore(ps.searchQuery, b.Title+" "+b.Author)
		s2 := fuzzyScore(ps.searchQuery, b.Author+" "+b.Title)
		score := s1
		if s2 > s1 {
			score = s2
		}
		results = append(results, result{i, score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	for _, r := range results {
		if r.score > 0 {
			ps.searchMatches = append(ps.searchMatches, r.idx)
		}
	}
}

func (ps *PlayerState) jumpToMatch(idx int) {
	ps.searchMatchIdx = idx
	ps.albumSelected = ps.searchMatches[idx]
	ps.trackSelected = 0
	ps.trackOffset = 0
}

func (ps *PlayerState) enterSearch() {
	ps.searchSavedAlbum = ps.albumSelected
	ps.searchSavedOffset = ps.albumOffset
	ps.searchQuery = ""
	ps.searchMatches = nil
	ps.searchMatchIdx = 0
	ps.mode = ModeSearch
}

func (ps *PlayerState) cancelSearch() {
	ps.albumSelected = ps.searchSavedAlbum
	ps.albumOffset = ps.searchSavedOffset
	ps.searchQuery = ""
	ps.searchMatches = nil
	ps.searchMatchIdx = 0
	ps.mode = ModeMain
}

func (ps *PlayerState) confirmSearch() {
	if ps.searchQuery == "" {
		ps.cancelSearch()
		return
	}
	ps.mode = ModeSearching
}

func (ps *PlayerState) exitSearchKeep() {
	ps.searchQuery = ""
	ps.searchMatches = nil
	ps.searchMatchIdx = 0
	ps.mode = ModeMain
}

func deleteLocalBookCmd(store *Store, hash string) tea.Cmd {
	return func() tea.Msg {
		os.RemoveAll(store.LibraryDir(hash))
		store.SaveDownloadState(hash, "remote")
		books, counts, times, err := store.LoadLocalAudiobooks()
		if err != nil {
			return localBooksMsg{}
		}
		return localBooksMsg{books: books, counts: counts, times: times}
	}
}
