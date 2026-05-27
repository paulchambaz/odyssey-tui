package main

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

//  helpers 

func requireMpv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("mpv"); err != nil {
		t.Skip("mpv not in PATH")
	}
}

func makeWAV(t *testing.T, secs int) string {
	t.Helper()
	const rate = 44100
	numSamples := rate * secs
	dataSize := numSamples * 2
	path := filepath.Join(t.TempDir(), "test.wav")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create WAV: %v", err)
	}
	defer f.Close()
	le := binary.LittleEndian
	f.Write([]byte("RIFF"))
	binary.Write(f, le, uint32(36+dataSize))
	f.Write([]byte("WAVE"))
	f.Write([]byte("fmt "))
	binary.Write(f, le, uint32(16))
	binary.Write(f, le, uint16(1))     // PCM
	binary.Write(f, le, uint16(1))     // mono
	binary.Write(f, le, uint32(rate))  // sample rate
	binary.Write(f, le, uint32(rate*2)) // byte rate
	binary.Write(f, le, uint16(2))
	binary.Write(f, le, uint16(16))
	f.Write([]byte("data"))
	binary.Write(f, le, uint32(dataSize))
	f.Write(make([]byte, dataSize))
	return path
}

func spawnPlayer(t *testing.T) *mpvPlayer {
	t.Helper()
	p, err := newMpvPlayer()
	if err != nil {
		t.Fatalf("newMpvPlayer: %v", err)
	}
	t.Cleanup(p.quit)
	return p
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// waitUntilStable polls getPositionMs until two consecutive reads are within
// 20ms — i.e. position is no longer advancing (pause took effect).
func waitUntilStable(t *testing.T, p *mpvPlayer) int64 {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	prev := int64(-1)
	for time.Now().Before(deadline) {
		time.Sleep(30 * time.Millisecond)
		pos, err := p.getPositionMs()
		if err != nil {
			t.Fatalf("getPositionMs: %v", err)
		}
		if prev >= 0 && absInt64(pos-prev) < 20 {
			return pos
		}
		prev = pos
	}
	t.Fatal("position did not stabilize within 1s after pause")
	return 0
}

//  TestNewMpvPlayer_SpawnsAndConnects 

func TestNewMpvPlayer_SpawnsAndConnects(t *testing.T) {
	requireMpv(t)
	p, err := newMpvPlayer()
	if err != nil {
		t.Fatalf("newMpvPlayer: %v", err)
	}
	if _, err := os.Stat(p.sockPath); err != nil {
		t.Errorf("socket %s not found: %v", p.sockPath, err)
	}
	p.quit()
}

//  TestLoadFile_LoadsWithoutError 

func TestLoadFile_LoadsWithoutError(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	dur, err := p.getDurationMs()
	if err != nil {
		t.Fatalf("getDurationMs: %v", err)
	}
	if dur <= 0 {
		t.Errorf("duration %d ms: want > 0", dur)
	}
}

//  TestLoadFile_WithSeek 

func TestLoadFile_WithSeek(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 2000); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	pos, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	if absInt64(pos-2000) > 500 {
		t.Errorf("position %d ms: want within 500 ms of 2000", pos)
	}
}

//  TestPlay_AdvancesPosition 

func TestPlay_AdvancesPosition(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(700 * time.Millisecond)
	pos, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	if pos < 100 {
		t.Errorf("position %d ms after 700 ms of play: want >= 100", pos)
	}
}

//  TestPause_StopsAdvancement 

func TestPause_StopsAdvancement(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	if err := p.pause(); err != nil {
		t.Fatalf("pause: %v", err)
	}
	pos1 := waitUntilStable(t, p)
	time.Sleep(400 * time.Millisecond)
	pos2, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	if absInt64(pos2-pos1) > 100 {
		t.Errorf("position moved %d ms while paused (pos1=%d pos2=%d): want < 100", absInt64(pos2-pos1), pos1, pos2)
	}
}

//  TestSeekTo_LandsWithin500ms 

func TestSeekTo_LandsWithin500ms(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := p.seekTo(2000); err != nil {
		t.Fatalf("seekTo: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	pos, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	if absInt64(pos-2000) > 500 {
		t.Errorf("seekTo(2000): landed at %d ms, want within 500 ms of 2000", pos)
	}
}

//  TestSetSpeed_DoublesAdvancementRate 

func TestSetSpeed_DoublesAdvancementRate(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	if err := p.setSpeed(2.0); err != nil {
		t.Fatalf("setSpeed: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // let speed change stabilize
	pos1, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	time.Sleep(1000 * time.Millisecond)
	pos2, err := p.getPositionMs()
	if err != nil {
		t.Fatalf("getPositionMs: %v", err)
	}
	elapsed := pos2 - pos1
	if elapsed < 1200 || elapsed > 2800 {
		t.Errorf("at 2x speed over 1 s wall clock: advanced %d ms, want 1200–2800", elapsed)
	}
}

//  TestSetVolume_DoesNotError 

func TestSetVolume_DoesNotError(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	for _, v := range []float64{0.0, 0.5, 1.0} {
		if err := p.setVolume(v); err != nil {
			t.Errorf("setVolume(%v): %v", v, err)
		}
	}
}

//  TestGetDurationMs_AfterLoad 

func TestGetDurationMs_AfterLoad(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	dur, err := p.getDurationMs()
	if err != nil {
		t.Fatalf("getDurationMs: %v", err)
	}
	if dur < 5500 || dur > 6500 {
		t.Errorf("duration %d ms: want 5500–6500 for a 6 s file", dur)
	}
}

//  TestIsEnded_FalseWhilePlaying 

func TestIsEnded_FalseWhilePlaying(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	ended, err := p.isEnded()
	if err != nil {
		t.Fatalf("isEnded: %v", err)
	}
	if ended {
		t.Error("isEnded = true while playing mid-file: want false")
	}
}

//  TestIsEnded_TrueAfterFileFinishes 

func TestIsEnded_TrueAfterFileFinishes(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 1)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(2500 * time.Millisecond)
	ended, err := p.isEnded()
	if err != nil {
		t.Fatalf("isEnded: %v", err)
	}
	if !ended {
		t.Error("isEnded = false after 1 s file finished: want true")
	}
}

//  TestQueryPositionCmd_ReturnsMsg 

func TestQueryPositionCmd_ReturnsMsg(t *testing.T) {
	requireMpv(t)
	p := spawnPlayer(t)
	wav := makeWAV(t, 6)
	if err := p.loadFile(wav, 0); err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if err := p.play(); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	cmd := queryPositionCmd(p)
	raw := cmd()
	msg, ok := raw.(posQueryMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want posQueryMsg", raw)
	}
	if msg.err != nil {
		t.Fatalf("posQueryMsg.err = %v", msg.err)
	}
	if msg.posMs < 0 {
		t.Errorf("posMs %d: want >= 0", msg.posMs)
	}
	if msg.durMs <= 0 {
		t.Errorf("durMs %d: want > 0", msg.durMs)
	}
	if msg.ended {
		t.Error("ended = true mid-file: want false")
	}
}

//  TestQuit_RemovesSockFile 

func TestQuit_RemovesSockFile(t *testing.T) {
	requireMpv(t)
	p, err := newMpvPlayer()
	if err != nil {
		t.Fatalf("newMpvPlayer: %v", err)
	}
	sockPath := p.sockPath
	p.quit()
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Errorf("socket %s still exists after quit", sockPath)
	}
}
