package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

var ErrMissingConfig = errors.New("missing config")

type LoadOptions struct {
	HomeDir   string
	LookupEnv func(string) (string, bool)
}

func Load(path string, opts LoadOptions) (session.Config, error) {
	opts = opts.withDefaults()
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return session.Config{}, fmt.Errorf("%w: %s", ErrMissingConfig, path)
		}
		return session.Config{}, err
	}
	defer file.Close()

	var remotes []session.Remote
	var current *session.Remote
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if !strings.HasPrefix(section, "remote.") || len(section) == len("remote.") {
				return session.Config{}, fmt.Errorf("line %d: unsupported section [%s]", lineNo, section)
			}
			name := strings.TrimPrefix(section, "remote.")
			if seen[name] {
				return session.Config{}, fmt.Errorf("line %d: duplicate remote %q", lineNo, name)
			}
			seen[name] = true
			remotes = append(remotes, session.Remote{Name: name, Destination: "/tmp/ssh-drop/"})
			current = &remotes[len(remotes)-1]
			continue
		}
		if current == nil {
			return session.Config{}, fmt.Errorf("line %d: key outside remote section", lineNo)
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return session.Config{}, fmt.Errorf("line %d: expected key = value", lineNo)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		expanded, err := expand(value, opts, key)
		if err != nil {
			return session.Config{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if err := assign(current, key, expanded); err != nil {
			return session.Config{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return session.Config{}, err
	}
	if len(remotes) == 0 {
		return session.Config{}, errors.New("config must define at least one [remote.<name>] section")
	}
	for _, remote := range remotes {
		if strings.TrimSpace(remote.Host) == "" {
			return session.Config{}, fmt.Errorf("remote %q: host is required", remote.Name)
		}
	}
	return session.Config{Remotes: remotes}, nil
}

func (o LoadOptions) withDefaults() LoadOptions {
	if o.HomeDir == "" {
		o.HomeDir = "."
	}
	if o.LookupEnv == nil {
		o.LookupEnv = os.LookupEnv
	}
	return o
}

func assign(remote *session.Remote, key string, value string) error {
	switch key {
	case "host":
		remote.Host = value
	case "user":
		remote.User = value
	case "port":
		if value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("port must be numeric")
			}
		}
		remote.Port = value
	case "identity_file":
		remote.IdentityFile = value
	case "forward_agent":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("forward_agent must be true or false")
		}
		remote.ForwardAgent = parsed
	case "destination":
		if value == "" {
			return fmt.Errorf("destination must not be empty")
		}
		remote.Destination = value
	default:
		return fmt.Errorf("unsupported key %q", key)
	}
	return nil
}

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

func expand(value string, opts LoadOptions, key string) (string, error) {
	value = strings.Trim(value, `"`)
	if value == "~" {
		value = opts.HomeDir
	} else if strings.HasPrefix(value, "~/") {
		value = filepath.Join(opts.HomeDir, strings.TrimPrefix(value, "~/"))
	}

	var missing []string
	expanded := envPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := envPattern.FindStringSubmatch(match)
		name := parts[1]
		if name == "" {
			name = parts[2]
		}
		val, ok := opts.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("%s references missing environment variable %s", key, missing[0])
	}
	return expanded, nil
}
