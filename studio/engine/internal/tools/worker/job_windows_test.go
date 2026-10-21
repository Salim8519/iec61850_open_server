//go:build windows

package worker

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// jobRole turns the test binary into a fake child: "sleep" idles like a worker that never
// exits by itself; "server" spawns a sleeper through Spawn, prints its pid and idles.
const jobRole = "STUDIO_TOOLS_JOB_TEST_ROLE"

const (
	synchronize             = 0x00100000
	queryLimitedInformation = 0x1000
)

func TestMain(m *testing.M) {
	switch os.Getenv(jobRole) {
	case "sleep":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "server":
		self, _ := os.Executable()
		os.Setenv(jobRole, "sleep")
		child, err := Spawn(self, nil)
		if err != nil {
			fmt.Println("spawn failed:", err)
			os.Exit(2)
		}
		fmt.Println(child.(*process).command.Process.Pid)
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func sleeper(t *testing.T, role string) *exec.Cmd {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(self, "-test.run=^$")
	command.Env = append(os.Environ(), jobRole+"="+role)
	return command
}

// watch opens pid for waiting before anything is killed, so a reused pid cannot fool the test.
func watch(t *testing.T, pid int) syscall.Handle {
	handle, err := syscall.OpenProcess(synchronize|queryLimitedInformation|processTerminate, false, uint32(pid))
	if err != nil {
		t.Fatalf("open %d: %v", pid, err)
	}
	t.Cleanup(func() { syscall.CloseHandle(handle) })
	return handle
}

func exitsWithin(handle syscall.Handle, limit time.Duration) bool {
	event, _ := syscall.WaitForSingleObject(handle, uint32(limit/time.Millisecond))
	return event == syscall.WAIT_OBJECT_0
}

func TestJobCloseKillsMember(t *testing.T) {
	job, err := newKillOnCloseJob()
	if err != nil {
		t.Fatal(err)
	}
	command := sleeper(t, "sleep")
	if err := command.Start(); err != nil {
		syscall.CloseHandle(job)
		t.Fatal(err)
	}
	defer command.Wait()
	defer command.Process.Kill()
	child := watch(t, command.Process.Pid)
	if err := assign(job, command.Process.Pid); err != nil {
		syscall.CloseHandle(job)
		t.Fatalf("assign: %v", err)
	}
	if exitsWithin(child, 200*time.Millisecond) {
		t.Fatal("child exited before the job closed")
	}
	syscall.CloseHandle(job)
	if !exitsWithin(child, 5*time.Second) {
		t.Fatal("child still running after its kill-on-close job was closed")
	}
}

// TestHardKillLeavesNoChildren kills a server-like process (TerminateProcess, no cleanup)
// and checks that the worker it started through Spawn dies with it.
func TestHardKillLeavesNoChildren(t *testing.T) {
	server := sleeper(t, "server")
	stdout, err := server.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Wait()
	defer server.Process.Kill()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	pid, convErr := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || convErr != nil {
		t.Fatalf("server said %q (%v)", line, err)
	}
	child := watch(t, pid)
	if exitsWithin(child, 200*time.Millisecond) {
		t.Fatal("worker exited while the server was alive")
	}
	if err := server.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if !exitsWithin(child, 5*time.Second) {
		syscall.TerminateProcess(child, 1)
		t.Fatal("worker outlived the hard-killed server")
	}
}
