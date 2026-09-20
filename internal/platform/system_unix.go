//go:build darwin || linux

package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/charlesonunze/nuke/internal/nuke"
)

type system struct{}

func (system) Terminate(ctx context.Context, pid int, signal nuke.Signal) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	osSignal, err := signalFor(signal)
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process: %w", err)
	}
	if err := proc.Signal(osSignal); err != nil {
		return fmt.Errorf("signaling process: %w", err)
	}

	return nil
}

func signalFor(signal nuke.Signal) (os.Signal, error) {
	switch signal {
	case nuke.SignalTerminate:
		return syscall.SIGTERM, nil
	case nuke.SignalKill:
		return syscall.SIGKILL, nil
	default:
		return nil, fmt.Errorf("unsupported signal %q", signal)
	}
}

func normalizeProcessName(value string) string {
	if value == "" {
		return ""
	}
	return filepath.Base(value)
}

func matchPortTarget(port int, target nuke.PortTarget) bool {
	if target.Range != nil {
		return port >= target.Range.Start && port <= target.Range.End
	}
	for _, targetPort := range target.Ports {
		if port == targetPort {
			return true
		}
	}
	return false
}

func dedupePortOwners(owners []nuke.PortOwner) []nuke.PortOwner {
	seen := make(map[string]struct{}, len(owners))
	result := make([]nuke.PortOwner, 0, len(owners))
	for _, owner := range owners {
		key := strconv.Itoa(owner.Port) + "/" + owner.Protocol + "/" + strconv.Itoa(owner.Process.PID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, owner)
	}
	return result
}
