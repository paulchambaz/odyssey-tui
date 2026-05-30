package main

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type LibQuery struct {
	Text     string
	Tags     []string
	YearLow  *int
	YearHigh *int
}

var reTag      = regexp.MustCompile(`\+\S+`)
var reYearHigh = regexp.MustCompile(`<(\d+)`)
var reYearLow  = regexp.MustCompile(`>(\d+)`)

func parseLibQuery(raw string) LibQuery {
	var q LibQuery

	raw = reTag.ReplaceAllStringFunc(raw, func(m string) string {
		q.Tags = append(q.Tags, m[1:])
		return " "
	})

	raw = reYearLow.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reYearLow.FindStringSubmatch(m)
		if y, err := strconv.Atoi(sub[1]); err == nil {
			if q.YearLow == nil || y > *q.YearLow {
				q.YearLow = &y
			}
		}
		return " "
	})

	raw = reYearHigh.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reYearHigh.FindStringSubmatch(m)
		if y, err := strconv.Atoi(sub[1]); err == nil {
			if q.YearHigh == nil || y < *q.YearHigh {
				q.YearHigh = &y
			}
		}
		return " "
	})

	q.Text = strings.TrimSpace(raw)

	// Strip words starting with filter chars that weren't extracted as complete tokens
	// (e.g., lone "<", ">", "+", or incomplete "<20" still being typed).
	// Exception: filter chars in the middle of a word (e.g., "abc<def") are kept because
	// strings.Fields returns them as a single word not starting with the filter char.
	if q.Text != "" {
		words := strings.Fields(q.Text)
		kept := words[:0]
		for _, w := range words {
			if w[0] == '<' || w[0] == '>' || w[0] == '+' {
				continue
			}
			kept = append(kept, w)
		}
		q.Text = strings.Join(kept, " ")
	}

	return q
}

func libTextScore(text string, b Audiobook) int {
	if text == "" {
		return 1
	}
	if idx := strings.Index(text, " - "); idx >= 0 {
		left := text[:idx]
		right := text[idx+3:]
		s1t := fuzzyScore(left, b.Title)
		s1a := fuzzyScore(right, b.Author)
		s2a := fuzzyScore(left, b.Author)
		s2t := fuzzyScore(right, b.Title)
		var best int
		if s1t > 0 && s1a > 0 {
			best = s1t + s1a
		}
		if s2a > 0 && s2t > 0 {
			if s := s2a + s2t; s > best {
				best = s
			}
		}
		return best
	}
	s1 := fuzzyScore(text, b.Title+" "+b.Author)
	s2 := fuzzyScore(text, b.Author+" "+b.Title)
	if s2 > s1 {
		return s2
	}
	return s1
}

func allGenres(books []Audiobook) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, b := range books {
		for _, g := range b.Genres {
			if _, ok := seen[g]; !ok {
				seen[g] = struct{}{}
				out = append(out, g)
			}
		}
	}
	return out
}

func buildTagCandidates(tagQuery string, genres []string) []string {
	var out []string
	for _, g := range genres {
		if fuzzyScore(tagQuery, g) > 0 {
			out = append(out, g)
		}
	}
	return out
}

func scoreBookTags(bookGenres []string, tagLists [][]string) int {
	if len(tagLists) == 0 {
		return 0
	}
	lists := make([][]string, len(tagLists))
	for i, l := range tagLists {
		cp := make([]string, len(l))
		copy(cp, l)
		lists[i] = cp
	}
	satisfied := make([]bool, len(lists))

	for _, genre := range bookGenres {
		for i, list := range lists {
			if satisfied[i] {
				continue
			}
			for _, candidate := range list {
				if candidate == genre {
					satisfied[i] = true
					for j := range lists {
						filtered := lists[j][:0]
						for _, c := range lists[j] {
							if c != genre {
								filtered = append(filtered, c)
							}
						}
						lists[j] = filtered
					}
					break
				}
			}
			if satisfied[i] {
				break
			}
		}
	}

	count := 0
	for _, s := range satisfied {
		if s {
			count++
		}
	}
	return count
}

func (ps *PlayerState) runLibSearch() {
	ps.libMatches = nil
	ps.libMatchIdx = 0
	if ps.libQuery == "" {
		return
	}

	q := parseLibQuery(ps.libQuery)
	genres := allGenres(ps.lib.Books)

	tagCandidateLists := make([][]string, len(q.Tags))
	for i, tag := range q.Tags {
		tagCandidateLists[i] = buildTagCandidates(tag, genres)
	}

	type result struct {
		idx   int
		score int
	}
	var results []result

	for i, b := range ps.lib.Books {
		if q.YearLow != nil && b.Date < *q.YearLow {
			continue
		}
		if q.YearHigh != nil && b.Date > *q.YearHigh {
			continue
		}

		textScore := libTextScore(q.Text, b)
		if q.Text != "" && textScore == 0 {
			continue
		}

		tagMult := 1
		if len(q.Tags) > 0 {
			lists := make([][]string, len(tagCandidateLists))
			for j, l := range tagCandidateLists {
				cp := make([]string, len(l))
				copy(cp, l)
				lists[j] = cp
			}
			tagMult = scoreBookTags(b.Genres, lists)
			if tagMult == 0 {
				continue
			}
		}

		results = append(results, result{i, textScore * tagMult})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].idx < results[j].idx
	})

	for _, r := range results {
		ps.libMatches = append(ps.libMatches, r.idx)
	}
}

func (ps *PlayerState) libSelPos() int {
	for pos, idx := range ps.libMatches {
		if idx == ps.libSel {
			return pos
		}
	}
	return -1
}

func (ps *PlayerState) jumpToLibMatch(idx int) {
	ps.libMatchIdx = idx
	ps.libSel = ps.libMatches[idx]
	ps.libInfoOff = 0
	ps.libOffset = clampOffset(ps.libOffset, idx, ps.libListH(), 2, len(ps.libMatches))
}

func (ps *PlayerState) enterLibSearch() {
	ps.libSavedSel = ps.libSel
	ps.libSavedOffset = ps.libOffset
	ps.libQuery = ""
	ps.libMatches = nil
	ps.libMatchIdx = 0
	ps.libOffset = 0
	ps.mode = ModeLibSearch
}

func (ps *PlayerState) cancelLibSearch() {
	ps.libSel = ps.libSavedSel
	ps.libOffset = ps.libSavedOffset
	ps.libQuery = ""
	ps.libMatches = nil
	ps.libMatchIdx = 0
	ps.mode = ModeMain
}

func (ps *PlayerState) confirmLibSearch() {
	if ps.libQuery == "" {
		ps.cancelLibSearch()
		return
	}
	ps.mode = ModeLibSearching
}

func (ps *PlayerState) exitLibSearchKeep() {
	ps.libQuery = ""
	ps.libMatches = nil
	ps.libMatchIdx = 0
	ps.mode = ModeMain
}

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
