//go:build linux

package platform

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charlesonunze/nuke/internal/nuke"
)

func newSystem() (nuke.System, error) {
	return system{}, nil
}

func (system) FindProcesses(ctx context.Context, target nuke.ProcessTarget) ([]nuke.Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pids, err := targetPIDs(target)
	if err != nil {
		return nil, err
	}

	processes := make([]nuke.Process, 0, len(pids))
	userCache := make(map[string]string)
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		proc, ok := readLinuxProcess(pid, userCache)
		if !ok {
			continue
		}
		if target.Name != "" && proc.Name != target.Name {
			continue
		}
		processes = append(processes, proc)
	}

	return processes, nil
}

func (system) FindPortOwners(ctx context.Context, target nuke.PortTarget) ([]nuke.PortOwner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sockets, err := targetSockets(target)
	if err != nil {
		return nil, err
	}
	if len(sockets) == 0 {
		return nil, nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	owners := make([]nuke.PortOwner, 0)
	userCache := make(map[string]string)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		proc, ok := readLinuxProcess(pid, userCache)
		if !ok {
			continue
		}

		socketOwners := socketsForPID(pid, sockets)
		for _, socketOwner := range socketOwners {
			owners = append(owners, nuke.PortOwner{
				Port:     socketOwner.Port,
				Protocol: socketOwner.Protocol,
				Process:  proc,
			})
		}
	}

	return dedupePortOwners(owners), nil
}

type socketOwner struct {
	Port     int
	Protocol string
}

func targetPIDs(target nuke.ProcessTarget) ([]int, error) {
	if len(target.PIDs) > 0 {
		return target.PIDs, nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	pids := make([]int, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err == nil {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

func targetSockets(target nuke.PortTarget) (map[string][]socketOwner, error) {
	sockets := make(map[string][]socketOwner)
	tables := []struct {
		Path     string
		Protocol string
		TCP      bool
	}{
		{Path: "/proc/net/tcp", Protocol: "tcp", TCP: true},
		{Path: "/proc/net/tcp6", Protocol: "tcp6", TCP: true},
		{Path: "/proc/net/udp", Protocol: "udp", TCP: false},
		{Path: "/proc/net/udp6", Protocol: "udp6", TCP: false},
	}

	for _, table := range tables {
		if err := readSocketTable(table.Path, table.Protocol, table.TCP, target, sockets); err != nil {
			return nil, err
		}
	}

	return sockets, nil
}

func readSocketTable(path string, protocol string, tcp bool, target nuke.PortTarget, sockets map[string][]socketOwner) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		if tcp && fields[3] != "0A" {
			continue
		}

		port, ok := parseProcNetPort(fields[1])
		if !ok || !matchPortTarget(port, target) {
			continue
		}

		inode := fields[9]
		sockets[inode] = append(sockets[inode], socketOwner{Port: port, Protocol: protocol})
	}

	return nil
}

func parseProcNetPort(localAddress string) (int, bool) {
	_, portText, ok := strings.Cut(localAddress, ":")
	if !ok {
		return 0, false
	}
	port64, err := strconv.ParseInt(portText, 16, 32)
	if err != nil || port64 < 1 || port64 > 65535 {
		return 0, false
	}
	return int(port64), true
}

func socketsForPID(pid int, sockets map[string][]socketOwner) []socketOwner {
	fdDir := filepath.Join("/proc", strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil
	}

	owners := make([]socketOwner, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		linkPath := filepath.Join(fdDir, entry.Name())
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}

		inode, ok := socketInode(target)
		if !ok {
			continue
		}
		socketOwners, ok := sockets[inode]
		if !ok {
			continue
		}

		for _, owner := range socketOwners {
			key := owner.Protocol + "/" + strconv.Itoa(owner.Port)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			owners = append(owners, owner)
		}
	}

	return owners
}

func socketInode(value string) (string, bool) {
	if !strings.HasPrefix(value, "socket:[") || !strings.HasSuffix(value, "]") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(value, "socket:["), "]"), true
}

func readLinuxProcess(pid int, userCache map[string]string) (nuke.Process, bool) {
	status, err := readStatus(pid)
	if err != nil {
		return nuke.Process{}, false
	}

	command := readCmdline(pid)
	if command == "" {
		command = status.Name
	}

	return nuke.Process{
		PID:     pid,
		PPID:    status.PPID,
		User:    usernameForUID(status.UID, userCache),
		Name:    status.Name,
		Command: command,
	}, true
}

type linuxStatus struct {
	Name string
	PPID int
	UID  string
}

func readStatus(pid int) (linuxStatus, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return linuxStatus{}, err
	}

	var status linuxStatus
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		value = strings.TrimSpace(value)
		switch key {
		case "Name":
			status.Name = normalizeProcessName(value)
		case "PPid":
			ppid, err := strconv.Atoi(value)
			if err == nil {
				status.PPID = ppid
			}
		case "Uid":
			fields := strings.Fields(value)
			if len(fields) > 0 {
				status.UID = fields[0]
			}
		}
	}
	if status.Name == "" {
		return linuxStatus{}, fs.ErrNotExist
	}
	return status, nil
}

func readCmdline(pid int) string {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(data) == 0 {
		return ""
	}

	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	return strings.Join(parts, " ")
}

func usernameForUID(uid string, cache map[string]string) string {
	if uid == "" {
		return ""
	}
	if name, ok := cache[uid]; ok {
		return name
	}

	name := uid
	if u, err := user.LookupId(uid); err == nil && u.Username != "" {
		name = u.Username
	}
	cache[uid] = name
	return name
}
