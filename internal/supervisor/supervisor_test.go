package supervisor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--subprocess-supervisor":
			os.Exit(BecomeSupervisor())
		case "--supervisor-test-parent":
			os.Exit(runTestParent())
		case "--supervisor-test-target":
			time.Sleep(30 * time.Second)
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

func runTestParent() int {
	s, err := EnsureSupervisor()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	executable, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(executable, "--supervisor-test-target")
	cmd.Stdout = os.Stdout // Keep the test pipe open until the target is killed.
	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := s.UpdateSupervisor(cmd.Process.Pid, true); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(os.Stdout, cmd.Process.Pid)
	time.Sleep(30 * time.Second) // The test kills this process after registration.
	return 0
}

func TestKillsRegisteredProcessWhenParentExits(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = read.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	parent := exec.CommandContext(ctx, executable, "--supervisor-test-parent")
	parent.Stdout = write
	var stderr bytes.Buffer
	parent.Stderr = &stderr
	if err := parent.Start(); err != nil {
		_ = write.Close()
		t.Fatal(err)
	}
	_ = write.Close()
	reader := bufio.NewReader(read)
	line := make(chan string, 1)
	go func() {
		value, _ := reader.ReadString('\n')
		line <- value
	}()
	var pid int
	select {
	case value := <-line:
		pid, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			_ = parent.Wait()
			t.Fatalf("parent did not report a target PID: %q; stderr: %s", value, stderr.String())
		}
	case <-ctx.Done():
		_ = read.Close()
		_ = parent.Wait()
		t.Fatal("timed out waiting for parent to register the target")
	}
	if err := parent.Process.Kill(); err != nil {
		_ = parent.Wait()
		t.Fatalf("killing parent: %v; stderr: %s", err, stderr.String())
	}
	_ = parent.Wait()

	closed := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, reader)
		closed <- err
	}()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("reading target output: %v", err)
		}
	case <-ctx.Done():
		if process, err := os.FindProcess(pid); err == nil {
			_ = process.Kill()
		}
		_ = read.Close()
		t.Fatalf("registered process %d survived parent exit", pid)
	}
}
