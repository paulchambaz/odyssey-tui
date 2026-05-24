package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name    string
	content string // empty = directory entry
	isDir   bool
}

func makeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		if e.isDir {
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     e.name,
				Mode:     0755,
			}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0644,
			Size: int64(len(e.content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractTarGzBasic(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "info.yml", content: "title: Test\nauthor: Author\n"},
		{name: "chapter1.opus", content: "audio data 1"},
		{name: "chapter2.opus", content: "audio data 2"},
	})
	dest := t.TempDir()

	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	cases := map[string]string{
		"info.yml":      "title: Test\nauthor: Author\n",
		"chapter1.opus": "audio data 1",
		"chapter2.opus": "audio data 2",
	}
	for name, want := range cases {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Errorf("file %q missing: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("file %q: got %q, want %q", name, got, want)
		}
	}
}

func TestExtractTarGzNestedDirs(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "subdir/nested.txt", content: "nested content"},
		{name: "subdir/deep/file.txt", content: "deep content"},
	})
	dest := t.TempDir()

	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	cases := map[string]string{
		filepath.Join("subdir", "nested.txt"):        "nested content",
		filepath.Join("subdir", "deep", "file.txt"):  "deep content",
	}
	for rel, want := range cases {
		got, err := os.ReadFile(filepath.Join(dest, rel))
		if err != nil {
			t.Errorf("file %q missing: %v", rel, err)
			continue
		}
		if string(got) != want {
			t.Errorf("file %q: got %q, want %q", rel, got, want)
		}
	}
}

func TestExtractTarGzSkipsDirectoryEntries(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "subdir/", isDir: true},
		{name: "subdir/file.txt", content: "hello"},
	})
	dest := t.TempDir()

	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "subdir", "file.txt"))
	if err != nil {
		t.Fatalf("file missing after skipping dir entry: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestExtractTarGzInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(path, []byte("not a valid gzip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(path, t.TempDir()); err == nil {
		t.Error("expected error for invalid archive, got nil")
	}
}
