package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ApiClient is the interface satisfied by IliadApi, allowing mocks in tests.
type ApiClient interface {
	Login(baseURL, user, pass string) (string, error)
	Register(baseURL, user, pass string) (string, error)
	GetAudiobooks() ([]Audiobook, error)
	GetAudiobook(hash string) (Audiobook, error)
	DownloadAudiobook(hash, dest string, startByte int64, onProgress func(received, total int64)) error
	CancelDownload()
	GetPosition(hash string) (Position, error)
	PutPosition(hash string, pos Position) error
}

var _ ApiClient = (*IliadApi)(nil) // compile-time satisfaction check

type Credentials struct {
	BaseURL  string
	Username string
	Password string
	Token    string
}

type AuthError struct{}

func (e *AuthError) Error() string { return "authentication failed" }

type ArchiveNotReadyError struct{}

func (e *ArchiveNotReadyError) Error() string { return "archive not ready" }

type IliadApi struct {
	creds      Credentials
	httpClient *http.Client
	cancelDL   context.CancelFunc
}

func NewIliadApi(creds Credentials) *IliadApi {
	return &IliadApi{
		creds:      creds,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *IliadApi) Login(baseURL, user, pass string) (string, error) {
	logf("API", "Login url=%s user=%s", baseURL, user)
	payload, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	req, err := http.NewRequest("POST", baseURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		logf("API", "Login error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		logf("API", "Login -> 401 auth error")
		return "", &AuthError{}
	}
	if resp.StatusCode != 200 {
		logf("API", "Login -> HTTP %d", resp.StatusCode)
		return "", fmt.Errorf("login failed: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	logf("API", "Login -> ok")
	return result.Token, nil
}

func (a *IliadApi) Register(baseURL, user, pass string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	req, err := http.NewRequest("POST", baseURL+"/auth/register", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", &AuthError{}
	}
	if resp.StatusCode == 409 {
		return "", fmt.Errorf("username already taken")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("registration failed: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (a *IliadApi) GetAudiobooks() ([]Audiobook, error) {
	logf("API", "GetAudiobooks")
	req, err := http.NewRequest("GET", a.creds.BaseURL+"/audiobooks", nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.execute(req)
	if err != nil {
		logf("API", "GetAudiobooks error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var books []Audiobook
	if err := json.NewDecoder(resp.Body).Decode(&books); err != nil {
		return nil, err
	}
	for i := range books {
		books[i].Duration *= 1000
	}
	logf("API", "GetAudiobooks -> %d books", len(books))
	return books, nil
}

func (a *IliadApi) GetAudiobook(hash string) (Audiobook, error) {
	logf("API", "GetAudiobook hash=%s", hash)
	req, err := http.NewRequest("GET", a.creds.BaseURL+"/audiobooks/"+hash, nil)
	if err != nil {
		return Audiobook{}, err
	}

	resp, err := a.execute(req)
	if err != nil {
		logf("API", "GetAudiobook error hash=%s: %v", hash, err)
		return Audiobook{}, err
	}
	defer resp.Body.Close()

	var book Audiobook
	if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
		return Audiobook{}, err
	}
	logf("API", "GetAudiobook -> title=%q archiveReady=%v", book.Title, book.ArchiveReady)
	return book, nil
}

// DownloadAudiobook streams hash's archive to dest, resuming from startByte if > 0.
// onProgress is called after each chunk with (bytesReceived, totalBytes); total is -1 if unknown.
func (a *IliadApi) DownloadAudiobook(hash, dest string, startByte int64, onProgress func(received, total int64)) error {
	logf("API", "DownloadAudiobook hash=%s dest=%s resume=%d", hash, dest, startByte)
	ctx, cancel := context.WithCancel(context.Background())
	if a.cancelDL != nil {
		a.cancelDL()
	}
	a.cancelDL = cancel

	req, err := http.NewRequestWithContext(ctx, "GET", a.creds.BaseURL+"/audiobooks/"+hash+"/download", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.creds.Token)
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// no timeout: streaming download may take a long time
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		return &ArchiveNotReadyError{}
	}
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	if totalSize >= 0 {
		totalSize += startByte
	}

	flags := os.O_CREATE | os.O_WRONLY
	if startByte > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(dest, flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	received := startByte
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			received += int64(n)
			if onProgress != nil {
				onProgress(received, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *IliadApi) CancelDownload() {
	if a.cancelDL != nil {
		a.cancelDL()
		a.cancelDL = nil
	}
}

func (a *IliadApi) GetPosition(hash string) (Position, error) {
	logf("API", "GetPosition hash=%s", hash)
	req, err := http.NewRequest("GET", a.creds.BaseURL+"/positions/"+hash, nil)
	if err != nil {
		return Position{}, err
	}

	resp, err := a.execute(req)
	if err != nil {
		logf("API", "GetPosition error hash=%s: %v", hash, err)
		return Position{}, err
	}
	defer resp.Body.Close()

	var pos Position
	if err := json.NewDecoder(resp.Body).Decode(&pos); err != nil {
		return Position{}, err
	}
	logf("API", "GetPosition -> ch=%d pos=%dms ts=%d", pos.ChapterIndex, pos.ChapterPosition, pos.Timestamp)
	return pos, nil
}

func (a *IliadApi) PutPosition(hash string, pos Position) error {
	logf("API", "PutPosition hash=%s ch=%d pos=%dms", hash, pos.ChapterIndex, pos.ChapterPosition)
	data, err := json.Marshal(pos)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", a.creds.BaseURL+"/positions/"+hash, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	resp, err := a.execute(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// execute attaches the Bearer token and retries once on 401 after re-auth.
// Returns AuthError if the retry also fails with 401.
func (a *IliadApi) execute(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+a.creds.Token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 401 {
		return resp, nil
	}
	resp.Body.Close()

	if err := a.reauth(); err != nil {
		return nil, &AuthError{}
	}

	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		req.Body = body
	}
	req.Header.Set("Authorization", "Bearer "+a.creds.Token)

	resp, err = a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 401 {
		logf("API", "execute second 401 -> AuthError")
		resp.Body.Close()
		return nil, &AuthError{}
	}
	return resp, nil
}

func (a *IliadApi) reauth() error {
	logf("API", "reauth user=%s", a.creds.Username)
	token, err := a.Login(a.creds.BaseURL, a.creds.Username, a.creds.Password)
	if err != nil {
		logf("API", "reauth failed: %v", err)
		return err
	}
	logf("API", "reauth -> token updated")
	a.creds.Token = token
	return nil
}
