// Package herdr invokes the installed Herdr CLI without passing through a shell.
package herdr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

const Source = "herdr-web-bridge"

type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type ExecRunner struct {
	Path string
}

type CommandError struct {
	ExitCode int
	Message  string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("herdr command failed (exit %d): %s", e.ExitCode, e.Message)
}

func (r ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	path := r.Path
	if path == "" {
		path = "herdr"
	}
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		message := security.TruncateRunes(security.RedactLog(stderr.String()), 240)
		if message == "" {
			message = security.TruncateRunes(err.Error(), 240)
		}
		return nil, &CommandError{ExitCode: exitCode, Message: message}
	}
	if stdout.Len() > 8<<20 {
		return nil, errors.New("Herdr output exceeded 8 MiB")
	}
	return stdout.Bytes(), nil
}

type Client struct {
	Runner Runner
}

func NewClient(runner Runner) *Client { return &Client{Runner: runner} }

func (c *Client) Status(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, "status")
	return err
}

func (c *Client) FocusWorkspace(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		return errors.New("workspace id is required")
	}
	_, err := c.Runner.Run(ctx, "workspace", "focus", workspaceID)
	return err
}

func MetadataArgs(workspaceID, state, title string, seq uint64) ([]string, error) {
	if workspaceID == "" || seq == 0 {
		return nil, errors.New("workspace id and non-zero sequence are required")
	}
	args := []string{"workspace", "report-metadata", workspaceID, "--source", Source}
	if state == "idle" || state == "viewed" {
		return append(args, "--clear-token", "web_status", "--seq", strconv.FormatUint(seq, 10)), nil
	}
	prefixes := map[string]string{
		"running": "⏳ 正在生成：",
		"done_unread": "✅ 等待查看：",
		"error": "⚠️ 网页出错：",
		"unknown": "❔ 状态未知：",
	}
	prefix, ok := prefixes[state]
	if !ok {
		return nil, fmt.Errorf("unsupported state %q", state)
	}
	text := security.TruncateRunes(prefix+strings.TrimSpace(title), 80)
	return append(args,
		"--token", "web_status="+text,
		"--seq", strconv.FormatUint(seq, 10),
		"--ttl-ms", "86400000",
	), nil
}

func (c *Client) ReportMetadata(ctx context.Context, workspaceID, state, title string, seq uint64) error {
	args, err := MetadataArgs(workspaceID, state, title, seq)
	if err != nil {
		return err
	}
	return c.retry(ctx, args, 3)
}

func NotificationArgs(title, body, sound string) ([]string, error) {
	if sound != "done" && sound != "request" && sound != "none" {
		return nil, errors.New("invalid notification sound")
	}
	title = security.TruncateRunes(title, 80)
	body = security.TruncateRunes(body, 240)
	if title == "" {
		return nil, errors.New("notification title is required")
	}
	return []string{"notification", "show", title, "--body", body, "--sound", sound}, nil
}

func (c *Client) Notify(ctx context.Context, title, body, sound string) error {
	args, err := NotificationArgs(title, body, sound)
	if err != nil {
		return err
	}
	_, err = c.Runner.Run(ctx, args...)
	return err
}

func (c *Client) retry(ctx context.Context, args []string, attempts int) error {
	delays := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		_, err = c.Runner.Run(ctx, args...)
		if err == nil {
			return nil
		}
		if attempt+1 >= attempts {
			break
		}
		timer := time.NewTimer(delays[min(attempt, len(delays)-1)])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

