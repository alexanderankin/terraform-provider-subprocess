package internal

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type runningProcess struct {
	cmd              *exec.Cmd
	started          time.Time
	done             chan struct{}
	exitErr          error // written before done is closed
	mu               sync.Mutex
	exited           bool
	opened           bool
	closing          bool
	onUnexpectedExit func(error)
}

func startProcess(cmd *exec.Cmd, onUnexpectedExit func(error)) *runningProcess {
	p := &runningProcess{cmd: cmd, started: time.Now(), done: make(chan struct{}), onUnexpectedExit: onUnexpectedExit}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.exitErr = err
		p.exited = true
		logExit := p.opened && !p.closing && p.onUnexpectedExit != nil
		close(p.done)
		p.mu.Unlock()
		if logExit {
			p.onUnexpectedExit(err)
		}
	}()
	return p
}

// markOpened establishes whether the exit happened during Open or afterward.
func (p *runningProcess) markOpened() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		return fmt.Errorf("process exited before Open completed: %s", processExit(p.exitErr))
	}
	p.opened = true
	return nil
}

func (p *runningProcess) stop() error {
	p.mu.Lock()
	p.closing = true
	p.mu.Unlock()
	err := p.cmd.Process.Kill()
	<-p.done
	return err
}

type waitConfig struct {
	delayMs     *int64
	outputRegex *regexp.Regexp
	ports       []int
	timeoutMs   *int64
}

// A millisecond count can exceed time.Duration's nanosecond range. Saturate
// instead of wrapping a long configured wait into an immediate one.
func durationMilliseconds(ms int64) time.Duration {
	const maxDuration = time.Duration(1<<63 - 1)
	if ms > int64(maxDuration/time.Millisecond) {
		return maxDuration
	}
	return time.Duration(ms) * time.Millisecond
}

// Each stream has its own matcher. Writes from exec's copy goroutines are
// serialized per stream, and only complete newline-terminated lines match.
type lineMatcher struct {
	mu      sync.Mutex
	partial string
	regex   *regexp.Regexp
	matched chan<- struct{}
	done    bool
}

var _ io.Writer = (*lineMatcher)(nil)

func newLineMatcher(regex *regexp.Regexp, matched chan<- struct{}) *lineMatcher {
	return &lineMatcher{regex: regex, matched: matched}
}

func (m *lineMatcher) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.regex == nil || m.done {
		return len(p), nil
	}
	m.partial += string(p)
	for {
		i := strings.IndexByte(m.partial, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSuffix(m.partial[:i], "\r")
		m.partial = m.partial[i+1:]
		if m.regex != nil && m.regex.MatchString(line) {
			m.done = true
			m.partial = ""
			select {
			case m.matched <- struct{}{}:
			default:
			}
			break
		}
	}
	return len(p), nil
}

func waitForReady(ctx context.Context, process *runningProcess, config waitConfig, matched <-chan struct{}) error {
	if config.delayMs == nil && config.outputRegex == nil && len(config.ports) == 0 {
		return nil
	}
	if config.timeoutMs != nil && *config.timeoutMs == 0 {
		return fmt.Errorf("wait timed out after 0 ms")
	}
	var delayTimer *time.Timer
	var delayC <-chan time.Time
	delayDone := config.delayMs == nil || *config.delayMs == 0
	if !delayDone {
		remaining := max(time.Until(process.started.Add(durationMilliseconds(*config.delayMs))), 0)
		delayTimer = time.NewTimer(remaining)
		delayC = delayTimer.C
		defer delayTimer.Stop()
	}
	var timeoutTimer *time.Timer
	var timeoutC <-chan time.Time
	waitCtx := ctx
	if config.timeoutMs != nil {
		deadline := process.started.Add(durationMilliseconds(*config.timeoutMs))
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
		remaining := max(time.Until(deadline), 0)
		timeoutTimer = time.NewTimer(remaining)
		timeoutC = timeoutTimer.C
		defer timeoutTimer.Stop()
	}
	outputDone := config.outputRegex == nil
	portDone := make([]bool, len(config.ports))
	probe := func() bool {
		all := true
		for i, port := range config.ports {
			if portDone[i] {
				continue
			}
			address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			connection, err := (&net.Dialer{Timeout: 100 * time.Millisecond}).DialContext(waitCtx, "tcp", address)
			if err == nil {
				_ = connection.Close()
				portDone[i] = true
			} else {
				all = false
			}
		}
		return all
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	portsDone := probe()
	for {
		if config.timeoutMs != nil && time.Since(process.started) >= durationMilliseconds(*config.timeoutMs) {
			return fmt.Errorf("wait timed out after %d ms", *config.timeoutMs)
		}
		select {
		case <-process.done:
			return fmt.Errorf("process exited before readiness: %s", processExit(process.exitErr))
		default:
		}
		if delayDone && outputDone && portsDone {
			return nil
		}
		select {
		case <-process.done:
			return fmt.Errorf("process exited before readiness: %s", processExit(process.exitErr))
		case <-ctx.Done():
			return fmt.Errorf("wait cancelled: %w", ctx.Err())
		case <-timeoutC:
			return fmt.Errorf("wait timed out after %d ms", *config.timeoutMs)
		case <-delayC:
			delayDone = true
			delayC = nil
		case <-matched:
			outputDone = true
		case <-ticker.C:
			portsDone = probe()
		}
	}
}

func processExit(err error) string {
	if err == nil {
		return "exit status 0"
	}
	return err.Error()
}
