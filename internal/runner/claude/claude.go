// Package claude implements the Runner interface for the Claude CLI agent.
package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/runner"
)

func init() {
	runner.Register("claude", func() runner.Runner { return New() })
}

// Claude implements runner.Runner by spawning the Claude CLI subprocess.
type Claude struct{}

// New returns a new Claude runner.
func New() *Claude { return &Claude{} }

// Run spawns the claude subprocess and returns a channel of events and an error channel.
func (c *Claude) Run(opts runner.RunOptions) (<-chan runner.Event, <-chan error) {
	events := make(chan runner.Event, 64)
	errc := make(chan error, 1)

	go func() {
		defer close(events)
		if err := run(opts, events); err != nil {
			errc <- err
		}
		close(errc)
	}()

	return events, errc
}

func run(opts runner.RunOptions, events chan<- runner.Event) error {
	cfg := opts.Config
	cmd := []string{cfg.Agent.Command, "-p", opts.Prompt}
	if opts.SystemPrompt != "" {
		cmd = append(cmd, "--system-prompt", opts.SystemPrompt)
	}
	cmd = append(cmd, cfg.Agent.Args...)

	if len(cfg.Agent.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(cfg.Agent.AllowedTools, ","))
	}
	if cfg.Agent.DangerouslySkipPermissions {
		cmd = append(cmd, "--dangerously-skip-permissions")
	}
	if opts.SessionID != "" {
		cmd = append(cmd, "--resume", opts.SessionID)
	}

	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	proc := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
	proc.Dir = cwd
	proc.Stderr = io.Discard

	stdout, err := proc.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting claude process: %w", err)
	}

	var debugLog *os.File
	if cfg.Debug.Enabled {
		debugLog = openDebugLog(cfg, opts.Command, cwd)
		if debugLog != nil {
			defer debugLog.Close()
		}
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MiB lines
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if debugLog != nil {
			fmt.Fprintln(debugLog, line)
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		eventType, _ := data["type"].(string)
		events <- runner.Event{Type: eventType, Data: data}
	}

	if err := proc.Wait(); err != nil {
		return fmt.Errorf("claude process exited with error: %w", err)
	}
	return nil
}

func openDebugLog(cfg config.Config, command, cwd string) *os.File {
	logDir := filepath.Join(cwd, cfg.Debug.LogDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}
	ts := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("%s_%s_%s.log", ts, cfg.Agent.Command, command)
	f, err := os.Create(filepath.Join(logDir, filename))
	if err != nil {
		return nil
	}
	return f
}
