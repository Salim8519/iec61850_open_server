//go:build windows

package labnet

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

// pipeSDDL: SYSTEM and administrators full control, interactive users read and write.
const pipeSDDL = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;IU)"

const (
	pipeRejectRemote = 0x00000008 // PIPE_REJECT_REMOTE_CLIENTS
	pipeBuffer       = 64 * 1024
)

// Serve answers requests on the pipe one at a time until stop is closed. A request is one JSON
// line; the answer is one JSON line.
func Serve(stop <-chan struct{}, handle func(Request) Reply) error {
	descriptor, err := windows.SecurityDescriptorFromString(pipeSDDL)
	if err != nil {
		return err
	}
	attributes := &windows.SecurityAttributes{SecurityDescriptor: descriptor}
	attributes.Length = uint32(sizeOfAttributes)
	name, _ := windows.UTF16PtrFromString(PipeName)
	var stopping atomic.Bool
	var one sync.Mutex
	go func() {
		<-stop
		stopping.Store(true)
		wake() // ConnectNamedPipe below returns once someone connects
	}()
	for !stopping.Load() {
		pipe, err := windows.CreateNamedPipe(name, windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT|pipeRejectRemote,
			windows.PIPE_UNLIMITED_INSTANCES, pipeBuffer, pipeBuffer, 0, attributes)
		if err != nil {
			return err
		}
		err = windows.ConnectNamedPipe(pipe, nil)
		connected := err == nil || errors.Is(err, windows.ERROR_PIPE_CONNECTED)
		// Answer on its own goroutine so the next pipe instance is listening at once; requests
		// still change Windows one at a time.
		go func() {
			if connected && !stopping.Load() {
				one.Lock()
				answer(pipe, handle)
				one.Unlock()
			}
			windows.DisconnectNamedPipe(pipe)
			windows.CloseHandle(pipe)
		}()
	}
	return nil
}

func answer(pipe windows.Handle, handle func(Request) Reply) {
	buffer := make([]byte, maxRequest+1)
	var read uint32
	if err := windows.ReadFile(pipe, buffer, &read, nil); err != nil && !errors.Is(err, windows.ERROR_MORE_DATA) {
		return
	}
	var reply Reply
	if request, err := DecodeRequest(trimLine(buffer[:read])); err != nil {
		reply = Reply{Message: err.Error(), Addresses: []string{}}
	} else {
		reply = handle(request)
	}
	data, _ := json.Marshal(reply)
	var written uint32
	windows.WriteFile(pipe, append(data, '\n'), &written, nil)
	windows.FlushFileBuffers(pipe)
}

func trimLine(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}

// wake connects once so a blocked ConnectNamedPipe returns while the service stops.
func wake() {
	name, _ := windows.UTF16PtrFromString(PipeName)
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, 0, 0)
	if err == nil {
		windows.CloseHandle(handle)
	}
}
