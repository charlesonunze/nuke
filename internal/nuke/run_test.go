package nuke

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	for _, args := range [][]string{{"-v"}, {"version"}} {
		t.Run(joinArgs(args), func(t *testing.T) {
			var out bytes.Buffer
			code := Run(context.Background(), nil, args, strings.NewReader(""), &out, &bytes.Buffer{})

			if code != 0 {
				t.Fatalf("Run() code = %d, want 0", code)
			}
			if !strings.HasPrefix(out.String(), "nuke ") {
				t.Fatalf("output = %q, want version output", out.String())
			}
		})
	}
}

func TestRunPIDNamePromptsAndDeclinesByDefault(t *testing.T) {
	sys := &fakeSystem{
		processes: []Process{
			{PID: 10, PPID: 1, User: "me", Name: "main", Command: "./main"},
			{PID: 11, PPID: 1, User: "me", Name: "main", Command: "/tmp/main"},
		},
	}

	var out bytes.Buffer
	code := Run(context.Background(), sys, []string{"pid", "-n", "main"}, strings.NewReader("\n"), &out, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if len(sys.terminated) != 0 {
		t.Fatalf("terminated = %+v, want none", sys.terminated)
	}
	if !strings.Contains(out.String(), "no processes terminated") {
		t.Fatalf("output = %q, want decline message", out.String())
	}
}

func TestRunPIDNameYesTerminatesMatches(t *testing.T) {
	sys := &fakeSystem{
		processes: []Process{
			{PID: 10, PPID: 1, User: "me", Name: "main", Command: "./main"},
			{PID: 11, PPID: 1, User: "me", Name: "main", Command: "/tmp/main"},
		},
	}

	code := Run(context.Background(), sys, []string{"pid", "-n", "main", "--yes"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	assertTerminated(t, sys.terminated, []termination{{PID: 10, Signal: SignalTerminate}, {PID: 11, Signal: SignalTerminate}})
}

func TestRunPortRangeDryRunDoesNotTerminate(t *testing.T) {
	sys := &fakeSystem{
		portOwners: []PortOwner{
			{Port: 3000, Protocol: "tcp", Process: Process{PID: 10, User: "me", Name: "node", Command: "node server.js"}},
		},
	}

	var out bytes.Buffer
	code := Run(context.Background(), sys, []string{"port", "-r", "3000-3999", "--dry-run"}, strings.NewReader(""), &out, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if len(sys.terminated) != 0 {
		t.Fatalf("terminated = %+v, want none", sys.terminated)
	}
	if !strings.Contains(out.String(), "dry run: would send TERM to 1 process") {
		t.Fatalf("output = %q, want dry-run message", out.String())
	}
}

func TestRunPIDForceUsesKill(t *testing.T) {
	sys := &fakeSystem{
		processes: []Process{{PID: 10, PPID: 1, User: "me", Name: "main", Command: "./main"}},
	}

	code := Run(context.Background(), sys, []string{"pid", "10", "--force"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	assertTerminated(t, sys.terminated, []termination{{PID: 10, Signal: SignalKill}})
}

func TestRunRefusesProtectedPID(t *testing.T) {
	sys := &fakeSystem{
		processes: []Process{{PID: 1, PPID: 0, User: "root", Name: "launchd", Command: "/sbin/launchd"}},
	}

	var errOut bytes.Buffer
	code := Run(context.Background(), sys, []string{"pid", "1", "--yes"}, strings.NewReader(""), &bytes.Buffer{}, &errOut)

	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if len(sys.terminated) != 0 {
		t.Fatalf("terminated = %+v, want none", sys.terminated)
	}
	if !strings.Contains(errOut.String(), "protected system pid") {
		t.Fatalf("stderr = %q, want protected message", errOut.String())
	}
}

type fakeSystem struct {
	portOwners []PortOwner
	processes  []Process
	terminate  error
	terminated []termination
}

type termination struct {
	PID    int
	Signal Signal
}

func (s *fakeSystem) FindPortOwners(_ context.Context, target PortTarget) ([]PortOwner, error) {
	matches := make([]PortOwner, 0)
	for _, owner := range s.portOwners {
		if target.Range != nil {
			if owner.Port >= target.Range.Start && owner.Port <= target.Range.End {
				matches = append(matches, owner)
			}
			continue
		}
		for _, port := range target.Ports {
			if owner.Port == port {
				matches = append(matches, owner)
			}
		}
	}
	return matches, nil
}

func (s *fakeSystem) FindProcesses(_ context.Context, target ProcessTarget) ([]Process, error) {
	matches := make([]Process, 0)
	for _, proc := range s.processes {
		if target.Name != "" {
			if proc.Name == target.Name {
				matches = append(matches, proc)
			}
			continue
		}
		for _, pid := range target.PIDs {
			if proc.PID == pid {
				matches = append(matches, proc)
			}
		}
	}
	return matches, nil
}

func (s *fakeSystem) Terminate(_ context.Context, pid int, signal Signal) error {
	if s.terminate != nil {
		return fmt.Errorf("fake terminate: %w", s.terminate)
	}
	s.terminated = append(s.terminated, termination{PID: pid, Signal: signal})
	return nil
}

func assertTerminated(t *testing.T, got []termination, want []termination) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("terminated length = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("terminated[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFakeSystemTerminateErrorIsWrapped(t *testing.T) {
	sys := &fakeSystem{
		processes: []Process{{PID: 10, PPID: 1, User: "me", Name: "main", Command: "./main"}},
		terminate: errors.New("denied"),
	}

	var errOut bytes.Buffer
	code := Run(context.Background(), sys, []string{"pid", "10"}, strings.NewReader(""), &bytes.Buffer{}, &errOut)

	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "fake terminate: denied") {
		t.Fatalf("stderr = %q, want terminate error", errOut.String())
	}
}
