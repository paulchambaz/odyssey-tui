package main

import (
	"fmt"
	"log"
	"os"
)

var appLog *log.Logger

func initLogger(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	appLog = log.New(f, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	logf("APP", "=== session start pid=%d ===", os.Getpid())
}

func logf(cat, format string, args ...any) {
	if appLog == nil {
		return
	}
	appLog.Printf("[%-8s] %s", cat, fmt.Sprintf(format, args...))
}

func modeName(m Mode) string {
	switch m {
	case ModeMain:
		return "main"
	case ModeHelp:
		return "help"
	case ModeSearch:
		return "search"
	case ModeSearching:
		return "searching"
	default:
		return "unknown"
	}
}
