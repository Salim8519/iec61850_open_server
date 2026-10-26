//go:build windows

package labnet

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var sizeOfAttributes = unsafe.Sizeof(windows.SecurityAttributes{})

// ErrNoService means the lab network service is not installed or not running.
var ErrNoService = errors.New("the lab network service is not running; use Repair on the lab network page")

// callTimeout bounds one request; the service answers in milliseconds.
const callTimeout = 10 * time.Second

// InstalledProgram is where the installer copies the service program; only administrators can
// write there.
func InstalledProgram() string {
	folder, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil || folder == "" {
		folder = `C:\Program Files`
	}
	return filepath.Join(folder, "ScadaBench", "LabNetwork", ProgramName)
}

// Call sends one request to the service and returns its answer. It refuses to talk to a pipe
// whose server is not the installed service program.
func Call(request Request) (Reply, error) {
	name, _ := windows.UTF16PtrFromString(PipeName)
	start := time.Now()
	var pipe windows.Handle
	for {
		var err error
		pipe, err = windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
			windows.OPEN_EXISTING, windows.SECURITY_SQOS_PRESENT|windows.SECURITY_IDENTIFICATION, 0)
		if err == nil {
			break
		}
		// Busy, or between two pipe instances of the running service: try again shortly. A
		// missing pipe that stays missing for a second means no service.
		waited := time.Since(start)
		busy := errors.Is(err, windows.ERROR_PIPE_BUSY) && waited < callTimeout
		between := errors.Is(err, windows.ERROR_FILE_NOT_FOUND) && waited < time.Second
		if !busy && !between {
			return Reply{}, ErrNoService
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := checkServer(pipe); err != nil {
		windows.CloseHandle(pipe)
		return Reply{}, err
	}
	mode := uint32(windows.PIPE_READMODE_MESSAGE)
	windows.SetNamedPipeHandleState(pipe, &mode, nil, nil)
	done := make(chan struct{})
	var reply Reply
	var failure error
	go func() {
		defer close(done)
		reply, failure = exchange(pipe, request)
	}()
	select {
	case <-done:
	case <-time.After(callTimeout):
		windows.CloseHandle(pipe) // unblocks the read
		<-done
		return Reply{}, errors.New("the lab network service did not answer")
	}
	windows.CloseHandle(pipe)
	return reply, failure
}

func exchange(pipe windows.Handle, request Request) (Reply, error) {
	data, _ := json.Marshal(request)
	var written uint32
	if err := windows.WriteFile(pipe, append(data, '\n'), &written, nil); err != nil {
		return Reply{}, ErrNoService
	}
	buffer := make([]byte, pipeBuffer)
	var read uint32
	if err := windows.ReadFile(pipe, buffer, &read, nil); err != nil {
		return Reply{}, ErrNoService
	}
	var reply Reply
	if err := json.Unmarshal(trimLine(buffer[:read]), &reply); err != nil {
		return Reply{}, errors.New("the lab network service answered something unreadable")
	}
	return reply, nil
}

// checkServer makes sure the pipe belongs to the running ScadaBench lab network service: the pipe's
// server process must be the process Windows runs that service in, so another program that created
// the same pipe name first never receives the app's requests. (A normal user may not open a SYSTEM
// process to read its path, but anyone may ask the Service Control Manager for its process ID.)
func checkServer(pipe windows.Handle) error {
	var pid uint32
	if err := windows.GetNamedPipeServerProcessId(pipe, &pid); err != nil {
		return ErrNoService
	}
	service, err := servicePID()
	if err != nil || service == 0 {
		return ErrNoService
	}
	if pid != service {
		return errors.New("another program answers on the lab network pipe; nothing was sent")
	}
	return nil
}
