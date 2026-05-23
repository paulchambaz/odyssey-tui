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
		return nil, fmt.Errorf("spawn mpv: %w", err)
	}
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
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("connect to mpv socket: %w", err)
	}
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
	resp, err := m.sendCommand([]any{"loadfile", path, "replace"})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	if seekMs > 0 {
		// Poll until duration is available (file demuxed), then seek.
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
			if dur, err := m.getDurationMs(); err == nil && dur > 0 {
				break
			}
		}
		_ = m.seekTo(seekMs)
	}
	return nil
}

func (m *mpvPlayer) play() error {
	resp, err := m.sendCommand([]any{"set_property", "pause", false})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) pause() error {
	resp, err := m.sendCommand([]any{"set_property", "pause", true})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) seekTo(ms int64) error {
	resp, err := m.sendCommand([]any{"seek", float64(ms) / 1000.0, "absolute"})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) setSpeed(s float64) error {
	resp, err := m.sendCommand([]any{"set_property", "speed", s})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
		return fmt.Errorf("mpv: %s", resp.Error)
	}
	return nil
}

func (m *mpvPlayer) setVolume(v float64) error {
	resp, err := m.sendCommand([]any{"set_property", "volume", v * 100})
	if err != nil {
		return err
	}
	if resp.Error != "success" {
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
