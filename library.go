package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type DownloadState int

const (
	DownloadRemote     DownloadState = iota
	DownloadPreparing
	DownloadInProgress
	DownloadReady
)

type Chapter struct {
	Title    string `yaml:"title"    json:"title"`
	Path     string `yaml:"path"     json:"path"`
	Duration int64  `yaml:"duration" json:"duration"`
}

type Position struct {
	ChapterIndex    int   `yaml:"chapter_index"    json:"chapter_index"`
	ChapterPosition int64 `yaml:"chapter_position" json:"chapter_position"`
	Timestamp       int64 `yaml:"timestamp"        json:"timestamp"`
}

type Audiobook struct {
	Hash         string    `yaml:"hash"              json:"hash"`
	Title        string    `yaml:"title"             json:"title"`
	Author       string    `yaml:"author"            json:"author"`
	Date         int       `yaml:"date"              json:"date"`
	Description  string    `yaml:"description"       json:"description"`
	Genres       []string  `yaml:"genres"            json:"genres"`
	Duration     int64     `yaml:"duration"          json:"duration"`
	Size         int64     `yaml:"size"              json:"size"`
	ArchiveReady bool      `yaml:"archive_ready"     json:"archive_ready"`

	// local-only: not from API
	DownloadStateRaw string    `yaml:"download_state"    json:"-"`
	DownloadProgress float64   `yaml:"download_progress" json:"-"`
	Position         *Position `yaml:"position"          json:"-"`
	Chapters         []Chapter `yaml:"chapters"          json:"-"`
	State            DownloadState
}

type Library struct {
	PlayingHash string      `yaml:"playing_hash"`
	Books       []Audiobook `yaml:"books"`
}

func loadLibrary(path string) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lib Library
	if err := yaml.Unmarshal(data, &lib); err != nil {
		return nil, err
	}
	for i := range lib.Books {
		switch lib.Books[i].DownloadStateRaw {
		case "preparing":
			lib.Books[i].State = DownloadPreparing
		case "downloading":
			lib.Books[i].State = DownloadInProgress
		case "ready":
			lib.Books[i].State = DownloadReady
		default:
			lib.Books[i].State = DownloadRemote
		}
	}
	sort.SliceStable(lib.Books, func(a, b int) bool {
		return libSortKey(lib.Books[a]) < libSortKey(lib.Books[b])
	})
	return &lib, nil
}

func libSortKey(b Audiobook) int {
	switch b.State {
	case DownloadInProgress, DownloadPreparing:
		return 0
	case DownloadReady:
		return 2
	default:
		return 1
	}
}

// fmtHM formats milliseconds as "4h 3m" or "42m".
func fmtHM(ms int64) string {
	total := ms / 1000
	m := (total / 60) % 60
	h := total / 3600
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// fmtClock formats milliseconds as "H:MM:SS" or "M:SS".
func fmtClock(ms int64) string {
	total := ms / 1000
	s := total % 60
	m := (total / 60) % 60
	h := total / 3600
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// fmtMMSS formats milliseconds as "MM:SS", rolling minutes past 59 if needed.
func fmtMMSS(ms int64) string {
	total := ms / 1000
	s := total % 60
	m := total / 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// progressBar renders filled/empty block characters for a 0.0-1.0 value.
func progressBar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// truncate shortens s to at most n runes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
