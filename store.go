package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

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
		Title    string `yaml:"title"`
		Path     string `yaml:"path"`
		Duration int64  `yaml:"duration"`
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
	var totalDur int64
	for i, ch := range info.Chapters {
		chapters[i] = Chapter{Title: ch.Title, Path: ch.Path, Duration: ch.Duration}
		totalDur += ch.Duration
	}
	return Audiobook{
		Title:       info.Title,
		Author:      info.Author,
		Date:        info.Date,
		Description: info.Description,
		Genres:      info.Genres,
		Chapters:    chapters,
		Duration:    totalDur,
	}, nil
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
