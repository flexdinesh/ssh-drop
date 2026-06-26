package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dineshpandiyan/ssh-drop/internal/config"
	"github.com/dineshpandiyan/ssh-drop/internal/session"
	"github.com/dineshpandiyan/ssh-drop/internal/tui"
)

type Deps struct {
	Stdout    io.Writer
	Stderr    io.Writer
	Version   string
	HomeDir   string
	EnvLookup func(string) (string, bool)
	LookPath  func(string) (string, error)
	RunUI     func(session.Start) (session.Summary, error)
}

func RealDeps(version string) Deps {
	home, _ := os.UserHomeDir()
	return Deps{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Version:   version,
		HomeDir:   home,
		EnvLookup: os.LookupEnv,
		LookPath:  exec.LookPath,
		RunUI:     tui.Run,
	}
}

type options struct {
	configPath string
	to         string
	help       bool
	version    bool
}

func Run(args []string, deps Deps) int {
	deps = deps.withDefaults()

	opts, err := parseArgs(args, deps.HomeDir, deps.EnvLookup)
	if err != nil {
		fmt.Fprintln(deps.Stderr, err)
		printUsage(deps.Stderr)
		return 2
	}
	if opts.help {
		printUsage(deps.Stdout)
		return 0
	}
	if opts.version {
		fmt.Fprintf(deps.Stdout, "ssh-drop %s\n", deps.Version)
		return 0
	}

	cfg, err := config.Load(opts.configPath, config.LoadOptions{
		HomeDir:   deps.HomeDir,
		LookupEnv: deps.EnvLookup,
	})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "config error: %v\n", err)
		if errors.Is(err, config.ErrMissingConfig) {
			printConfigSample(deps.Stderr)
		}
		return 1
	}

	if opts.to != "" && cfg.FindRemote(opts.to) == nil {
		fmt.Fprintf(deps.Stderr, "unknown remote %q\n", opts.to)
		return 1
	}

	if _, err := deps.LookPath("rsync"); err != nil {
		fmt.Fprintln(deps.Stderr, "rsync is required but was not found in PATH")
		return 1
	}

	summary, err := deps.RunUI(session.Start{
		Config:            cfg,
		PreselectedRemote: opts.to,
	})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "ui error: %v\n", err)
		return 1
	}
	printSummary(deps.Stdout, summary)
	return 0
}

func (d Deps) withDefaults() Deps {
	if d.Stdout == nil {
		d.Stdout = io.Discard
	}
	if d.Stderr == nil {
		d.Stderr = io.Discard
	}
	if d.Version == "" {
		d.Version = "dev"
	}
	if d.HomeDir == "" {
		d.HomeDir = "."
	}
	if d.EnvLookup == nil {
		d.EnvLookup = func(string) (string, bool) { return "", false }
	}
	if d.LookPath == nil {
		d.LookPath = exec.LookPath
	}
	if d.RunUI == nil {
		d.RunUI = func(session.Start) (session.Summary, error) {
			return session.Summary{}, nil
		}
	}
	return d
}

func parseArgs(args []string, home string, lookupEnv func(string) (string, bool)) (options, error) {
	opts := options{configPath: defaultConfigPath(home, lookupEnv)}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			opts.help = true
		case "--version":
			opts.version = true
		case "--config":
			i++
			if i >= len(args) || args[i] == "" {
				return opts, errors.New("--config requires a path")
			}
			opts.configPath = args[i]
		case "--to":
			i++
			if i >= len(args) || args[i] == "" {
				return opts, errors.New("--to requires a remote name")
			}
			opts.to = args[i]
		default:
			return opts, fmt.Errorf("unsupported argument %q", args[i])
		}
	}
	return opts, nil
}

func defaultConfigPath(home string, lookupEnv func(string) (string, bool)) string {
	if xdgConfigHome, ok := lookupEnv("XDG_CONFIG_HOME"); ok && xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "ssh-drop.conf")
	}
	return filepath.Join(home, ".config", "ssh-drop.conf")
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ssh-drop [--config <path>] [--to <remote-name>]")
	fmt.Fprintln(w, "  ssh-drop --help")
	fmt.Fprintln(w, "  ssh-drop --version")
}

func printConfigSample(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Create a config file like:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "[remote.cb]")
	fmt.Fprintln(w, "host = cb")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "[remote.files]")
	fmt.Fprintln(w, "host = files.example.com")
	fmt.Fprintln(w, "user = deploy")
	fmt.Fprintln(w, "identity_file = ~/.ssh/files")
	fmt.Fprintln(w, "forward_agent = true")
	fmt.Fprintln(w, "destination = /tmp/ssh-drop/")
}

func printSummary(w io.Writer, summary session.Summary) {
	if summary.Empty() {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Summary")
	fmt.Fprintf(w, "  success: %d\n", summary.Successes)
	fmt.Fprintf(w, "  failed: %d\n", summary.Failures)
	fmt.Fprintf(w, "  canceled: %d\n", summary.Canceled)
	if len(summary.SuccessfulDestinations) > 0 {
		fmt.Fprintln(w, "  destinations:")
		for _, destination := range summary.SuccessfulDestinations {
			fmt.Fprintf(w, "    %s\n", destination)
		}
	}
}
