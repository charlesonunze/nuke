package nuke

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/charlesonunze/nuke/internal/buildinfo"
)

func Run(ctx context.Context, sys System, args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	cfg, err := Parse(args)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n\n%s", err, Usage())
		return 2
	}
	if cfg.Help {
		fmt.Fprint(out, Usage())
		return 0
	}

	switch cfg.Command {
	case CommandVersion:
		fmt.Fprintln(out, buildinfo.String())
		return 0
	case CommandPort:
		return runPort(ctx, sys, cfg, in, out, errOut)
	case CommandPID:
		return runPID(ctx, sys, cfg, in, out, errOut)
	default:
		fmt.Fprintf(errOut, "error: unsupported command %q\n", cfg.Command)
		return 2
	}
}

func runPort(ctx context.Context, sys System, cfg Config, in io.Reader, out io.Writer, errOut io.Writer) int {
	owners, err := sys.FindPortOwners(ctx, PortTarget{Ports: cfg.Ports, Range: cfg.PortRange})
	if err != nil {
		fmt.Fprintf(errOut, "error: finding port owners: %v\n", err)
		return 1
	}
	if len(owners) == 0 {
		fmt.Fprintf(out, "no processes found for %s\n", describePortTarget(cfg))
		return 0
	}

	sortPortOwners(owners)
	writePortOwners(out, owners)

	processes := uniqueProcessesFromPorts(owners)
	return terminateMatches(ctx, sys, cfg, in, out, errOut, processes, confirmationPolicy{
		PromptForAny: cfg.PortMode == PortModeRange,
		PromptPlural: true,
	})
}

func runPID(ctx context.Context, sys System, cfg Config, in io.Reader, out io.Writer, errOut io.Writer) int {
	processes, err := sys.FindProcesses(ctx, ProcessTarget{PIDs: cfg.PIDs, Name: cfg.Name})
	if err != nil {
		fmt.Fprintf(errOut, "error: finding processes: %v\n", err)
		return 1
	}
	if len(processes) == 0 {
		fmt.Fprintf(out, "no processes found for %s\n", describeProcessTarget(cfg))
		return 0
	}

	sortProcesses(processes)
	writeProcesses(out, processes)

	return terminateMatches(ctx, sys, cfg, in, out, errOut, uniqueProcesses(processes), confirmationPolicy{
		PromptForAny: cfg.ProcessMode == ProcessModeName,
		PromptPlural: true,
	})
}

type confirmationPolicy struct {
	PromptForAny bool
	PromptPlural bool
}

func terminateMatches(ctx context.Context, sys System, cfg Config, in io.Reader, out io.Writer, errOut io.Writer, processes []Process, policy confirmationPolicy) int {
	killable, refused := partitionProtected(processes)
	for _, item := range refused {
		fmt.Fprintf(errOut, "refusing pid %d (%s): %s\n", item.Process.PID, displayProcessName(item.Process), item.Reason)
	}

	if len(killable) == 0 {
		fmt.Fprintln(out, "no processes terminated")
		if len(refused) > 0 {
			return 1
		}
		return 0
	}

	signal := SignalTerminate
	if cfg.Force {
		signal = SignalKill
	}

	if cfg.DryRun {
		fmt.Fprintf(out, "dry run: would send %s to %d process%s\n", signal, len(killable), plural(len(killable)))
		return 0
	}

	if !cfg.Yes && shouldConfirm(len(killable), policy) && !confirm(in, out, confirmationPrompt(len(killable), signal)) {
		fmt.Fprintln(out, "no processes terminated")
		return 0
	}

	exitCode := 0
	for _, proc := range killable {
		if err := sys.Terminate(ctx, proc.PID, signal); err != nil {
			fmt.Fprintf(errOut, "failed to terminate pid %d (%s): %v\n", proc.PID, displayProcessName(proc), err)
			exitCode = 1
			continue
		}
		fmt.Fprintf(out, "sent %s to pid %d (%s)\n", signal, proc.PID, displayProcessName(proc))
	}
	if len(refused) > 0 {
		exitCode = 1
	}

	return exitCode
}

func shouldConfirm(count int, policy confirmationPolicy) bool {
	if count == 0 {
		return false
	}
	if policy.PromptForAny {
		return true
	}
	return policy.PromptPlural && count > 1
}

func confirmationPrompt(count int, signal Signal) string {
	if count == 1 {
		return fmt.Sprintf("send %s to this process?", signal)
	}
	return fmt.Sprintf("send %s to all %d processes?", signal, count)
}

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N] ", prompt)
	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil && len(answer) == 0 {
		fmt.Fprintln(out)
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

type refusal struct {
	Process Process
	Reason  string
}

func partitionProtected(processes []Process) ([]Process, []refusal) {
	killable := make([]Process, 0, len(processes))
	refused := make([]refusal, 0)
	currentPID := os.Getpid()

	for _, proc := range processes {
		switch {
		case proc.PID <= 1:
			refused = append(refused, refusal{Process: proc, Reason: "protected system pid"})
		case proc.PID == currentPID:
			refused = append(refused, refusal{Process: proc, Reason: "refusing to terminate nuke itself"})
		default:
			killable = append(killable, proc)
		}
	}

	return killable, refused
}

func writePortOwners(out io.Writer, owners []PortOwner) {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "PORT\tPROTO\tPID\tUSER\tNAME\tCOMMAND")
	for _, owner := range owners {
		proc := owner.Process
		fmt.Fprintf(writer, "%d\t%s\t%d\t%s\t%s\t%s\n", owner.Port, owner.Protocol, proc.PID, proc.User, displayProcessName(proc), proc.Command)
	}
	_ = writer.Flush()
}

func writeProcesses(out io.Writer, processes []Process) {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "PID\tPPID\tUSER\tNAME\tCOMMAND")
	for _, proc := range processes {
		fmt.Fprintf(writer, "%d\t%d\t%s\t%s\t%s\n", proc.PID, proc.PPID, proc.User, displayProcessName(proc), proc.Command)
	}
	_ = writer.Flush()
}

func uniqueProcessesFromPorts(owners []PortOwner) []Process {
	processes := make([]Process, 0, len(owners))
	for _, owner := range owners {
		processes = append(processes, owner.Process)
	}
	return uniqueProcesses(processes)
}

func uniqueProcesses(processes []Process) []Process {
	seen := make(map[int]struct{}, len(processes))
	unique := make([]Process, 0, len(processes))
	for _, proc := range processes {
		if _, ok := seen[proc.PID]; ok {
			continue
		}
		seen[proc.PID] = struct{}{}
		unique = append(unique, proc)
	}
	sortProcesses(unique)
	return unique
}

func sortPortOwners(owners []PortOwner) {
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].Port != owners[j].Port {
			return owners[i].Port < owners[j].Port
		}
		if owners[i].Protocol != owners[j].Protocol {
			return owners[i].Protocol < owners[j].Protocol
		}
		return owners[i].Process.PID < owners[j].Process.PID
	})
}

func sortProcesses(processes []Process) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
}

func displayProcessName(proc Process) string {
	if proc.Name != "" {
		return proc.Name
	}
	return "unknown"
}

func describePortTarget(cfg Config) string {
	if cfg.PortRange != nil {
		return fmt.Sprintf("ports %d-%d", cfg.PortRange.Start, cfg.PortRange.End)
	}
	if len(cfg.Ports) == 1 {
		return fmt.Sprintf("port %d", cfg.Ports[0])
	}
	return fmt.Sprintf("ports %s", joinInts(cfg.Ports))
}

func describeProcessTarget(cfg Config) string {
	if cfg.Name != "" {
		return fmt.Sprintf("name %q", cfg.Name)
	}
	if len(cfg.PIDs) == 1 {
		return fmt.Sprintf("pid %d", cfg.PIDs[0])
	}
	return fmt.Sprintf("pids %s", joinInts(cfg.PIDs))
}

func joinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "es"
}
