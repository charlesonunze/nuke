package nuke

import "context"

type Command string

const (
	CommandPort Command = "port"
	CommandPID  Command = "pid"
)

type PortMode int

const (
	PortModeNone PortMode = iota
	PortModeExplicit
	PortModeRange
)

type ProcessMode int

const (
	ProcessModeNone ProcessMode = iota
	ProcessModeExplicit
	ProcessModeName
)

type Config struct {
	Command Command
	Help    bool

	Force  bool
	DryRun bool
	Yes    bool

	PortMode  PortMode
	Ports     []int
	PortRange *PortRange

	ProcessMode ProcessMode
	PIDs        []int
	Name        string
}

type PortRange struct {
	Start int
	End   int
}

type Signal string

const (
	SignalTerminate Signal = "TERM"
	SignalKill      Signal = "KILL"
)

type Process struct {
	PID     int
	PPID    int
	User    string
	Name    string
	Command string
}

type PortOwner struct {
	Port     int
	Protocol string
	Process  Process
}

type PortTarget struct {
	Ports []int
	Range *PortRange
}

type ProcessTarget struct {
	PIDs []int
	Name string
}

type System interface {
	FindPortOwners(ctx context.Context, target PortTarget) ([]PortOwner, error)
	FindProcesses(ctx context.Context, target ProcessTarget) ([]Process, error)
	Terminate(ctx context.Context, pid int, signal Signal) error
}
