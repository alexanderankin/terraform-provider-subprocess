package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type Supervisor struct {
	once sync.Once
	mu   sync.Mutex
	pipe *os.File
	err  error
}

var supervisor Supervisor

// BecomeSupervisor receives only PIDs of commands the provider successfully started.
// EOF means the provider is gone, so any registered processes must be killed.
func BecomeSupervisor() int {
	pids := make(map[int]struct{})
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		pid, err := strconv.Atoi(line[1:])
		if err != nil || pid <= 0 {
			continue
		}
		switch line[0] {
		case '+':
			pids[pid] = struct{}{}
		case '-':
			delete(pids, pid)
		}
	}
	for pid := range pids {
		if process, err := os.FindProcess(pid); err == nil {
			_ = process.Kill()
		}
	}
	return 0
}

func EnsureSupervisor() (*Supervisor, error) {
	supervisor.once.Do(func() {
		executable, err := os.Executable()
		if err != nil {
			supervisor.err = err
			return
		}
		read, write, err := os.Pipe()
		if err != nil {
			supervisor.err = err
			return
		}
		cmd := exec.Command(executable, "--subprocess-supervisor") // #nosec G204 -- executable is this provider binary
		cmd.Stdin = read
		if err := cmd.Start(); err != nil {
			_ = read.Close()
			_ = write.Close()
			supervisor.err = err
			return
		}
		_ = read.Close()
		go func() { _ = cmd.Wait() }()
		supervisor.pipe = write
	})
	return &supervisor, supervisor.err
}

func (s *Supervisor) UpdateSupervisor(pid int, add bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	op := '-'
	if add {
		op = '+'
	}
	_, err := fmt.Fprintf(s.pipe, "%c%d\n", op, pid)
	return err
}
