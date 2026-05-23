package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseArgsDefault(t *testing.T) {
	got, err := parseArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.command != "tui" {
		t.Errorf("command = %q, want %q", got.command, "tui")
	}
	if got.configFile != "" {
		t.Errorf("configFile = %q, want empty", got.configFile)
	}
}

func TestParseArgsLogin(t *testing.T) {
	got, err := parseArgs([]string{"login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.command != "login" {
		t.Errorf("command = %q, want %q", got.command, "login")
	}
}

func TestParseArgsLogout(t *testing.T) {
	got, err := parseArgs([]string{"logout"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.command != "logout" {
		t.Errorf("command = %q, want %q", got.command, "logout")
	}
}

func TestParseArgsUnknownCommand(t *testing.T) {
	_, err := parseArgs([]string{"foo"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestParseArgsConfigFlag(t *testing.T) {
	got, err := parseArgs([]string{"--config", "odyssey.cfg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.configFile != "odyssey.cfg" {
		t.Errorf("configFile = %q, want %q", got.configFile, "odyssey.cfg")
	}
	if got.command != "tui" {
		t.Errorf("command = %q, want tui", got.command)
	}
}

func TestParseArgsConfigWithCommand(t *testing.T) {
	got, err := parseArgs([]string{"--config", "odyssey.cfg", "login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.configFile != "odyssey.cfg" {
		t.Errorf("configFile = %q, want %q", got.configFile, "odyssey.cfg")
	}
	if got.command != "login" {
		t.Errorf("command = %q, want login", got.command)
	}
}

func TestParseArgsConfigMissingValue(t *testing.T) {
	_, err := parseArgs([]string{"--config"})
	if err == nil {
		t.Fatal("expected error when --config has no value")
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	_, err := parseArgs([]string{"--foo"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseArgsRegisterFlag(t *testing.T) {
	got, err := parseArgs([]string{"--register", "login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.register {
		t.Error("expected register=true")
	}
	if got.command != "login" {
		t.Errorf("command = %q, want login", got.command)
	}
}

func TestExecLoginSuccess(t *testing.T) {
	store := newTestStore(t)
	mock := &MockApiClient{
		LoginFn: func(baseURL, user, pass string) (string, error) {
			return "tok-abc", nil
		},
	}
	err := execLogin("https://example.com", "alice", "s3cr3t", false, mock, store)
	if err != nil {
		t.Fatalf("execLogin error: %v", err)
	}
	creds := store.LoadCredentials()
	if creds == nil {
		t.Fatal("expected credentials stored")
	}
	if creds.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %q", creds.BaseURL)
	}
	if creds.Username != "alice" {
		t.Errorf("Username = %q", creds.Username)
	}
	if creds.Token != "tok-abc" {
		t.Errorf("Token = %q", creds.Token)
	}
}

func TestExecLoginAuthError(t *testing.T) {
	store := newTestStore(t)
	mock := &MockApiClient{
		LoginFn: func(baseURL, user, pass string) (string, error) {
			return "", &AuthError{}
		},
	}
	err := execLogin("https://example.com", "alice", "wrong", false, mock, store)
	if err == nil {
		t.Fatal("expected error")
	}
	if store.LoadCredentials() != nil {
		t.Error("credentials should not be stored on failure")
	}
}

func TestExecLoginNetworkError(t *testing.T) {
	store := newTestStore(t)
	netErr := errors.New("connection refused")
	mock := &MockApiClient{
		LoginFn: func(baseURL, user, pass string) (string, error) {
			return "", netErr
		},
	}
	err := execLogin("https://example.com", "alice", "pass", false, mock, store)
	if !errors.Is(err, netErr) {
		t.Errorf("expected netErr, got %v", err)
	}
}

func TestExecRegisterSuccess(t *testing.T) {
	store := newTestStore(t)
	mock := &MockApiClient{
		RegisterFn: func(baseURL, user, pass string) (string, error) {
			return "tok-reg", nil
		},
	}
	err := execLogin("https://example.com", "newuser", "pass", true, mock, store)
	if err != nil {
		t.Fatalf("execLogin (register) error: %v", err)
	}
	creds := store.LoadCredentials()
	if creds == nil {
		t.Fatal("expected credentials stored after register")
	}
	if creds.Token != "tok-reg" {
		t.Errorf("Token = %q", creds.Token)
	}
}

func TestExecRegisterUsernameTaken(t *testing.T) {
	store := newTestStore(t)
	mock := &MockApiClient{
		RegisterFn: func(baseURL, user, pass string) (string, error) {
			return "", errors.New("username already taken")
		},
	}
	err := execLogin("https://example.com", "taken", "pass", true, mock, store)
	if err == nil {
		t.Fatal("expected error")
	}
	if store.LoadCredentials() != nil {
		t.Error("credentials should not be stored on failure")
	}
}

func TestExecLogoutClearsCreds(t *testing.T) {
	store := newTestStore(t)
	store.SaveCredentials(Credentials{
		BaseURL: "https://example.com", Username: "alice", Password: "pass", Token: "tok",
	})
	if err := execLogout(store); err != nil {
		t.Fatalf("execLogout error: %v", err)
	}
	if store.LoadCredentials() != nil {
		t.Error("expected credentials to be cleared")
	}
}

func TestExecLogoutIdempotent(t *testing.T) {
	store := newTestStore(t)
	if err := execLogout(store); err != nil {
		t.Fatalf("execLogout with no creds: %v", err)
	}
}

func TestCheckCredentialsPresent(t *testing.T) {
	store := newTestStore(t)
	store.SaveCredentials(Credentials{
		BaseURL: "https://example.com", Username: "alice", Password: "pass", Token: "tok",
	})
	if err := checkCredentials(store); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckCredentialsMissing(t *testing.T) {
	store := newTestStore(t)
	err := checkCredentials(store)
	if err == nil {
		t.Fatal("expected error when no credentials")
	}
	if !strings.Contains(err.Error(), "odyssey login") {
		t.Errorf("error should mention 'odyssey login', got: %v", err)
	}
}
