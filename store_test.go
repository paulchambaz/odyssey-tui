package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

//  helpers 

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := &Store{dir: dir, dataDir: dir}
	os.MkdirAll(filepath.Join(dir, "library"), 0755)
	return s
}

func writeInfoYml(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

//  NewStore 

func TestNewStore_CreatesDirectories(t *testing.T) {
	base := t.TempDir()
	cfgDir := filepath.Join(base, "config", "odyssey-tui")
	dataDir := filepath.Join(base, "data", "odyssey-tui")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))

	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}

	for _, dir := range []string{cfgDir, dataDir, filepath.Join(dataDir, "library")} {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("expected dir %s to exist: %v", dir, err)
		}
	}
}

//  Credentials 

func TestSaveLoadCredentials_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	want := Credentials{
		BaseURL:  "http://localhost:9090",
		Username: "alice",
		Password: "secret",
		Token:    "tok123",
	}
	if err := s.SaveCredentials(want); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	got := s.LoadCredentials()
	if got == nil {
		t.Fatal("LoadCredentials returned nil")
	}
	if *got != want {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestLoadCredentials_NilWhenFileAbsent(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadCredentials(); got != nil {
		t.Errorf("expected nil, got %+v", *got)
	}
}

func TestLoadCredentials_NilWhenTokenMissing(t *testing.T) {
	s := newTestStore(t)
	// save without token
	cfg := storeConfig{BaseURL: "http://x", Username: "u", Password: "p"}
	if err := saveTOML(filepath.Join(s.dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadCredentials(); got != nil {
		t.Errorf("expected nil with empty token, got %+v", *got)
	}
}

func TestLoadCredentials_NilWhenBaseURLMissing(t *testing.T) {
	s := newTestStore(t)
	cfg := storeConfig{Username: "u", Password: "p", Token: "t"}
	if err := saveTOML(filepath.Join(s.dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadCredentials(); got != nil {
		t.Errorf("expected nil with empty base_url, got %+v", *got)
	}
}

//  Positions (local) 

func TestSaveLoadPosition_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	want := Position{ChapterIndex: 2, ChapterPosition: 45000, Timestamp: 1714521600}
	if err := s.SavePosition("hash1", want); err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	got := s.LoadPosition("hash1")
	if got == nil {
		t.Fatal("LoadPosition returned nil")
	}
	if *got != want {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestLoadPosition_NilForUnknownHash(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadPosition("nope"); got != nil {
		t.Errorf("expected nil, got %+v", *got)
	}
}

func TestSavePosition_MultipleHashes(t *testing.T) {
	s := newTestStore(t)
	p1 := Position{ChapterIndex: 0, ChapterPosition: 1000, Timestamp: 1}
	p2 := Position{ChapterIndex: 3, ChapterPosition: 5000, Timestamp: 2}
	s.SavePosition("a", p1)
	s.SavePosition("b", p2)
	if got := s.LoadPosition("a"); got == nil || *got != p1 {
		t.Errorf("hash a: got %v, want %+v", got, p1)
	}
	if got := s.LoadPosition("b"); got == nil || *got != p2 {
		t.Errorf("hash b: got %v, want %+v", got, p2)
	}
}

//  Positions (server) 

func TestSaveLoadServerPosition_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	want := Position{ChapterIndex: 1, ChapterPosition: 9000, Timestamp: 999}
	if err := s.SaveServerPosition("hash2", want); err != nil {
		t.Fatalf("SaveServerPosition: %v", err)
	}
	got := s.LoadServerPosition("hash2")
	if got == nil {
		t.Fatal("LoadServerPosition returned nil")
	}
	if *got != want {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestLoadServerPosition_NilForUnknownHash(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadServerPosition("nope"); got != nil {
		t.Errorf("expected nil, got %+v", *got)
	}
}

func TestServerPositions_SeparateFromLocalPositions(t *testing.T) {
	s := newTestStore(t)
	local := Position{ChapterIndex: 0, Timestamp: 1}
	server := Position{ChapterIndex: 5, Timestamp: 2}
	s.SavePosition("h", local)
	s.SaveServerPosition("h", server)
	if got := s.LoadPosition("h"); got == nil || *got != local {
		t.Errorf("local: got %v, want %+v", got, local)
	}
	if got := s.LoadServerPosition("h"); got == nil || *got != server {
		t.Errorf("server: got %v, want %+v", got, server)
	}
}

//  Download states 

func TestSaveLoadDownloadState_AllStates(t *testing.T) {
	cases := []struct {
		raw  string
		want DownloadState
	}{
		{"remote", DownloadRemote},
		{"preparing", DownloadPreparing},
		{"downloading", DownloadInProgress},
		{"ready", DownloadReady},
	}
	for _, tc := range cases {
		s := newTestStore(t)
		if err := s.SaveDownloadState("h", tc.raw); err != nil {
			t.Fatalf("SaveDownloadState(%q): %v", tc.raw, err)
		}
		if got := s.LoadDownloadState("h"); got != tc.want {
			t.Errorf("raw=%q: got %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestLoadDownloadState_RemoteForUnknownHash(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadDownloadState("nope"); got != DownloadRemote {
		t.Errorf("got %d, want DownloadRemote", got)
	}
}

//  ClearAll 

func TestClearAll_WipesPositionFiles(t *testing.T) {
	s := newTestStore(t)
	pos := Position{ChapterIndex: 1, Timestamp: 100}
	s.SavePosition("h", pos)
	s.SaveServerPosition("h", pos)
	s.SaveDownloadState("h", "ready")

	if err := s.ClearAll(); err != nil {
		t.Fatalf("ClearAll: %v", err)
	}

	if got := s.LoadPosition("h"); got != nil {
		t.Errorf("LoadPosition after ClearAll: expected nil, got %+v", *got)
	}
	if got := s.LoadServerPosition("h"); got != nil {
		t.Errorf("LoadServerPosition after ClearAll: expected nil, got %+v", *got)
	}
	if got := s.LoadDownloadState("h"); got != DownloadRemote {
		t.Errorf("LoadDownloadState after ClearAll: got %d, want DownloadRemote", got)
	}
}

func TestClearAll_IdempotentOnEmpty(t *testing.T) {
	s := newTestStore(t)
	if err := s.ClearAll(); err != nil {
		t.Fatalf("ClearAll on empty store: %v", err)
	}
}

//  Settings 

func TestSaveSetting_Float_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveSetting("playback_speed", 1.5); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	if got := s.LoadFloat("playback_speed", 1.0); got != 1.5 {
		t.Errorf("got %v, want 1.5", got)
	}
}

func TestSaveSetting_Int_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveSetting("rewind_on_resume", 5); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	if got := s.LoadInt("rewind_on_resume", 0); got != 5 {
		t.Errorf("got %v, want 5", got)
	}
}

func TestSaveSetting_Bool_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveSetting("volume_normalization", true); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	if got := s.LoadBool("volume_normalization", false); !got {
		t.Error("got false, want true")
	}
}

func TestSaveSetting_String_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	want := "/home/user/audiobooks"
	if err := s.SaveSetting("download_location", want); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	if got := s.LoadString("download_location", ""); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadSetting_DefaultWhenAbsent(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadFloat("playback_speed", 1.0); got != 1.0 {
		t.Errorf("float default: got %v, want 1.0", got)
	}
	if got := s.LoadInt("rewind_on_resume", 3); got != 3 {
		t.Errorf("int default: got %v, want 3", got)
	}
	if got := s.LoadBool("volume_normalization", true); !got {
		t.Error("bool default: got false, want true")
	}
	if got := s.LoadString("download_location", "/default"); got != "/default" {
		t.Errorf("string default: got %q, want %q", got, "/default")
	}
}

//  LibraryDir 

func TestLibraryDir_DefaultPath(t *testing.T) {
	s := newTestStore(t)
	want := filepath.Join(s.dataDir, "library", "abc123")
	if got := s.LibraryDir("abc123"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLibraryDir_CustomPath(t *testing.T) {
	s := newTestStore(t)
	s.SaveSetting("download_location", "/tmp/mybooks")
	want := "/tmp/mybooks/abc123"
	if got := s.LibraryDir("abc123"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

//  parseInfoYml 

const minimalInfoYml = `
title: "Dune"
author: "Frank Herbert"
date: 1965
description: "A sci-fi epic."
genres:
  - "Science Fiction"
chapters:
  - title: "Part One"
    path: "01-part-one.opus"
  - title: "Part Two"
    path: "02-part-two.opus"
`

func TestParseInfoYml_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "info.yml")
	writeInfoYml(t, path, minimalInfoYml)

	book, err := parseInfoYml(path)
	if err != nil {
		t.Fatalf("parseInfoYml: %v", err)
	}
	if book.Title != "Dune" {
		t.Errorf("Title = %q, want %q", book.Title, "Dune")
	}
	if book.Author != "Frank Herbert" {
		t.Errorf("Author = %q", book.Author)
	}
	if book.Date != 1965 {
		t.Errorf("Date = %d, want 1965", book.Date)
	}
	if book.Description != "A sci-fi epic." {
		t.Errorf("Description = %q", book.Description)
	}
	if len(book.Genres) != 1 || book.Genres[0] != "Science Fiction" {
		t.Errorf("Genres = %v", book.Genres)
	}
	if len(book.Chapters) != 2 {
		t.Errorf("len(Chapters) = %d, want 2", len(book.Chapters))
	}
	if book.Chapters[0].Title != "Part One" {
		t.Errorf("Chapters[0].Title = %q", book.Chapters[0].Title)
	}
	if book.Chapters[0].Path != "01-part-one.opus" {
		t.Errorf("Chapters[0].Path = %q", book.Chapters[0].Path)
	}
}

func TestParseInfoYml_WithDurations(t *testing.T) {
	// info.yml stores duration in seconds; parseInfoYml must convert to ms internally.
	dir := t.TempDir()
	path := filepath.Join(dir, "info.yml")
	writeInfoYml(t, path, `
title: "Book"
author: "Author"
date: 2000
chapters:
  - title: "Ch1"
    path: "01.opus"
    duration: 120
  - title: "Ch2"
    path: "02.opus"
    duration: 80
`)
	book, err := parseInfoYml(path)
	if err != nil {
		t.Fatalf("parseInfoYml: %v", err)
	}
	if book.Chapters[0].Duration != 120000 {
		t.Errorf("Ch1 duration = %d, want 120000 ms", book.Chapters[0].Duration)
	}
	if book.Chapters[1].Duration != 80000 {
		t.Errorf("Ch2 duration = %d, want 80000 ms", book.Chapters[1].Duration)
	}
	if book.Duration != 200000 {
		t.Errorf("total Duration = %d, want 200000 ms", book.Duration)
	}
}

func TestParseInfoYml_MissingFile(t *testing.T) {
	_, err := parseInfoYml("/nonexistent/path/info.yml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestParseInfoYml_MalformedYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "info.yml")
	writeInfoYml(t, path, "title: [unclosed bracket")
	_, err := parseInfoYml(path)
	if err == nil {
		t.Error("expected error for malformed yaml, got nil")
	}
}

//  LoadLocalAudiobooks 

func TestLoadLocalAudiobooks_EmptyDir(t *testing.T) {
	s := newTestStore(t)
	books, counts, times, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
	if len(counts) != 0 {
		t.Errorf("len(counts) = %d, want 0", len(counts))
	}
	if len(times) != 0 {
		t.Errorf("len(times) = %d, want 0", len(times))
	}
}

func TestLoadLocalAudiobooks_ParsesBook(t *testing.T) {
	s := newTestStore(t)
	hash := "abc123"
	bookDir := filepath.Join(s.dataDir, "library", hash)
	writeInfoYml(t, filepath.Join(bookDir, "info.yml"), minimalInfoYml)

	books, _, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("len(books) = %d, want 1", len(books))
	}
	b := books[0]
	if b.Hash != hash {
		t.Errorf("Hash = %q, want %q", b.Hash, hash)
	}
	if b.Title != "Dune" {
		t.Errorf("Title = %q, want Dune", b.Title)
	}
	if b.State != DownloadReady {
		t.Errorf("State = %d, want DownloadReady", b.State)
	}
}

func TestLoadLocalAudiobooks_ChapterCount(t *testing.T) {
	s := newTestStore(t)
	hash := "xyz"
	bookDir := filepath.Join(s.dataDir, "library", hash)
	writeInfoYml(t, filepath.Join(bookDir, "info.yml"), minimalInfoYml)

	_, counts, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if counts[hash] != 2 {
		t.Errorf("counts[%q] = %d, want 2", hash, counts[hash])
	}
}

func TestLoadLocalAudiobooks_DownloadTime(t *testing.T) {
	s := newTestStore(t)
	hash := "timehash"
	bookDir := filepath.Join(s.dataDir, "library", hash)
	writeInfoYml(t, filepath.Join(bookDir, "info.yml"), minimalInfoYml)

	_, _, times, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if _, ok := times[hash]; !ok {
		t.Errorf("times[%q] missing", hash)
	}
	if times[hash] <= 0 {
		t.Errorf("times[%q] = %d, want > 0", hash, times[hash])
	}
}

func TestLoadLocalAudiobooks_SkipsNonDir(t *testing.T) {
	s := newTestStore(t)
	base := filepath.Join(s.dataDir, "library")
	os.WriteFile(filepath.Join(base, "strayfile.txt"), []byte("x"), 0644)

	books, _, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}
}

func TestLoadLocalAudiobooks_SkipsMissingInfoYml(t *testing.T) {
	s := newTestStore(t)
	base := filepath.Join(s.dataDir, "library")
	os.MkdirAll(filepath.Join(base, "emptyhash"), 0755)

	books, _, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}
}

//  makeTestOpus / opusDurationMs

// makeTestOpus writes a minimal syntactically-valid Ogg Opus file whose
// duration is exactly durationMs milliseconds. CRC fields are zeroed (the
// parser does not verify checksums).
func makeTestOpus(t *testing.T, durationMs int64) string {
	t.Helper()
	const preSkip = 312
	const sampleRate = 48000
	granule := durationMs*sampleRate/1000 + preSkip

	var buf []byte

	writePage := func(headerType byte, gran int64, seq uint32, data []byte) {
		// Segment table: split data into 255-byte runs
		var lace []byte
		rem := len(data)
		for rem >= 255 {
			lace = append(lace, 255)
			rem -= 255
		}
		lace = append(lace, byte(rem))

		buf = append(buf, 'O', 'g', 'g', 'S')
		buf = append(buf, 0) // version
		buf = append(buf, headerType)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(gran))
		buf = binary.LittleEndian.AppendUint32(buf, 1)       // serial
		buf = binary.LittleEndian.AppendUint32(buf, seq)     // sequence
		buf = binary.LittleEndian.AppendUint32(buf, 0)       // CRC (ignored)
		buf = append(buf, byte(len(lace)))
		buf = append(buf, lace...)
		buf = append(buf, data...)
	}

	// Page 0: BOS with OpusHead (19 bytes of data)
	head := make([]byte, 19)
	copy(head[0:8], "OpusHead")
	head[8] = 1 // version
	head[9] = 2 // channels
	binary.LittleEndian.PutUint16(head[10:12], preSkip)
	binary.LittleEndian.PutUint32(head[12:16], sampleRate)
	writePage(2, 0, 0, head)

	// Page 1: OpusTags (minimal comment header, 16 bytes)
	tags := make([]byte, 16)
	copy(tags[0:8], "OpusTags")
	binary.LittleEndian.PutUint32(tags[8:12], 4)
	copy(tags[12:16], "test")
	writePage(0, 0, 1, tags)

	// Page 2: EOS with final granule
	writePage(4, granule, 2, []byte{0xf8, 0xff, 0xfe})

	path := filepath.Join(t.TempDir(), "test.opus")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.opus: %v", err)
	}
	return path
}

func TestOpusDurationMs_Basic(t *testing.T) {
	want := int64(120000) // 2 minutes
	path := makeTestOpus(t, want)
	got, err := opusDurationMs(path)
	if err != nil {
		t.Fatalf("opusDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestOpusDurationMs_Short(t *testing.T) {
	want := int64(5000) // 5 seconds
	path := makeTestOpus(t, want)
	got, err := opusDurationMs(path)
	if err != nil {
		t.Fatalf("opusDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestOpusDurationMs_MissingFile(t *testing.T) {
	_, err := opusDurationMs("/nonexistent/file.opus")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  SaveBookDurations / LoadBookDurations

func TestSaveLoadBookDurations(t *testing.T) {
	s := newTestStore(t)
	hash := "abc"
	os.MkdirAll(s.LibraryDir(hash), 0755)
	want := []int64{60000, 120000, 90000}
	if err := s.SaveBookDurations(hash, want); err != nil {
		t.Fatalf("SaveBookDurations: %v", err)
	}
	got := s.LoadBookDurations(hash)
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("durations[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestLoadBookDurations_Missing(t *testing.T) {
	s := newTestStore(t)
	got := s.LoadBookDurations("nohash")
	if got != nil {
		t.Errorf("expected nil for missing durations, got %v", got)
	}
}

//  LoadLocalAudiobooks with probed durations

func TestLoadLocalAudiobooks_ProbesDurations(t *testing.T) {
	s := newTestStore(t)
	hash := "probe1"
	bookDir := filepath.Join(s.dataDir, "library", hash)
	os.MkdirAll(bookDir, 0755)

	ch1 := makeTestOpus(t, 60000)
	ch2 := makeTestOpus(t, 90000)

	infoYml := `
title: "Test Book"
author: "Author"
date: 2000
chapters:
  - title: "Ch1"
    path: "ch1.opus"
  - title: "Ch2"
    path: "ch2.opus"
`
	writeInfoYml(t, filepath.Join(bookDir, "info.yml"), infoYml)
	os.Link(ch1, filepath.Join(bookDir, "ch1.opus"))
	os.Link(ch2, filepath.Join(bookDir, "ch2.opus"))

	books, _, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("len(books) = %d, want 1", len(books))
	}
	b := books[0]
	if b.Chapters[0].Duration != 60000 {
		t.Errorf("ch0 duration = %d, want 60000", b.Chapters[0].Duration)
	}
	if b.Chapters[1].Duration != 90000 {
		t.Errorf("ch1 duration = %d, want 90000", b.Chapters[1].Duration)
	}
	if b.Duration != 150000 {
		t.Errorf("total duration = %d, want 150000", b.Duration)
	}
}

func TestLoadLocalAudiobooks_UsesCachedDurations(t *testing.T) {
	s := newTestStore(t)
	hash := "cache1"
	bookDir := filepath.Join(s.dataDir, "library", hash)
	os.MkdirAll(bookDir, 0755)

	infoYml := `
title: "Test Book"
author: "Author"
date: 2000
chapters:
  - title: "Ch1"
    path: "ch1.opus"
`
	writeInfoYml(t, filepath.Join(bookDir, "info.yml"), infoYml)
	// Pre-save durations (as if already probed) — no audio file present.
	s.SaveBookDurations(hash, []int64{75000})

	books, _, _, err := s.LoadLocalAudiobooks()
	if err != nil {
		t.Fatalf("LoadLocalAudiobooks: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("len(books) = %d, want 1", len(books))
	}
	if books[0].Chapters[0].Duration != 75000 {
		t.Errorf("duration = %d, want 75000 (from cache, no probe)", books[0].Chapters[0].Duration)
	}
}
