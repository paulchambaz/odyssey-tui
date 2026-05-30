package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type storeConfig struct {
	BaseURL          string  `toml:"base_url"`
	Username         string  `toml:"username"`
	Password         string  `toml:"password"`
	Token            string  `toml:"token"`
	DownloadLocation string  `toml:"download_location"`
	RewindOnResume   int     `toml:"rewind_on_resume"`
	PlaybackSpeed    float64 `toml:"playback_speed"`
	Volume           int     `toml:"volume,omitempty"`
	VolumeNorm       *bool   `toml:"volume_normalization,omitempty"`
}

type Store struct {
	dir        string
	dataDir    string
	configFile string // if set, overrides dir/config.toml
}

func NewStore() (*Store, error) {
	cfgBase := os.Getenv("XDG_CONFIG_HOME")
	if cfgBase == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfgBase = filepath.Join(home, ".config")
	}
	dataBase := os.Getenv("XDG_DATA_HOME")
	if dataBase == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dataBase = filepath.Join(home, ".local", "share")
	}

	s := &Store{
		dir:     filepath.Join(cfgBase, "odyssey-tui"),
		dataDir: filepath.Join(dataBase, "odyssey-tui"),
	}
	for _, dir := range []string{s.dir, s.dataDir, filepath.Join(s.dataDir, "library")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) LogPath() string {
	return filepath.Join(s.dataDir, "odyssey-tui.log")
}

//  low-level helpers

func saveJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func saveTOML(path string, v any) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadTOML(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, v)
}

func (s *Store) configPath() string {
	if s.configFile != "" {
		return s.configFile
	}
	return filepath.Join(s.dir, "config.toml")
}

// NewStoreAt uses cfgFile as the config file instead of the XDG default.
// Positions and library data still go to XDG data dirs.
func NewStoreAt(cfgFile string) (*Store, error) {
	s, err := NewStore()
	if err != nil {
		return nil, err
	}
	s.configFile = cfgFile
	return s, nil
}

func (s *Store) loadConfig() storeConfig {
	var cfg storeConfig
	loadTOML(s.configPath(), &cfg)
	return cfg
}

func (s *Store) saveConfig(cfg storeConfig) error {
	return saveTOML(s.configPath(), cfg)
}

//  credentials 

func (s *Store) SaveCredentials(c Credentials) error {
	cfg := s.loadConfig()
	cfg.BaseURL = c.BaseURL
	cfg.Username = c.Username
	cfg.Password = c.Password
	cfg.Token = c.Token
	return s.saveConfig(cfg)
}

func (s *Store) LoadCredentials() *Credentials {
	cfg := s.loadConfig()
	if cfg.BaseURL == "" || cfg.Username == "" || cfg.Password == "" || cfg.Token == "" {
		return nil
	}
	return &Credentials{
		BaseURL:  cfg.BaseURL,
		Username: cfg.Username,
		Password: cfg.Password,
		Token:    cfg.Token,
	}
}

//  positions 

func (s *Store) positionsPath() string {
	return filepath.Join(s.dir, "positions.json")
}

func (s *Store) serverPositionsPath() string {
	return filepath.Join(s.dir, "server_positions.json")
}

func (s *Store) SavePosition(hash string, pos Position) error {
	m := map[string]Position{}
	loadJSON(s.positionsPath(), &m)
	m[hash] = pos
	return saveJSON(s.positionsPath(), m)
}

func (s *Store) LoadPosition(hash string) *Position {
	m := map[string]Position{}
	loadJSON(s.positionsPath(), &m)
	p, ok := m[hash]
	if !ok {
		return nil
	}
	return &p
}

func (s *Store) SaveServerPosition(hash string, pos Position) error {
	m := map[string]Position{}
	loadJSON(s.serverPositionsPath(), &m)
	m[hash] = pos
	return saveJSON(s.serverPositionsPath(), m)
}

func (s *Store) LoadServerPosition(hash string) *Position {
	m := map[string]Position{}
	loadJSON(s.serverPositionsPath(), &m)
	p, ok := m[hash]
	if !ok {
		return nil
	}
	return &p
}

//  download states 

func (s *Store) downloadsPath() string {
	return filepath.Join(s.dir, "downloads.json")
}

func (s *Store) SaveDownloadState(hash, state string) error {
	m := map[string]string{}
	loadJSON(s.downloadsPath(), &m)
	m[hash] = state
	return saveJSON(s.downloadsPath(), m)
}

func (s *Store) LoadDownloadState(hash string) DownloadState {
	m := map[string]string{}
	loadJSON(s.downloadsPath(), &m)
	switch m[hash] {
	case "preparing":
		return DownloadPreparing
	case "downloading":
		return DownloadInProgress
	case "ready":
		return DownloadReady
	default:
		return DownloadRemote
	}
}

//  clear credentials 

func (s *Store) ClearCredentials() error {
	cfg := s.loadConfig()
	cfg.BaseURL = ""
	cfg.Username = ""
	cfg.Password = ""
	cfg.Token = ""
	return s.saveConfig(cfg)
}

//  clear all 

func (s *Store) ClearAll() error {
	for _, path := range []string{s.positionsPath(), s.serverPositionsPath(), s.downloadsPath()} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

//  settings 

func (s *Store) LoadFloat(key string, def float64) float64 {
	cfg := s.loadConfig()
	switch key {
	case "playback_speed":
		if cfg.PlaybackSpeed != 0 {
			return cfg.PlaybackSpeed
		}
	}
	return def
}

func (s *Store) LoadInt(key string, def int) int {
	cfg := s.loadConfig()
	switch key {
	case "rewind_on_resume":
		if cfg.RewindOnResume != 0 {
			return cfg.RewindOnResume
		}
	case "volume":
		if cfg.Volume != 0 {
			return cfg.Volume
		}
	}
	return def
}

func (s *Store) LoadBool(key string, def bool) bool {
	cfg := s.loadConfig()
	switch key {
	case "volume_normalization":
		if cfg.VolumeNorm != nil {
			return *cfg.VolumeNorm
		}
	}
	return def
}

func (s *Store) LoadString(key string, def string) string {
	cfg := s.loadConfig()
	switch key {
	case "download_location":
		if cfg.DownloadLocation != "" {
			return cfg.DownloadLocation
		}
	}
	return def
}

func (s *Store) SaveSetting(key string, val any) error {
	cfg := s.loadConfig()
	switch key {
	case "playback_speed":
		cfg.PlaybackSpeed = val.(float64)
	case "rewind_on_resume":
		cfg.RewindOnResume = val.(int)
	case "volume":
		cfg.Volume = val.(int)
	case "volume_normalization":
		v := val.(bool)
		cfg.VolumeNorm = &v
	case "download_location":
		cfg.DownloadLocation = val.(string)
	}
	return s.saveConfig(cfg)
}

//  library dir 

func (s *Store) LibraryDir(hash string) string {
	base := s.LoadString("download_location", filepath.Join(s.dataDir, "library"))
	return filepath.Join(base, hash)
}

//  info.yml parsing 

type infoYml struct {
	Title       string   `yaml:"title"`
	Author      string   `yaml:"author"`
	Date        int      `yaml:"date"`
	Description string   `yaml:"description"`
	Genres      []string `yaml:"genres"`
	Chapters    []struct {
		Title string `yaml:"title"`
		Path  string `yaml:"path"`
	} `yaml:"chapters"`
}

func parseInfoYml(path string) (Audiobook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Audiobook{}, err
	}
	var info infoYml
	if err := yaml.Unmarshal(data, &info); err != nil {
		return Audiobook{}, err
	}
	chapters := make([]Chapter, len(info.Chapters))
	for i, ch := range info.Chapters {
		chapters[i] = Chapter{Title: ch.Title, Path: ch.Path}
	}
	return Audiobook{
		Title:       info.Title,
		Author:      info.Author,
		Date:        info.Date,
		Description: info.Description,
		Genres:      info.Genres,
		Chapters:    chapters,
	}, nil
}

//  local books cache

type cachedBook struct {
	Hash        string    `json:"hash"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Date        int       `json:"date"`
	Description string    `json:"description"`
	Genres      []string  `json:"genres"`
	Duration    int64     `json:"duration"`
	Size        int64     `json:"size"`
	Chapters    []Chapter `json:"chapters"`
}

func (s *Store) localBooksCachePath() string {
	return filepath.Join(s.dataDir, "books_cache.json")
}

func (s *Store) SaveLocalBooksCache(books []Audiobook, times map[string]int64) error {
	cache := make([]cachedBook, len(books))
	for i, b := range books {
		cache[i] = cachedBook{
			Hash:        b.Hash,
			Title:       b.Title,
			Author:      b.Author,
			Date:        b.Date,
			Description: b.Description,
			Genres:      b.Genres,
			Duration:    b.Duration,
			Size:        b.Size,
			Chapters:    b.Chapters,
		}
	}
	return saveJSON(s.localBooksCachePath(), cache)
}

func (s *Store) LoadLocalBooksCache() ([]Audiobook, map[string]int64) {
	var cache []cachedBook
	if err := loadJSON(s.localBooksCachePath(), &cache); err != nil || len(cache) == 0 {
		return nil, nil
	}
	books := make([]Audiobook, len(cache))
	for i, c := range cache {
		books[i] = Audiobook{
			Hash:        c.Hash,
			Title:       c.Title,
			Author:      c.Author,
			Date:        c.Date,
			Description: c.Description,
			Genres:      c.Genres,
			Duration:    c.Duration,
			Size:        c.Size,
			Chapters:    c.Chapters,
			State:       DownloadReady,
		}
	}
	return books, nil
}

//  server catalog cache

func (s *Store) serverCatalogPath() string {
	return filepath.Join(s.dataDir, "server_catalog.json")
}

func (s *Store) SaveServerCatalog(books []Audiobook) error {
	cache := make([]cachedBook, len(books))
	for i, b := range books {
		cache[i] = cachedBook{
			Hash:        b.Hash,
			Title:       b.Title,
			Author:      b.Author,
			Date:        b.Date,
			Description: b.Description,
			Genres:      b.Genres,
			Duration:    b.Duration,
			Size:        b.Size,
		}
	}
	return saveJSON(s.serverCatalogPath(), cache)
}

func (s *Store) LoadServerCatalog() []Audiobook {
	var cache []cachedBook
	if err := loadJSON(s.serverCatalogPath(), &cache); err != nil || len(cache) == 0 {
		return nil
	}
	books := make([]Audiobook, len(cache))
	for i, c := range cache {
		books[i] = Audiobook{
			Hash:        c.Hash,
			Title:       c.Title,
			Author:      c.Author,
			Date:        c.Date,
			Description: c.Description,
			Genres:      c.Genres,
			Duration:    c.Duration,
			Size:        c.Size,
			State:       DownloadRemote,
		}
	}
	return books
}

//  chapter duration probing

// opusDurationMs returns the duration of an Ogg Opus file in milliseconds.
// It reads the OpusHead packet for the pre-skip value and finds the last
// granule position in the file without verifying Ogg CRC checksums.
func opusDurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Read the first 512 bytes to find OpusHead and extract pre-skip.
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	idx := bytes.Index(head, []byte("OpusHead"))
	if idx < 0 {
		return 0, fmt.Errorf("opusDurationMs: OpusHead not found in %s", path)
	}
	if idx+12 > len(head) {
		return 0, fmt.Errorf("opusDurationMs: OpusHead truncated in %s", path)
	}
	preSkip := int64(binary.LittleEndian.Uint16(head[idx+10 : idx+12]))

	// Read the tail of the file and find the last Ogg page with a valid granule.
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	const tailSize = 65536
	seekPos := info.Size() - tailSize
	if seekPos < 0 {
		seekPos = 0
	}
	if _, err := f.Seek(seekPos, io.SeekStart); err != nil {
		return 0, err
	}
	tail := make([]byte, tailSize)
	n, _ = io.ReadFull(f, tail)
	tail = tail[:n]

	// Scan backwards for "OggS", skipping pages with granule = -1 (no timestamp).
	for i := len(tail) - 4; i >= 0; i-- {
		if tail[i] != 'O' || tail[i+1] != 'g' || tail[i+2] != 'g' || tail[i+3] != 'S' {
			continue
		}
		if i+14 > len(tail) {
			continue
		}
		granule := int64(binary.LittleEndian.Uint64(tail[i+6 : i+14]))
		if granule < 0 {
			continue
		}
		samples := granule - preSkip
		if samples < 0 {
			samples = 0
		}
		return samples * 1000 / 48000, nil
	}
	return 0, fmt.Errorf("opusDurationMs: no valid Ogg page found in %s", path)
}

// mp3DurationMs returns the duration of an MP3 file in milliseconds.
// Checks for a Xing/Info VBR header in the first frame for accuracy;
// falls back to CBR estimation from the file size and bitrate.
func mp3DurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	head := make([]byte, 4096)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	// Skip ID3v2 tag if present.
	off := 0
	if len(head) >= 10 && head[0] == 'I' && head[1] == 'D' && head[2] == '3' {
		size := int(head[6])<<21 | int(head[7])<<14 | int(head[8])<<7 | int(head[9])
		off = 10 + size
	}

	// Find first sync word (0xffe0 or better).
	frameOff := -1
	for i := off; i < len(head)-3; i++ {
		if head[i] == 0xff && head[i+1]&0xe0 == 0xe0 {
			frameOff = i
			break
		}
	}
	if frameOff < 0 {
		return 0, fmt.Errorf("mp3DurationMs: no sync frame in %s", path)
	}
	h := head[frameOff:]

	// Decode MPEG layer/version/bitrate/samplerate from frame header.
	version := (h[1] >> 3) & 0x3  // 3=MPEG1, 2=MPEG2, 0=MPEG2.5
	layer := (h[1] >> 1) & 0x3    // 1=L3, 2=L2, 3=L1
	bitrateIdx := (h[2] >> 4) & 0xf
	srIdx := (h[2] >> 2) & 0x3

	bitrateTable := [4][4][16]int{
		// MPEG2.5
		{{}, {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0}},
		{}, // reserved
		// MPEG2
		{{}, {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0}},
		// MPEG1
		{{}, {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0}},
	}
	srTable := [4][4]int{
		{11025, 12000, 8000, 0},  // MPEG2.5
		{},                       // reserved
		{22050, 24000, 16000, 0}, // MPEG2
		{44100, 48000, 32000, 0}, // MPEG1
	}
	bitrate := bitrateTable[version][layer][bitrateIdx] * 1000 // bps
	sampleRate := srTable[version][srIdx]
	if bitrate == 0 || sampleRate == 0 {
		return 0, fmt.Errorf("mp3DurationMs: invalid header in %s", path)
	}

	// Xing/Info VBR header: 36 bytes into frame for MPEG1 stereo (most common).
	// Offsets vary by version/channel mode; search within first 256 bytes of frame.
	xingOff := -1
	for _, tag := range [][]byte{[]byte("Xing"), []byte("Info")} {
		if idx := bytes.Index(h[:min(256, len(h))], tag); idx >= 0 {
			xingOff = idx
			break
		}
	}
	if xingOff >= 0 && xingOff+120 <= len(h) {
		flags := binary.BigEndian.Uint32(h[xingOff+4:])
		if flags&0x1 != 0 && flags&0x4 != 0 {
			// Both frames count and byte count present.
			frames := int64(binary.BigEndian.Uint32(h[xingOff+8:]))
			samplesPerFrame := int64(1152)
			if layer == 1 {
				samplesPerFrame = 576
			}
			return frames * samplesPerFrame * 1000 / int64(sampleRate), nil
		}
	}

	// CBR fallback: estimate from file size minus ID3 tags.
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	audioBytes := fi.Size() - int64(frameOff)
	return audioBytes * 8 * 1000 / int64(bitrate), nil
}

// m4aDurationMs returns the duration of an M4A/M4B file in milliseconds.
// Parses the MPEG-4 box tree to find the mvhd atom which contains
// the movie timescale and duration.
func m4aDurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}

	var walk func(limit int64) (int64, error)
	walk = func(limit int64) (int64, error) {
		var buf [8]byte
		for {
			pos, _ := f.Seek(0, io.SeekCurrent)
			if pos >= limit {
				return 0, nil
			}
			if _, err := io.ReadFull(f, buf[:8]); err != nil {
				return 0, nil
			}
			boxSize := int64(binary.BigEndian.Uint32(buf[:4]))
			boxType := string(buf[4:8])
			if boxSize == 1 {
				// Extended 64-bit size.
				var ext [8]byte
				if _, err := io.ReadFull(f, ext[:]); err != nil {
					return 0, nil
				}
				boxSize = int64(binary.BigEndian.Uint64(ext[:]))
			}
			if boxSize < 8 {
				return 0, fmt.Errorf("m4aDurationMs: invalid box size in %s", path)
			}
			end := pos + boxSize
			switch boxType {
			case "moov", "trak", "mdia", "minf", "stbl":
				if ms, err := walk(end); ms > 0 || err != nil {
					return ms, err
				}
			case "mvhd":
				var hdr [4]byte
				if _, err := io.ReadFull(f, hdr[:]); err != nil {
					return 0, nil
				}
				version := hdr[0]
				if version == 1 {
					var d [28]byte
					if _, err := io.ReadFull(f, d[:]); err != nil {
						return 0, nil
					}
					ts := int64(binary.BigEndian.Uint32(d[16:20]))
					dur := int64(binary.BigEndian.Uint64(d[20:28]))
					if ts == 0 {
						return 0, nil
					}
					return dur * 1000 / ts, nil
				}
				var d [16]byte
				if _, err := io.ReadFull(f, d[:]); err != nil {
					return 0, nil
				}
				ts := int64(binary.BigEndian.Uint32(d[8:12]))
				dur := int64(binary.BigEndian.Uint32(d[12:16]))
				if ts == 0 {
					return 0, nil
				}
				return dur * 1000 / ts, nil
			}
			if _, err := f.Seek(end, io.SeekStart); err != nil {
				return 0, nil
			}
		}
	}

	ms, err := walk(fi.Size())
	if err != nil {
		return 0, err
	}
	if ms == 0 {
		return 0, fmt.Errorf("m4aDurationMs: mvhd not found in %s", path)
	}
	return ms, nil
}

// flacDurationMs returns the duration of a FLAC file in milliseconds.
// Reads the STREAMINFO metadata block which contains sample rate and total samples.
func flacDurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var sig [4]byte
	if _, err := io.ReadFull(f, sig[:]); err != nil || string(sig[:]) != "fLaC" {
		return 0, fmt.Errorf("flacDurationMs: not a FLAC file: %s", path)
	}

	// First metadata block must be STREAMINFO.
	var hdr [4]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return 0, fmt.Errorf("flacDurationMs: truncated metadata in %s", path)
	}
	blockType := hdr[0] & 0x7f
	blockLen := int(hdr[1])<<16 | int(hdr[2])<<8 | int(hdr[3])
	if blockType != 0 || blockLen < 18 {
		return 0, fmt.Errorf("flacDurationMs: STREAMINFO not first block in %s", path)
	}

	data := make([]byte, blockLen)
	if _, err := io.ReadFull(f, data); err != nil {
		return 0, fmt.Errorf("flacDurationMs: truncated STREAMINFO in %s", path)
	}

	// Bit layout of STREAMINFO (starting at byte 10):
	//   20 bits: sample rate
	//    3 bits: channels - 1
	//    5 bits: bits per sample - 1
	//   36 bits: total samples
	sampleRate := int64(data[10])<<12 | int64(data[11])<<4 | int64(data[12])>>4
	totalSamples := int64(data[13]&0xf)<<32 | int64(data[14])<<24 | int64(data[15])<<16 | int64(data[16])<<8 | int64(data[17])
	if sampleRate == 0 {
		return 0, fmt.Errorf("flacDurationMs: zero sample rate in %s", path)
	}
	return totalSamples * 1000 / sampleRate, nil
}

// oggVorbisDurationMs returns the duration of an Ogg Vorbis file in milliseconds.
// Reads the vorbis identification header for the sample rate then scans the tail
// for the last granule position, same strategy as opusDurationMs.
func oggVorbisDurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	// Vorbis identification header: "\x01vorbis" magic.
	idx := bytes.Index(head, []byte("\x01vorbis"))
	if idx < 0 {
		return 0, fmt.Errorf("oggVorbisDurationMs: vorbis header not found in %s", path)
	}
	if idx+17 > len(head) {
		return 0, fmt.Errorf("oggVorbisDurationMs: vorbis header truncated in %s", path)
	}
	sampleRate := int64(binary.LittleEndian.Uint32(head[idx+11 : idx+15]))
	if sampleRate == 0 {
		return 0, fmt.Errorf("oggVorbisDurationMs: zero sample rate in %s", path)
	}

	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	const tailSize = 65536
	seekPos := fi.Size() - tailSize
	if seekPos < 0 {
		seekPos = 0
	}
	if _, err := f.Seek(seekPos, io.SeekStart); err != nil {
		return 0, err
	}
	tail := make([]byte, tailSize)
	n, _ = io.ReadFull(f, tail)
	tail = tail[:n]

	for i := len(tail) - 4; i >= 0; i-- {
		if tail[i] != 'O' || tail[i+1] != 'g' || tail[i+2] != 'g' || tail[i+3] != 'S' {
			continue
		}
		if i+14 > len(tail) {
			continue
		}
		granule := int64(binary.LittleEndian.Uint64(tail[i+6 : i+14]))
		if granule < 0 {
			continue
		}
		return granule * 1000 / sampleRate, nil
	}
	return 0, fmt.Errorf("oggVorbisDurationMs: no valid Ogg page found in %s", path)
}

// wavDurationMs returns the duration of a WAV file in milliseconds.
// Parses the RIFF/WAVE header and fmt chunk for sample rate and byte rate.
func wavDurationMs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return 0, fmt.Errorf("wavDurationMs: truncated header in %s", path)
	}
	if string(riff[:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return 0, fmt.Errorf("wavDurationMs: not a WAV file: %s", path)
	}

	// Walk chunks to find fmt and data.
	var byteRate, dataSize int64
	var hdr [8]byte
	for {
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			break
		}
		chunkID := string(hdr[:4])
		chunkSize := int64(binary.LittleEndian.Uint32(hdr[4:8]))
		switch chunkID {
		case "fmt ":
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, fmtData); err != nil {
				return 0, fmt.Errorf("wavDurationMs: truncated fmt chunk in %s", path)
			}
			if len(fmtData) < 16 {
				return 0, fmt.Errorf("wavDurationMs: fmt chunk too short in %s", path)
			}
			byteRate = int64(binary.LittleEndian.Uint32(fmtData[8:12]))
		case "data":
			dataSize = chunkSize
			if byteRate > 0 {
				return dataSize * 1000 / byteRate, nil
			}
			return 0, fmt.Errorf("wavDurationMs: fmt chunk missing before data in %s", path)
		default:
			if _, err := f.Seek(chunkSize, io.SeekCurrent); err != nil {
				return 0, nil
			}
		}
	}
	return 0, fmt.Errorf("wavDurationMs: data chunk not found in %s", path)
}

// audioDurationMs dispatches to the correct prober based on file extension.
func audioDurationMs(path string) (int64, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".opus":
		return opusDurationMs(path)
	case ".mp3":
		return mp3DurationMs(path)
	case ".m4a", ".m4b":
		return m4aDurationMs(path)
	case ".flac":
		return flacDurationMs(path)
	case ".ogg":
		return oggVorbisDurationMs(path)
	case ".wav":
		return wavDurationMs(path)
	default:
		return 0, fmt.Errorf("audioDurationMs: unsupported format: %s", path)
	}
}

// probeChapterDurations returns durations in ms for each chapter by parsing
// the audio files. Chapters that cannot be probed get duration 0.
func probeChapterDurations(bookDir string, chapters []Chapter) []int64 {
	durations := make([]int64, len(chapters))
	for i, ch := range chapters {
		d, err := audioDurationMs(filepath.Join(bookDir, ch.Path))
		if err == nil {
			durations[i] = d
		}
	}
	return durations
}

func (s *Store) bookDurationsPath(hash string) string {
	return filepath.Join(s.LibraryDir(hash), "durations.json")
}

func (s *Store) SaveBookDurations(hash string, durations []int64) error {
	return saveJSON(s.bookDurationsPath(hash), durations)
}

func (s *Store) LoadBookDurations(hash string) []int64 {
	var d []int64
	if err := loadJSON(s.bookDurationsPath(hash), &d); err != nil {
		return nil
	}
	return d
}

//  load local audiobooks

func (s *Store) LoadLocalAudiobooks() ([]Audiobook, map[string]int, map[string]int64, error) {
	base := s.LoadString("download_location", filepath.Join(s.dataDir, "library"))
	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, map[string]int{}, map[string]int64{}, nil
		}
		return nil, nil, nil, err
	}

	var books []Audiobook
	counts := map[string]int{}
	times := map[string]int64{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hash := entry.Name()
		infoPath := filepath.Join(base, hash, "info.yml")
		if _, err := os.Stat(infoPath); err != nil {
			continue
		}
		book, err := parseInfoYml(infoPath)
		if err != nil {
			continue
		}
		book.Hash = hash
		book.State = DownloadReady

		// Attach chapter durations: use cached file, or probe audio files once.
		bookDir := filepath.Join(base, hash)
		if saved := s.LoadBookDurations(hash); len(saved) == len(book.Chapters) {
			var total int64
			for i, d := range saved {
				book.Chapters[i].Duration = d
				total += d
			}
			book.Duration = total
		} else {
			probed := probeChapterDurations(bookDir, book.Chapters)
			var total int64
			for _, d := range probed {
				total += d
			}
			if total > 0 {
				for i, d := range probed {
					book.Chapters[i].Duration = d
				}
				book.Duration = total
				s.SaveBookDurations(hash, probed)
			}
		}

		info, err := entry.Info()
		if err == nil {
			times[hash] = info.ModTime().Unix()
		}

		counts[hash] = len(book.Chapters)
		books = append(books, book)
	}

	if books == nil {
		books = []Audiobook{}
	}
	return books, counts, times, nil
}
