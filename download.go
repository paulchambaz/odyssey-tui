package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func startDownloadCmd(api ApiClient, store *Store, book *Audiobook, prog *tea.Program, ctx context.Context) tea.Cmd {
	hash := book.Hash
	return func() tea.Msg {
		go func() {
			// 1. Poll until archive is ready on the server.
			for {
				select {
				case <-ctx.Done():
					prog.Send(downloadDoneMsg{hash: hash, err: ctx.Err()})
					return
				default:
				}
				b, err := api.GetAudiobook(hash)
				if err == nil && b.ArchiveReady {
					break
				}
				select {
				case <-ctx.Done():
					prog.Send(downloadDoneMsg{hash: hash, err: ctx.Err()})
					return
				case <-time.After(3 * time.Second):
				}
			}

			// 2. Check for a partial archive from a previous attempt.
			archivePath := store.LibraryDir(hash) + ".tar.gz"
			var startByte int64
			if info, err := os.Stat(archivePath); err == nil {
				startByte = info.Size()
			}

			// 3. Stream download in a sub-goroutine so we can select on ctx.Done.
			dlDone := make(chan error, 1)
			go func() {
				dlDone <- api.DownloadAudiobook(hash, archivePath, startByte, func(received, total int64) {
					var pct float64
					if total > 0 {
						pct = float64(received) / float64(total)
					}
					prog.Send(downloadProgressMsg{hash: hash, pct: pct})
				})
			}()

			select {
			case <-ctx.Done():
				api.CancelDownload()
				<-dlDone
				os.Remove(archivePath)
				prog.Send(downloadDoneMsg{hash: hash, err: ctx.Err()})
				return
			case err := <-dlDone:
				if err != nil {
					prog.Send(downloadDoneMsg{hash: hash, err: err})
					return
				}
			}

			// 4. Extract.
			libDir := store.LibraryDir(hash)
			if err := extractTarGz(archivePath, libDir); err != nil {
				prog.Send(downloadDoneMsg{hash: hash, err: err})
				return
			}

			// 5. Delete archive, cache server position.
			os.Remove(archivePath)
			if pos, err := api.GetPosition(hash); err == nil {
				store.SaveServerPosition(hash, pos)
			}

			// 6. Signal success.
			prog.Send(downloadDoneMsg{hash: hash, err: nil})
		}()
		return nil
	}
}

func extractTarGz(archive, destDir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		if !filepath.IsLocal(hdr.Name) {
			return fmt.Errorf("unsafe path in archive: %s", hdr.Name)
		}
		dest := filepath.Join(destDir, hdr.Name)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, cpErr := io.Copy(out, tr)
		out.Close()
		if cpErr != nil {
			return cpErr
		}
	}
	return nil
}
