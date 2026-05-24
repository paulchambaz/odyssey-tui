package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

//  JSON decode 

func TestAudiobookDecode_AllFields(t *testing.T) {
	raw := `{
		"hash": "abc123",
		"title": "Foundation",
		"author": "Isaac Asimov",
		"date": 1951,
		"description": "A galactic empire falls.",
		"genres": ["science fiction"],
		"duration": 54321000,
		"size": 987654321,
		"archive_ready": true
	}`

	var b Audiobook
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		t.Fatal(err)
	}
	if b.Hash != "abc123" {
		t.Errorf("Hash = %q", b.Hash)
	}
	if b.Title != "Foundation" {
		t.Errorf("Title = %q", b.Title)
	}
	if b.Author != "Isaac Asimov" {
		t.Errorf("Author = %q", b.Author)
	}
	if b.Date != 1951 {
		t.Errorf("Date = %d", b.Date)
	}
	if b.Description != "A galactic empire falls." {
		t.Errorf("Description = %q", b.Description)
	}
	if len(b.Genres) != 1 || b.Genres[0] != "science fiction" {
		t.Errorf("Genres = %v", b.Genres)
	}
	if b.Duration != 54321000 {
		t.Errorf("Duration = %d", b.Duration)
	}
	if b.Size != 987654321 {
		t.Errorf("Size = %d", b.Size)
	}
	if !b.ArchiveReady {
		t.Error("ArchiveReady = false, want true")
	}
}

func TestAudiobookDecode_OptionalAbsent(t *testing.T) {
	raw := `{"hash":"xyz","title":"Dune","author":"Frank Herbert","date":1965,"duration":1000,"archive_ready":false}`

	var b Audiobook
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		t.Fatal(err)
	}
	if b.Description != "" {
		t.Errorf("Description = %q, want empty", b.Description)
	}
	if len(b.Genres) != 0 {
		t.Errorf("Genres = %v, want empty", b.Genres)
	}
	if b.Size != 0 {
		t.Errorf("Size = %d, want 0", b.Size)
	}
}

func TestPositionRoundtrip_WithTimestamp(t *testing.T) {
	p := Position{ChapterIndex: 3, ChapterPosition: 180000, Timestamp: 1714521600}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var got Position
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("got %+v, want %+v", got, p)
	}
}

func TestPositionRoundtrip_ZeroTimestamp(t *testing.T) {
	p := Position{}
	data, _ := json.Marshal(p)
	var got Position
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("got %+v, want %+v", got, p)
	}
}

//  401 retry 

func TestExecute_401Retry(t *testing.T) {
	loginCalls := 0
	booksCalls := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCalls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
	})
	mux.HandleFunc("/audiobooks", func(w http.ResponseWriter, r *http.Request) {
		booksCalls++
		if booksCalls == 1 {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Audiobook{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{
		BaseURL:  srv.URL,
		Username: "user",
		Password: "pass",
		Token:    "old-token",
	})

	_, err := api.GetAudiobooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginCalls != 1 {
		t.Errorf("login called %d times, want 1", loginCalls)
	}
	if booksCalls != 2 {
		t.Errorf("/audiobooks called %d times, want 2", booksCalls)
	}
}

func TestExecute_401DoubleFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
	})
	mux.HandleFunc("/audiobooks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{
		BaseURL:  srv.URL,
		Username: "user",
		Password: "pass",
		Token:    "old-token",
	})

	_, err := api.GetAudiobooks()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T: %v", err, err)
	}
}

//  Download 

func TestDownload_503(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/audiobooks/abc123/download", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	dest := t.TempDir() + "/test.tar.gz"

	err := api.DownloadAudiobook("abc123", dest, 0, nil)
	if _, ok := err.(*ArchiveNotReadyError); !ok {
		t.Errorf("expected *ArchiveNotReadyError, got %T: %v", err, err)
	}
}

func TestDownload_Resume(t *testing.T) {
	var gotRange string
	mux := http.NewServeMux()
	mux.HandleFunc("/audiobooks/abc123/download", func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.WriteHeader(206)
		w.Write([]byte("more data"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})

	// create a pre-existing partial file
	dir := t.TempDir()
	dest := dir + "/test.tar.gz"
	os.WriteFile(dest, []byte("existing"), 0644)

	err := api.DownloadAudiobook("abc123", dest, 1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotRange != "bytes=1024-" {
		t.Errorf("Range = %q, want %q", gotRange, "bytes=1024-")
	}
}

//  Login / Register 

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "mytoken"})
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	tok, err := api.Login(srv.URL, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "mytoken" {
		t.Errorf("token = %q, want %q", tok, "mytoken")
	}
}

func TestLogin_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	_, err := api.Login(srv.URL, "user", "wrong")
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T: %v", err, err)
	}
}

func TestLogin_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	_, err := api.Login(srv.URL, "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestRegister_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "regtoken"})
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	tok, err := api.Register(srv.URL, "newuser", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "regtoken" {
		t.Errorf("token = %q, want %q", tok, "regtoken")
	}
}

func TestRegister_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	_, err := api.Register(srv.URL, "user", "pass")
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T: %v", err, err)
	}
}

func TestRegister_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	_, err := api.Register(srv.URL, "taken", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "username already taken") {
		t.Errorf("error = %q, want 'username already taken'", err)
	}
}

//  GetAudiobooks 

func TestGetAudiobooks_Success(t *testing.T) {
	books := []Audiobook{
		{Hash: "h1", Title: "Book One"},
		{Hash: "h2", Title: "Book Two"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(books)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	got, err := api.GetAudiobooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Hash != "h1" {
		t.Errorf("books[0].Hash = %q, want %q", got[0].Hash, "h1")
	}
}

func TestGetAudiobooks_DurationConvertedToMs(t *testing.T) {
	// Server sends duration in seconds; GetAudiobooks must convert to ms.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"hash":"h1","title":"Book","duration":60000}]`))
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	got, err := api.GetAudiobooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 60000 seconds = 16h40m; internally should be 60000000 ms
	if got[0].Duration != 60000*1000 {
		t.Errorf("Duration = %d, want %d (seconds→ms conversion)", got[0].Duration, 60000*1000)
	}
}

func TestGetAudiobooks_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	_, err := api.GetAudiobooks()
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

//  GetAudiobook 

func TestGetAudiobook_Success(t *testing.T) {
	book := Audiobook{Hash: "abc", Title: "Foundation", Description: "Sci-fi epic", Size: 1 << 20}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	got, err := api.GetAudiobook("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hash != "abc" {
		t.Errorf("Hash = %q, want %q", got.Hash, "abc")
	}
	if got.Description != "Sci-fi epic" {
		t.Errorf("Description = %q", got.Description)
	}
}

//  GetPosition / PutPosition 

func TestGetPosition_Success(t *testing.T) {
	want := Position{ChapterIndex: 2, ChapterPosition: 5000, Timestamp: 1000}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	got, err := api.GetPosition("hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestPutPosition_Success(t *testing.T) {
	var received Position
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	sent := Position{ChapterIndex: 1, ChapterPosition: 30000, Timestamp: 9999}
	if err := api.PutPosition("hash", sent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != sent {
		t.Errorf("server received %+v, want %+v", received, sent)
	}
}

func TestPutPosition_ReauthOnRetry(t *testing.T) {
	loginCalls := 0
	putCalls := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCalls++
		json.NewEncoder(w).Encode(map[string]string{"token": "new-tok"})
	})
	mux.HandleFunc("/positions/hash", func(w http.ResponseWriter, r *http.Request) {
		putCalls++
		if putCalls == 1 {
			w.WriteHeader(401)
			return
		}
		// verify body is re-sent on retry
		var pos Position
		json.NewDecoder(r.Body).Decode(&pos)
		if pos.ChapterIndex != 3 {
			w.WriteHeader(400)
			return
		}
		w.WriteHeader(200)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Username: "u", Password: "p", Token: "old"})
	err := api.PutPosition("hash", Position{ChapterIndex: 3, Timestamp: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginCalls != 1 {
		t.Errorf("login called %d times, want 1", loginCalls)
	}
	if putCalls != 2 {
		t.Errorf("PUT called %d times, want 2", putCalls)
	}
}

//  Download (additional) 

func TestDownload_ProgressCallback(t *testing.T) {
	data := []byte("0123456789") // 10 bytes
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(200)
		w.Write(data)
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	dest := t.TempDir() + "/out.tar.gz"

	var calls []int64
	err := api.DownloadAudiobook("abc", dest, 0, func(received, total int64) {
		calls = append(calls, received)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 {
		t.Fatal("onProgress never called")
	}
	if calls[len(calls)-1] != 10 {
		t.Errorf("final received = %d, want 10", calls[len(calls)-1])
	}
}

func TestDownload_Cancel(t *testing.T) {
	started := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/audiobooks/abc/download", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(started)
		<-r.Context().Done()
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	dest := t.TempDir() + "/out.tar.gz"

	errCh := make(chan error, 1)
	go func() {
		errCh <- api.DownloadAudiobook("abc", dest, 0, nil)
	}()

	<-started
	api.CancelDownload()

	if err := <-errCh; err == nil {
		t.Error("expected error after cancel, got nil")
	}
}

func TestDownload_CreatesFileOnFreshStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	dest := t.TempDir() + "/out.tar.gz"

	if err := api.DownloadAudiobook("abc", dest, 0, nil); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(dest)
	if string(content) != "hello" {
		t.Errorf("file = %q, want %q", string(content), "hello")
	}
}

func TestDownload_AppendsOnResume(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(206)
		w.Write([]byte("more data"))
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "tok"})
	dir := t.TempDir()
	dest := dir + "/out.tar.gz"
	os.WriteFile(dest, []byte("existing"), 0644)

	if err := api.DownloadAudiobook("abc", dest, 8, nil); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(dest)
	if string(content) != "existingmore data" {
		t.Errorf("file = %q, want %q", string(content), "existingmore data")
	}
}

func TestCancelDownload_Noop(t *testing.T) {
	api := NewIliadApi(Credentials{})
	api.CancelDownload() // must not panic
}

//  Request body helpers 

func TestLogin_SendsCorrectBody(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]string{"token": "t"})
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{})
	api.Login(srv.URL, "alice", "secret")
	if gotBody["username"] != "alice" {
		t.Errorf("username = %q, want %q", gotBody["username"], "alice")
	}
	if gotBody["password"] != "secret" {
		t.Errorf("password = %q", gotBody["password"])
	}
}

func TestDownload_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		io.WriteString(w, "data")
	}))
	defer srv.Close()

	api := NewIliadApi(Credentials{BaseURL: srv.URL, Token: "mytoken"})
	dest := t.TempDir() + "/out.gz"
	api.DownloadAudiobook("abc", dest, 0, nil)

	if gotAuth != "Bearer mytoken" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer mytoken")
	}
}
