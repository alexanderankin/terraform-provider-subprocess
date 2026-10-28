package internal

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWaitForOutputRegex(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output string
		stderr bool
		match  bool
	}{
		{name: "abc", output: "abc\n"},
		{name: "abcc", output: "abcc\n", match: true},
		{name: "abcd on stderr", output: "abcd\n", stderr: true},
		{name: "abcc on stderr", output: "abcc\n", stderr: true, match: true},
		{name: "abcc without newline", output: "abcc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var script string
			if tc.stderr {
				script = "printf '%s' \"$1\" >&2; exec sleep 5"
			} else {
				script = "printf '%s' \"$1\"; exec sleep 5"
			}
			cmd := exec.Command("sh", "-c", script, "sh", tc.output)

			matched := make(chan struct{}, 1)
			regex := regexp.MustCompile("abcc+")
			cmd.Stdout = newLineMatcher(regex, matched)
			cmd.Stderr = newLineMatcher(regex, matched)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			process := startProcess(cmd, nil)
			defer func() { _ = process.stop() }()
			timeoutMs := int64(500)
			err := waitForReady(context.Background(), process, waitConfig{outputRegex: regex, timeoutMs: &timeoutMs}, matched)
			if tc.match && err != nil {
				t.Fatalf("expected readiness, got %v", err)
			}
			if !tc.match && (err == nil || !strings.Contains(err.Error(), "timed out")) {
				t.Fatalf("expected timeout, got %v", err)
			}
		})
	}
}

func TestWaitForStartupDelay(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	process := startProcess(cmd, nil)
	defer func() { _ = process.stop() }()
	err := waitForReady(context.Background(), process, waitConfig{delayMs: new(int64(40))}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(process.started) < 40*time.Millisecond {
		t.Fatal("returned before delay")
	}
}

func TestWaitTimeoutBeforeDelay(t *testing.T) {
	for _, delay := range []int64{1000, 300001, 1<<63 - 1} {
		t.Run(strconv.FormatInt(delay, 10), func(t *testing.T) {
			cmd := exec.Command("sleep", "5")
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			process := startProcess(cmd, nil)
			defer func() { _ = process.stop() }()
			timeout := int64(20)
			err := waitForReady(context.Background(), process, waitConfig{delayMs: &delay, timeoutMs: &timeout}, nil)
			if err == nil || !strings.Contains(err.Error(), "timed out") {
				t.Fatalf("expected timeout, got %v", err)
			}
		})
	}
}

func TestProcessExitDuringWait(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	logged := make(chan error, 1)
	process := startProcess(cmd, func(err error) { logged <- err })
	defer func() { _ = process.stop() }()
	delay := int64(500)
	timeout := int64(1000)
	err := waitForReady(context.Background(), process, waitConfig{delayMs: &delay, timeoutMs: &timeout}, nil)
	if err == nil || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("expected child exit during wait, got %v", err)
	}
	if err := process.markOpened(); err == nil {
		t.Fatal("Open should fail after process exit")
	}
	select {
	case <-logged:
		t.Fatal("exit during Open should not be logged as a later exit")
	default:
	}
}

func TestProcessExitAfterOpen(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 0.2; exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	logged := make(chan error, 1)
	process := startProcess(cmd, func(err error) { logged <- err })
	defer func() { _ = process.stop() }()
	if err := process.markOpened(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-logged:
		if err == nil || !strings.Contains(err.Error(), "exit status 7") {
			t.Fatalf("unexpected exit status: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected exit notification before Close")
	}
}

func TestCloseSuppressesExitLog(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	logged := make(chan error, 1)
	process := startProcess(cmd, func(err error) { logged <- err })
	if err := process.markOpened(); err != nil {
		t.Fatal(err)
	}
	if err := process.stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-logged:
		t.Fatal("Close-triggered exit should not be logged")
	default:
	}
}
