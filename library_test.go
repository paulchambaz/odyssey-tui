package main

import (
	"os"
	"testing"
)

//  fmtHM 

func TestFmtHM_HoursAndMinutes(t *testing.T) {
	got := fmtHM((3*3600 + 5*60) * 1000)
	if got != "3h 5m" {
		t.Errorf("got %q, want %q", got, "3h 5m")
	}
}

func TestFmtHM_MinutesOnly(t *testing.T) {
	got := fmtHM(42 * 60 * 1000)
	if got != "42m" {
		t.Errorf("got %q, want %q", got, "42m")
	}
}

func TestFmtHM_Zero(t *testing.T) {
	got := fmtHM(0)
	if got != "0m" {
		t.Errorf("got %q, want %q", got, "0m")
	}
}

func TestFmtHM_LessThanOneMinute(t *testing.T) {
	got := fmtHM(45 * 1000) // 45 seconds
	if got != "0m" {
		t.Errorf("got %q, want %q", got, "0m")
	}
}

func TestFmtHM_ExactlyOneHour(t *testing.T) {
	got := fmtHM(3600 * 1000)
	if got != "1h 0m" {
		t.Errorf("got %q, want %q", got, "1h 0m")
	}
}

//  fmtClock 

func TestFmtClock_HoursMinutesSeconds(t *testing.T) {
	got := fmtClock((1*3600 + 23*60 + 45) * 1000)
	if got != "1:23:45" {
		t.Errorf("got %q, want %q", got, "1:23:45")
	}
}

func TestFmtClock_MinutesSeconds(t *testing.T) {
	got := fmtClock((7*60 + 9) * 1000)
	if got != "7:09" {
		t.Errorf("got %q, want %q", got, "7:09")
	}
}

func TestFmtClock_Zero(t *testing.T) {
	got := fmtClock(0)
	if got != "0:00" {
		t.Errorf("got %q, want %q", got, "0:00")
	}
}

func TestFmtClock_SingleDigitPad(t *testing.T) {
	got := fmtClock(5 * 1000)
	if got != "0:05" {
		t.Errorf("got %q, want %q", got, "0:05")
	}
}

func TestFmtClock_RoundsDown(t *testing.T) {
	got := fmtClock(1999) // 1.999 s
	if got != "0:01" {
		t.Errorf("got %q, want %q", got, "0:01")
	}
}

//  fmtMMSS 

func TestFmtMMSS_Normal(t *testing.T) {
	got := fmtMMSS((7*60 + 5) * 1000)
	if got != "07:05" {
		t.Errorf("got %q, want %q", got, "07:05")
	}
}

func TestFmtMMSS_OverOneHour(t *testing.T) {
	got := fmtMMSS((90*60 + 30) * 1000)
	if got != "90:30" {
		t.Errorf("got %q, want %q", got, "90:30")
	}
}

func TestFmtMMSS_Zero(t *testing.T) {
	got := fmtMMSS(0)
	if got != "00:00" {
		t.Errorf("got %q, want %q", got, "00:00")
	}
}

func TestFmtMMSS_PadsBothFields(t *testing.T) {
	got := fmtMMSS(5 * 1000)
	if got != "00:05" {
		t.Errorf("got %q, want %q", got, "00:05")
	}
}

//  progressBar 

func TestProgressBar_Zero(t *testing.T) {
	got := progressBar(0.0, 10)
	want := "░░░░░░░░░░"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProgressBar_Full(t *testing.T) {
	got := progressBar(1.0, 10)
	want := "██████████"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProgressBar_Half(t *testing.T) {
	got := progressBar(0.5, 10)
	want := "█████░░░░░"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProgressBar_OverFull(t *testing.T) {
	got := progressBar(1.5, 10)
	want := "██████████"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProgressBar_ZeroWidth(t *testing.T) {
	got := progressBar(0.5, 0)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestProgressBar_NegativeWidth(t *testing.T) {
	got := progressBar(0.5, -1)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestProgressBar_LengthInvariant(t *testing.T) {
	for _, pct := range []float64{0, 0.1, 0.5, 0.9, 1.0, 1.5} {
		got := progressBar(pct, 20)
		if len([]rune(got)) != 20 {
			t.Errorf("pct=%.1f: len=%d, want 20 (bar=%q)", pct, len([]rune(got)), got)
		}
	}
}

//  truncate 

func TestTruncate_Shorter(t *testing.T) {
	got := truncate("hello", 10)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestTruncate_Exact(t *testing.T) {
	got := truncate("hello", 5)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestTruncate_Longer(t *testing.T) {
	got := truncate("hello world", 5)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestTruncate_MultiByte(t *testing.T) {
	// é is 2 bytes but 1 rune; truncation must be rune-correct
	got := truncate("héllo", 3)
	if got != "hél" {
		t.Errorf("got %q, want %q", got, "hél")
	}
}

func TestTruncate_Empty(t *testing.T) {
	got := truncate("", 5)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestTruncate_ZeroN(t *testing.T) {
	got := truncate("hello", 0)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

//  libSortKey 

func TestLibSortKey_AllStates(t *testing.T) {
	cases := []struct {
		state DownloadState
		want  int
	}{
		{DownloadInProgress, 0},
		{DownloadPreparing, 0},
		{DownloadRemote, 1},
		{DownloadReady, 2},
	}
	for _, c := range cases {
		b := Audiobook{State: c.state}
		if got := libSortKey(b); got != c.want {
			t.Errorf("state=%d: key=%d, want %d", c.state, got, c.want)
		}
	}
}

//  loadLibrary 

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestLoadLibrary_Valid(t *testing.T) {
	path := writeYAML(t, `
playing_hash: h1
books:
  - hash: h1
    title: Book One
    author: Author One
    date: 2020
    duration: 3600000
    size: 100000000
    archive_ready: true
    download_state: ready
  - hash: h2
    title: Book Two
    author: Author Two
    date: 2021
    duration: 7200000
    size: 200000000
    archive_ready: false
    download_state: remote
`)
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lib.PlayingHash != "h1" {
		t.Errorf("PlayingHash = %q", lib.PlayingHash)
	}
	if len(lib.Books) != 2 {
		t.Fatalf("len(Books) = %d, want 2", len(lib.Books))
	}
}

func TestLoadLibrary_DownloadStateMapping(t *testing.T) {
	path := writeYAML(t, `
books:
  - hash: a
    download_state: remote
  - hash: b
    download_state: preparing
  - hash: c
    download_state: downloading
  - hash: d
    download_state: ready
  - hash: e
    download_state: unknown_value
`)
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byHash := map[string]DownloadState{}
	for _, b := range lib.Books {
		byHash[b.Hash] = b.State
	}

	cases := []struct {
		hash string
		want DownloadState
	}{
		{"a", DownloadRemote},
		{"b", DownloadPreparing},
		{"c", DownloadInProgress},
		{"d", DownloadReady},
		{"e", DownloadRemote},
	}
	for _, c := range cases {
		if got := byHash[c.hash]; got != c.want {
			t.Errorf("hash=%q: state=%v, want %v", c.hash, got, c.want)
		}
	}
}

func TestLoadLibrary_SortOrder(t *testing.T) {
	path := writeYAML(t, `
books:
  - hash: ready-one
    download_state: ready
  - hash: remote-one
    download_state: remote
  - hash: dl-one
    download_state: downloading
`)
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	if lib.Books[0].Hash != "dl-one" {
		t.Errorf("books[0] = %q, want dl-one", lib.Books[0].Hash)
	}
	if lib.Books[1].Hash != "remote-one" {
		t.Errorf("books[1] = %q, want remote-one", lib.Books[1].Hash)
	}
	if lib.Books[2].Hash != "ready-one" {
		t.Errorf("books[2] = %q, want ready-one", lib.Books[2].Hash)
	}
}

func TestLoadLibrary_SortStable(t *testing.T) {
	path := writeYAML(t, `
books:
  - hash: A
    download_state: ready
  - hash: B
    download_state: ready
  - hash: C
    download_state: remote
`)
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	// remote before ready; within each group, original order preserved
	if lib.Books[0].Hash != "C" {
		t.Errorf("books[0] = %q, want C (remote)", lib.Books[0].Hash)
	}
	if lib.Books[1].Hash != "A" {
		t.Errorf("books[1] = %q, want A (stable)", lib.Books[1].Hash)
	}
	if lib.Books[2].Hash != "B" {
		t.Errorf("books[2] = %q, want B (stable)", lib.Books[2].Hash)
	}
}

func TestLoadLibrary_FileNotFound(t *testing.T) {
	_, err := loadLibrary("/nonexistent/path/lib.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadLibrary_InvalidYAML(t *testing.T) {
	path := writeYAML(t, "not: valid: yaml: [")
	_, err := loadLibrary(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
