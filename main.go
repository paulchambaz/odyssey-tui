package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type cliArgs struct {
	command    string
	configFile string
	register   bool
	debug      bool
}

func parseArgs(args []string) (cliArgs, error) {
	var out cliArgs
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-config":
			if i+1 >= len(args) {
				return out, fmt.Errorf("-config requires a value")
			}
			out.configFile = args[i+1]
			i += 2
		case "-register":
			out.register = true
			i++
		case "-debug":
			out.debug = true
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return out, fmt.Errorf("unknown flag: %s\nusage: odyssey [-config <path>] [-register] [-debug] [login|logout]", args[i])
			}
			if out.command != "" {
				return out, fmt.Errorf("unexpected argument: %s", args[i])
			}
			switch args[i] {
			case "login", "logout":
				out.command = args[i]
			default:
				return out, fmt.Errorf("unknown command: %s\nusage: odyssey [-config <path>] [-register] [-debug] [login|logout]", args[i])
			}
			i++
		}
	}
	if out.command == "" {
		out.command = "tui"
	}
	return out, nil
}

func execLogin(url, user, pass string, register bool, api ApiClient, store *Store) error {
	var token string
	var err error
	if register {
		token, err = api.Register(url, user, pass)
	} else {
		token, err = api.Login(url, user, pass)
	}
	if err != nil {
		return err
	}
	return store.SaveCredentials(Credentials{
		BaseURL:  url,
		Username: user,
		Password: pass,
		Token:    token,
	})
}

func execLogout(store *Store) error {
	return store.ClearCredentials()
}

func checkCredentials(store *Store) error {
	if store.LoadCredentials() == nil {
		return fmt.Errorf("no credentials stored — run 'odyssey login' first")
	}
	return nil
}

func runLoginWizard(store *Store, register bool) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Server URL: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	fmt.Print("Username: ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print("Password: ")
	passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
		os.Exit(1)
	}
	pass := string(passBytes)

	api := NewIliadApi(Credentials{})
	if err := execLogin(url, user, pass, register, api, store); err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("logged in")
}

func runTUI(store *Store) {
	creds := store.LoadCredentials()
	api := NewIliadApi(*creds)

	logf("APP", "launching TUI user=%s url=%s", creds.Username, creds.BaseURL)

	ps := newPlayerState(&Library{})
	ps.api = api
	ps.store = store
	ps.playerSpeed = store.LoadFloat("playback_speed", 1.0)

	mpv, err := newMpvPlayer()
	if err != nil {
		logf("APP", "mpv init failed: %v", err)
	} else {
		ps.mpv = mpv
	}

	p := tea.NewProgram(ps, tea.WithAltScreen())
	ps.program = p

	if _, err := p.Run(); err != nil {
		logf("APP", "TUI error: %v", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if ps.mpv != nil {
		ps.mpv.quit()
	}
	logf("APP", "=== session end ===")
}

func main() {
	cli, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var store *Store
	if cli.configFile != "" {
		store, err = NewStoreAt(cli.configFile)
	} else {
		store, err = NewStore()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cli.debug {
		initLogger(store.LogPath())
	}
	logf("APP", "command=%s config=%q", cli.command, cli.configFile)

	switch cli.command {
	case "login":
		runLoginWizard(store, cli.register)
	case "logout":
		if err := execLogout(store); err != nil {
			fmt.Fprintf(os.Stderr, "logout failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("logged out")
	case "tui":
		if store.LoadCredentials() == nil {
			runLoginWizard(store, false)
		}
		runTUI(store)
	}
}
