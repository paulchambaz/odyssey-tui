package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var mpvInstanceCount int64

//  mpvPlayer 

type mpvPlayer struct {
	cmd      *exec.Cmd
	conn     net.Conn
	reader   *bufio.Reader // persists across calls — buffered bytes must not be lost
	sockPath string
	mu       sync.Mutex // serializes all socket writes + reads
	reqID    int
}

type mpvResponse struct {
	RequestID *int            `json:"request_id"` // nil for event messages
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
}

func newMpvPlayer() (*mpvPlayer, error) {
	id := atomic.AddInt64(&mpvInstanceCount, 1)
	sockPath := fmt.Sprintf("/tmp/odyssey-tui-%d-%d.sock", os.Getpid(), id)
	cmd := exec.Command("mpv",
		"--no-video",
		"--idle=yes",
		"--pause",
		"--really-quiet",
		"--input-ipc-server="+sockPath,
	)
	if err := cmd.Start(); err != nil {
		logf("PLAYER", "spawn mpv failed: %v", err)
		return nil, fmt.Errorf("spawn mpv: %w", err)
	}
	logf("PLAYER", "mpv spawned pid=%d sock=%s", cmd.Process.Pid, sockPath)
	var (
		conn net.Conn
		err  error
	)
	for i := 0; i < 20; i++ {
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		logf("PLAYER", "connect to mpv socket failed: %v", err)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("connect to mpv socket: %w", err)
	}
	logf("PLAYER", "mpv socket connected")
	return &mpvPlayer{
		cmd:      cmd,
		conn:     conn,
		reader:   bufio.NewReader(conn),
		sockPath: sockPath,
	}, nil
}

// sendCommand sends a JSON command and reads lines until the matching
// request_id arrives. Event lines (no request_id) are skipped.
func (m *mpvPlayer) sendCommand(args []any) (mpvResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reqID++
	id := m.reqID

	payload, err := json.Marshal(map[string]any{
		"command":    args,
		"request_id": id,
	})
	if err != nil {
		return mpvResponse{}, err
	}
	payload = append(payload, '\n')

	if _, err := m.conn.Write(payload); err != nil {
		return mpvResponse{}, fmt.Errorf("write: %w", err)
	}

	for {
		line, err := m.reader.ReadString('\n')
		if err != nil {
			return mpvResponse{}, fmt.Errorf("read: %w", err)
		}
		var resp mpvResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue // skip non-JSON lines
		}
		if resp.RequestID == nil || *resp.RequestID != id {
			continue // skip event messages and mismatched responses
		}
		return resp, nil
	}
}

//  control commands 

func (m *mpvPlayer) loadFile(path string, seekMs int64) error {
	logf("PLAYER", "loadFile path=%s seekMs=%d", path, seekMs)
	var args []any
	if seekMs > 0 {
		args = []any{"loadfile", path, "replace", 0, fmt.Sprintf("start=%f", float64(seekMs)/1000.0)}
	} else {
		args = []any{"loadfile", path, "replace"}
	}
	resp, err := m.sendCommand(args)
	if err != nil {
		logf("PLAYER", "loadFile error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "loadFile mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) play() error {
	logf("PLAYER", "play")
	resp, err := m.sendCommand([]any{"set_property", "pause", false})
	if err != nil {
		logf("PLAYER", "play error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "play mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) pause() error {
	logf("PLAYER", "pause")
	resp, err := m.sendCommand([]any{"set_property", "pause", true})
	if err != nil {
		logf("PLAYER", "pause error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "pause mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) seekTo(ms int64) error {
	logf("PLAYER", "seekTo ms=%d", ms)
	resp, err := m.sendCommand([]any{"seek", float64(ms) / 1000.0, "absolute"})
	if err != nil {
		logf("PLAYER", "seekTo error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "seekTo mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) seekRelative(ms int64) error {
	logf("PLAYER", "seekRelative ms=%d", ms)
	resp, err := m.sendCommand([]any{"seek", float64(ms) / 1000.0, "relative"})
	if err != nil {
		logf("PLAYER", "seekRelative error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "seekRelative mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) setSpeed(s float64) error {
	logf("PLAYER", "setSpeed %.2f", s)
	resp, err := m.sendCommand([]any{"set_property", "speed", s})
	if err != nil {
		logf("PLAYER", "setSpeed error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "setSpeed mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) setVolume(v float64) error {
	logf("PLAYER", "setVolume %.2f", v)
	resp, err := m.sendCommand([]any{"set_property", "volume", v * 100})
	if err != nil {
		logf("PLAYER", "setVolume error: %v", err)
		return err
	}
	if resp.Error != "success" {
		logf("PLAYER", "setVolume mpv error: %s", resp.Error)
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

//  property getters 

func (m *mpvPlayer) getPositionMs() (int64, error) {
	resp, err := m.sendCommand([]any{"get_property", "time-pos"})
	if err != nil {
		return 0, err
	}
	if resp.Error == "property unavailable" {
		return 0, nil
	}
	if resp.Error != "success" {
		return 0, fmt.Errorf("mpv: %s", resp.Error)
	}
	var secs float64
	if err := json.Unmarshal(resp.Data, &secs); err != nil {
		return 0, err
	}
	return int64(secs * 1000), nil
}

func (m *mpvPlayer) getDurationMs() (int64, error) {
	resp, err := m.sendCommand([]any{"get_property", "duration"})
	if err != nil {
		return 0, err
	}
	if resp.Error == "property unavailable" {
		return 0, nil
	}
	if resp.Error != "success" {
		return 0, fmt.Errorf("mpv: %s", resp.Error)
	}
	var secs float64
	if err := json.Unmarshal(resp.Data, &secs); err != nil {
		return 0, err
	}
	return int64(secs * 1000), nil
}

func (m *mpvPlayer) isEnded() (bool, error) {
	// idle-active stays true after file finishes (unlike eof-reached which resets)
	resp, err := m.sendCommand([]any{"get_property", "idle-active"})
	if err != nil {
		return false, err
	}
	if resp.Error == "property unavailable" {
		return false, nil
	}
	if resp.Error != "success" {
		return false, fmt.Errorf("mpv: %s", resp.Error)
	}
	var v bool
	if err := json.Unmarshal(resp.Data, &v); err != nil {
		return false, err
	}
	return v, nil
}

func (m *mpvPlayer) quit() {
	logf("PLAYER", "quit")
	_, _ = m.sendCommand([]any{"quit"})
	_ = m.conn.Close()
	_ = m.cmd.Wait()
	_ = os.Remove(m.sockPath)
}

//  tea integration 

func queryPositionCmd(mpv *mpvPlayer) tea.Cmd {
	return func() tea.Msg {
		pos, err := mpv.getPositionMs()
		if err != nil {
			return posQueryMsg{err: err}
		}
		dur, err := mpv.getDurationMs()
		if err != nil {
			return posQueryMsg{err: err}
		}
		ended, err := mpv.isEnded()
		if err != nil {
			return posQueryMsg{err: err}
		}
		return posQueryMsg{posMs: pos, durMs: dur, ended: ended}
	}
}
