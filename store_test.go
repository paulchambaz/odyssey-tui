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

func TestParseInfoYml_DurationIgnored(t *testing.T) {
	// info.yml duration fields are ignored; durations come from probing.
	dir := t.TempDir()
	path := filepath.Join(dir, "info.yml")
	writeInfoYml(t, path, `
title: "Book"
author: "Author"
date: 2000
chapters:
  - title: "Ch1"
    path: "01.opus"
  - title: "Ch2"
    path: "02.opus"
`)
	book, err := parseInfoYml(path)
	if err != nil {
		t.Fatalf("parseInfoYml: %v", err)
	}
	if book.Chapters[0].Duration != 0 {
		t.Errorf("Ch1 duration = %d, want 0 (probed separately)", book.Chapters[0].Duration)
	}
	if book.Chapters[1].Duration != 0 {
		t.Errorf("Ch2 duration = %d, want 0 (probed separately)", book.Chapters[1].Duration)
	}
	if book.Duration != 0 {
		t.Errorf("total Duration = %d, want 0 (probed separately)", book.Duration)
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

//  makeTestMP3 / mp3DurationMs

// makeTestMP3 writes a minimal CBR MPEG1 Layer III file whose duration is
// exactly durationMs milliseconds (within 1 ms). No actual audio data.
func makeTestMP3(t *testing.T, durationMs int64) string {
	t.Helper()
	// MPEG1, Layer III, 128 kbps, 44100 Hz, stereo.
	const bitrate = 128000
	const sampleRate = 44100
	const samplesPerFrame = 1152
	frameSize := 144 * bitrate / sampleRate // 417 bytes
	frames := durationMs * int64(sampleRate) / (int64(samplesPerFrame) * 1000)
	if frames == 0 {
		frames = 1
	}

	// Frame header: sync + MPEG1 Layer3 128kbps 44100Hz stereo.
	frameHdr := []byte{0xff, 0xfb, 0x90, 0x00}
	frame := make([]byte, frameSize)
	copy(frame, frameHdr)

	buf := make([]byte, 0, int(frames)*frameSize)
	for i := int64(0); i < frames; i++ {
		buf = append(buf, frame...)
	}

	path := filepath.Join(t.TempDir(), "test.mp3")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.mp3: %v", err)
	}
	return path
}

func TestMP3DurationMs_CBR(t *testing.T) {
	want := int64(10000) // 10 seconds
	path := makeTestMP3(t, want)
	got, err := mp3DurationMs(path)
	if err != nil {
		t.Fatalf("mp3DurationMs: %v", err)
	}
	// CBR estimation can be off by a frame; allow ±100 ms tolerance.
	if got < want-100 || got > want+100 {
		t.Errorf("duration = %d ms, want ~%d ms", got, want)
	}
}

func TestMP3DurationMs_MissingFile(t *testing.T) {
	_, err := mp3DurationMs("/nonexistent/file.mp3")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  makeTestM4A / m4aDurationMs

// makeTestM4A writes a minimal MPEG-4 file containing only ftyp + moov/mvhd.
func makeTestM4A(t *testing.T, durationMs int64) string {
	t.Helper()
	const timescale = 1000

	be32 := func(v uint32) []byte {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, v)
		return b
	}
	box := func(typ string, payload []byte) []byte {
		size := uint32(8 + len(payload))
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, size)
		b = append(b, []byte(typ)...)
		b = append(b, payload...)
		return b
	}

	// mvhd version 0: 4 flags + 4 ctime + 4 mtime + 4 timescale + 4 duration + 76 rest
	mvhdPayload := make([]byte, 96)
	copy(mvhdPayload[12:16], be32(timescale))
	copy(mvhdPayload[16:20], be32(uint32(durationMs)))
	// matrix identity + pre-defined zeros already zero
	mvhdPayload[95] = 0 // next track id byte (simplified)

	mvhd := box("mvhd", mvhdPayload)
	moov := box("moov", mvhd)
	ftyp := box("ftyp", []byte("M4A \x00\x00\x00\x00"))

	buf := append(ftyp, moov...)
	path := filepath.Join(t.TempDir(), "test.m4a")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.m4a: %v", err)
	}
	return path
}

func TestM4ADurationMs_Basic(t *testing.T) {
	want := int64(90000) // 90 seconds
	path := makeTestM4A(t, want)
	got, err := m4aDurationMs(path)
	if err != nil {
		t.Fatalf("m4aDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestM4ADurationMs_M4B(t *testing.T) {
	want := int64(3600000) // 1 hour
	path := makeTestM4A(t, want)
	// Rename to .m4b to verify dispatch works for that extension too.
	m4b := path[:len(path)-4] + ".m4b"
	if err := os.Rename(path, m4b); err != nil {
		t.Fatalf("rename: %v", err)
	}
	got, err := m4aDurationMs(m4b)
	if err != nil {
		t.Fatalf("m4aDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestM4ADurationMs_MissingFile(t *testing.T) {
	_, err := m4aDurationMs("/nonexistent/file.m4a")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  makeTestFLAC / flacDurationMs

// makeTestFLAC writes a minimal FLAC file with a STREAMINFO block only.
func makeTestFLAC(t *testing.T, durationMs int64) string {
	t.Helper()
	const sampleRate = 44100
	totalSamples := durationMs * int64(sampleRate) / 1000

	// STREAMINFO is 34 bytes. Bit layout from byte 10:
	//   20 bits sample rate | 3 bits channels-1 | 5 bits bps-1 | 36 bits total samples
	info := make([]byte, 34)
	// min/max block size (bytes 0-3): 4096
	binary.BigEndian.PutUint16(info[0:2], 4096)
	binary.BigEndian.PutUint16(info[2:4], 4096)
	// min/max frame size (bytes 4-9): 0 = unknown
	// Pack from byte 10:
	//   sampleRate=44100 (0xAC44) in bits [0..19]
	//   channels=1 (0) in bits [20..22]
	//   bps=16 (15) in bits [23..27]
	//   totalSamples in bits [28..63]
	v := uint64(sampleRate)<<44 | uint64(0)<<41 | uint64(15)<<36 | uint64(totalSamples)
	for i := 0; i < 8; i++ {
		info[10+i] = byte(v >> (56 - 8*i))
	}

	// Block header: type=0 (STREAMINFO), last-metadata=1, length=34
	hdr := []byte{0x80, 0x00, 0x00, 0x22}

	buf := append([]byte("fLaC"), hdr...)
	buf = append(buf, info...)

	path := filepath.Join(t.TempDir(), "test.flac")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.flac: %v", err)
	}
	return path
}

func TestFlacDurationMs_Basic(t *testing.T) {
	want := int64(120000) // 2 minutes
	path := makeTestFLAC(t, want)
	got, err := flacDurationMs(path)
	if err != nil {
		t.Fatalf("flacDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestFlacDurationMs_MissingFile(t *testing.T) {
	_, err := flacDurationMs("/nonexistent/file.flac")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  makeTestOggVorbis / oggVorbisDurationMs

// makeTestOggVorbis writes a minimal Ogg Vorbis file.
func makeTestOggVorbis(t *testing.T, durationMs int64) string {
	t.Helper()
	const sampleRate = 44100
	granule := durationMs * int64(sampleRate) / 1000

	var buf []byte
	writePage := func(headerType byte, gran int64, seq uint32, data []byte) {
		var lace []byte
		rem := len(data)
		for rem >= 255 {
			lace = append(lace, 255)
			rem -= 255
		}
		lace = append(lace, byte(rem))
		buf = append(buf, 'O', 'g', 'g', 'S', 0, headerType)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(gran))
		buf = binary.LittleEndian.AppendUint32(buf, 1)
		buf = binary.LittleEndian.AppendUint32(buf, seq)
		buf = binary.LittleEndian.AppendUint32(buf, 0)
		buf = append(buf, byte(len(lace)))
		buf = append(buf, lace...)
		buf = append(buf, data...)
	}

	// Vorbis identification header (30 bytes minimum).
	ident := make([]byte, 30)
	ident[0] = 0x01
	copy(ident[1:7], "vorbis")
	binary.LittleEndian.PutUint32(ident[11:15], uint32(sampleRate))
	writePage(2, 0, 0, ident)

	// Comment + setup pages (minimal).
	comment := make([]byte, 16)
	comment[0] = 0x03
	copy(comment[1:7], "vorbis")
	writePage(0, 0, 1, comment)

	// EOS page with final granule.
	writePage(4, granule, 2, []byte{0x00})

	path := filepath.Join(t.TempDir(), "test.ogg")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.ogg: %v", err)
	}
	return path
}

func TestOggVorbisDurationMs_Basic(t *testing.T) {
	want := int64(60000) // 1 minute
	path := makeTestOggVorbis(t, want)
	got, err := oggVorbisDurationMs(path)
	if err != nil {
		t.Fatalf("oggVorbisDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestOggVorbisDurationMs_MissingFile(t *testing.T) {
	_, err := oggVorbisDurationMs("/nonexistent/file.ogg")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  makeTestWAV / wavDurationMs

// makeTestWAV writes a minimal PCM WAV file with no audio data.
func makeTestWAV(t *testing.T, durationMs int64) string {
	t.Helper()
	const sampleRate = 44100
	const channels = 2
	const bitsPerSample = 16
	byteRate := int64(sampleRate * channels * bitsPerSample / 8)
	dataSize := byteRate * durationMs / 1000

	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:24], channels)
	binary.LittleEndian.PutUint32(buf[24:28], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:34], channels*bitsPerSample/8) // block align
	binary.LittleEndian.PutUint16(buf[34:36], bitsPerSample)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))

	path := filepath.Join(t.TempDir(), "test.wav")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("write test.wav: %v", err)
	}
	return path
}

func TestWAVDurationMs_Basic(t *testing.T) {
	want := int64(30000) // 30 seconds
	path := makeTestWAV(t, want)
	got, err := wavDurationMs(path)
	if err != nil {
		t.Fatalf("wavDurationMs: %v", err)
	}
	if got != want {
		t.Errorf("duration = %d ms, want %d ms", got, want)
	}
}

func TestWAVDurationMs_MissingFile(t *testing.T) {
	_, err := wavDurationMs("/nonexistent/file.wav")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

//  audioDurationMs dispatch

func TestAudioDurationMs_Dispatch(t *testing.T) {
	cases := []struct {
		ext  string
		make func(*testing.T, int64) string
	}{
		{".opus", makeTestOpus},
		{".mp3", makeTestMP3},
		{".m4a", makeTestM4A},
		{".flac", makeTestFLAC},
		{".ogg", makeTestOggVorbis},
		{".wav", makeTestWAV},
	}
	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			want := int64(10000)
			path := tc.make(t, want)
			got, err := audioDurationMs(path)
			if err != nil {
				t.Fatalf("audioDurationMs(%s): %v", tc.ext, err)
			}
			if got < want-200 || got > want+200 {
				t.Errorf("duration = %d ms, want ~%d ms", got, want)
			}
		})
	}
}

func TestAudioDurationMs_UnsupportedExt(t *testing.T) {
	_, err := audioDurationMs("file.aac")
	if err == nil {
		t.Error("expected error for unsupported extension, got nil")
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

//  ServerCatalog

func TestSaveLoadServerCatalog_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	books := []Audiobook{
		{Hash: "h1", Title: "Book One", Author: "Author A", Date: 2020, Duration: 3600000, Size: 100000, Genres: []string{"sci-fi"}},
		{Hash: "h2", Title: "Book Two", Author: "Author B", Date: 2021, Duration: 7200000, Size: 200000},
		{Hash: "h3", Title: "Book Three", Author: "Author C", Date: 2022, Duration: 1800000, Size: 50000, Description: "A great book"},
	}
	if err := s.SaveServerCatalog(books); err != nil {
		t.Fatalf("SaveServerCatalog: %v", err)
	}
	got := s.LoadServerCatalog()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, want := range books {
		g := got[i]
		if g.Hash != want.Hash {
			t.Errorf("[%d] Hash = %q, want %q", i, g.Hash, want.Hash)
		}
		if g.Title != want.Title {
			t.Errorf("[%d] Title = %q, want %q", i, g.Title, want.Title)
		}
		if g.Author != want.Author {
			t.Errorf("[%d] Author = %q, want %q", i, g.Author, want.Author)
		}
		if g.Duration != want.Duration {
			t.Errorf("[%d] Duration = %d, want %d", i, g.Duration, want.Duration)
		}
		if g.State != DownloadRemote {
			t.Errorf("[%d] State = %v, want DownloadRemote", i, g.State)
		}
		if len(g.Chapters) != 0 {
			t.Errorf("[%d] Chapters = %v, want empty (local-only field)", i, g.Chapters)
		}
	}
}

func TestLoadServerCatalog_NilWhenAbsent(t *testing.T) {
	s := newTestStore(t)
	if got := s.LoadServerCatalog(); got != nil {
		t.Errorf("expected nil, got %d books", len(got))
	}
}

func TestSaveServerCatalog_WritesFile(t *testing.T) {
	s := newTestStore(t)
	books := []Audiobook{{Hash: "h1", Title: "Book"}}
	if err := s.SaveServerCatalog(books); err != nil {
		t.Fatalf("SaveServerCatalog: %v", err)
	}
	catalogPath := filepath.Join(s.dataDir, "server_catalog.json")
	if _, err := os.Stat(catalogPath); err != nil {
		t.Errorf("server_catalog.json not found: %v", err)
	}
	tmpPath := catalogPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("tmp file should not exist after atomic write")
	}
}

func TestSaveServerCatalog_EmptySlice(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveServerCatalog([]Audiobook{}); err != nil {
		t.Fatalf("SaveServerCatalog: %v", err)
	}
	if got := s.LoadServerCatalog(); got != nil {
		t.Errorf("expected nil for empty catalog, got %d books", len(got))
	}
}
