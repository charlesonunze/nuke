//go:build darwin

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charlesonunze/nuke/internal/nuke"
)

const (
	macOSLsof = "/usr/sbin/lsof"
	macOSPS   = "/bin/ps"
)

func newSystem() (nuke.System, error) {
	return system{}, nil
}

func (system) FindProcesses(ctx context.Context, target nuke.ProcessTarget) ([]nuke.Process, error) {
	processes, err := psProcesses(ctx)
	if err != nil {
		return nil, err
	}

	if target.Name != "" {
		matches := make([]nuke.Process, 0)
		for _, proc := range processes {
			if proc.Name == target.Name {
				matches = append(matches, proc)
			}
		}
		return matches, nil
	}

	pids := make(map[int]struct{}, len(target.PIDs))
	for _, pid := range target.PIDs {
		pids[pid] = struct{}{}
	}

	matches := make([]nuke.Process, 0, len(target.PIDs))
	for _, proc := range processes {
		if _, ok := pids[proc.PID]; ok {
			matches = append(matches, proc)
		}
	}
	return matches, nil
}

func (system) FindPortOwners(ctx context.Context, target nuke.PortTarget) ([]nuke.PortOwner, error) {
	cmd := exec.CommandContext(ctx, macOSLsof, "-nP", "-iTCP", "-sTCP:LISTEN", "-iUDP")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 && len(output) == 0 {
			return nil, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("required native macOS command %s was not found", macOSLsof)
		}
		return nil, fmt.Errorf("running lsof: %w", err)
	}

	processes, err := psProcesses(ctx)
	if err != nil {
		return nil, err
	}
	processByPID := make(map[int]nuke.Process, len(processes))
	for _, proc := range processes {
		processByPID[proc.PID] = proc
	}

	owners := make([]nuke.PortOwner, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] {
		owner, ok := parseLsofLine(line, processByPID)
		if !ok || !matchPortTarget(owner.Port, target) {
			continue
		}
		owners = append(owners, owner)
	}

	return dedupePortOwners(owners), nil
}

func psProcesses(ctx context.Context) ([]nuke.Process, error) {
	cmd := exec.CommandContext(ctx, macOSPS, "-axo", "pid=,ppid=,user=,command=")
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("required native macOS command %s was not found", macOSPS)
		}
		return nil, fmt.Errorf("running ps: %w", err)
	}

	processes := make([]nuke.Process, 0)
	for _, line := range strings.Split(string(output), "\n") {
		proc, ok := parsePSLine(line)
		if ok {
			processes = append(processes, proc)
		}
	}
	return processes, nil
}

func parsePSLine(line string) (nuke.Process, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nuke.Process{}, false
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return nuke.Process{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nuke.Process{}, false
	}

	command := strings.Join(fields[3:], " ")
	name := normalizeProcessName(fields[3])

	return nuke.Process{
		PID:     pid,
		PPID:    ppid,
		User:    fields[2],
		Name:    name,
		Command: command,
	}, true
}

func parseLsofLine(line string, processByPID map[int]nuke.Process) (nuke.PortOwner, bool) {
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nuke.PortOwner{}, false
	}

	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nuke.PortOwner{}, false
	}

	nameColumn := strings.Join(fields[8:], " ")
	port, ok := extractPortFromName(nameColumn)
	if !ok {
		return nuke.PortOwner{}, false
	}

	protocol := strings.ToLower(fields[7])
	if protocol != "tcp" && protocol != "udp" {
		return nuke.PortOwner{}, false
	}

	proc, ok := processByPID[pid]
	if !ok {
		proc = nuke.Process{
			PID:     pid,
			User:    fields[2],
			Name:    normalizeProcessName(fields[0]),
			Command: fields[0],
		}
	}

	return nuke.PortOwner{
		Port:     port,
		Protocol: protocol,
		Process:  proc,
	}, true
}

func extractPortFromName(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if idx := strings.IndexByte(value, ' '); idx >= 0 {
		value = value[:idx]
	}

	colon := strings.LastIndexByte(value, ':')
	if colon < 0 || colon == len(value)-1 {
		return 0, false
	}

	portText := value[colon+1:]
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return 0, false
	}
	return port, true
}
