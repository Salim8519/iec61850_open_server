package desktop

import "os/exec"

// process is a started browser as the window watcher sees it.
type process interface {
	PID() int
	Wait()        // blocks until the process has exited
	Exited() bool // Popen.poll() is not None
}

// browserProcess is a browser started by startBrowser; one goroutine reaps it.
type browserProcess struct {
	cmd  *exec.Cmd
	done chan struct{}
}

// startBrowser is the Popen of app_window.open_app_window: a new process group and the
// null device for stdin, stdout and stderr (Python left stdin inherited; the windowed
// EXE has none).
func startBrowser(command []string) (process, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.SysProcAttr = newProcessGroup()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	started := &browserProcess{cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(started.done)
	}()
	return started, nil
}

func (p *browserProcess) PID() int { return p.cmd.Process.Pid }

func (p *browserProcess) Wait() { <-p.done }

func (p *browserProcess) Exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}
