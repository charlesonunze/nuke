package nuke

import (
	"fmt"
	"strconv"
	"strings"
)

type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

func Parse(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{Help: true}, nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		return Config{Help: true}, nil
	case string(CommandVersion), "-v":
		if len(args) != 1 {
			return Config{}, usageErrorf("%s does not accept arguments", args[0])
		}
		return Config{Command: CommandVersion}, nil
	case string(CommandPort):
		return parsePort(args[1:])
	case string(CommandPID):
		return parsePID(args[1:])
	default:
		return Config{}, usageErrorf("unknown command %q", args[0])
	}
}

func parsePort(args []string) (Config, error) {
	cfg := Config{Command: CommandPort}
	var rangeText string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case isHelpFlag(arg):
			cfg.Help = true
			return cfg, nil
		case parseCommonFlag(arg, &cfg):
			continue
		case arg == "-r" || arg == "--range":
			next, ok := nextValue(args, &i, arg)
			if !ok {
				return Config{}, usageErrorf("missing value for %s", arg)
			}
			if rangeText != "" {
				return Config{}, usageErrorf("range can only be provided once")
			}
			rangeText = next
		case strings.HasPrefix(arg, "--range="):
			if rangeText != "" {
				return Config{}, usageErrorf("range can only be provided once")
			}
			rangeText = strings.TrimPrefix(arg, "--range=")
		case strings.HasPrefix(arg, "-"):
			return Config{}, usageErrorf("unknown flag %q", arg)
		default:
			port, err := parsePortNumber(arg)
			if err != nil {
				return Config{}, err
			}
			cfg.Ports = append(cfg.Ports, port)
		}
	}

	if rangeText != "" && len(cfg.Ports) > 0 {
		return Config{}, usageErrorf("use either explicit ports or --range, not both")
	}
	if rangeText != "" {
		portRange, err := parsePortRange(rangeText)
		if err != nil {
			return Config{}, err
		}
		cfg.PortMode = PortModeRange
		cfg.PortRange = &portRange
		return cfg, nil
	}
	if len(cfg.Ports) == 0 {
		return Config{}, usageErrorf("port requires at least one port or --range")
	}

	cfg.PortMode = PortModeExplicit
	return cfg, nil
}

func parsePID(args []string) (Config, error) {
	cfg := Config{Command: CommandPID}
	var name string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case isHelpFlag(arg):
			cfg.Help = true
			return cfg, nil
		case parseCommonFlag(arg, &cfg):
			continue
		case arg == "-n" || arg == "--name":
			next, ok := nextValue(args, &i, arg)
			if !ok {
				return Config{}, usageErrorf("missing value for %s", arg)
			}
			if name != "" {
				return Config{}, usageErrorf("name can only be provided once")
			}
			name = next
		case strings.HasPrefix(arg, "--name="):
			if name != "" {
				return Config{}, usageErrorf("name can only be provided once")
			}
			name = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "-"):
			return Config{}, usageErrorf("unknown flag %q", arg)
		default:
			pid, err := parsePIDNumber(arg)
			if err != nil {
				return Config{}, err
			}
			cfg.PIDs = append(cfg.PIDs, pid)
		}
	}

	if name != "" && len(cfg.PIDs) > 0 {
		return Config{}, usageErrorf("use either explicit pids or --name, not both")
	}
	if name != "" {
		cfg.ProcessMode = ProcessModeName
		cfg.Name = name
		return cfg, nil
	}
	if len(cfg.PIDs) == 0 {
		return Config{}, usageErrorf("pid requires at least one pid or --name")
	}

	cfg.ProcessMode = ProcessModeExplicit
	return cfg, nil
}

func parseCommonFlag(arg string, cfg *Config) bool {
	switch arg {
	case "--force":
		cfg.Force = true
		return true
	case "--dry-run":
		cfg.DryRun = true
		return true
	case "--yes":
		cfg.Yes = true
		return true
	default:
		return false
	}
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func nextValue(args []string, index *int, flag string) (string, bool) {
	nextIndex := *index + 1
	if nextIndex >= len(args) || strings.HasPrefix(args[nextIndex], "-") {
		return "", false
	}
	*index = nextIndex
	return args[nextIndex], true
}

func parsePortNumber(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, usageErrorf("invalid port %q: must be between 1 and 65535", value)
	}
	return port, nil
}

func parsePIDNumber(value string) (int, error) {
	pid, err := strconv.Atoi(value)
	if err != nil || pid < 1 {
		return 0, usageErrorf("invalid pid %q: must be a positive integer", value)
	}
	return pid, nil
}

func parsePortRange(value string) (PortRange, error) {
	startText, endText, ok := strings.Cut(value, "-")
	if !ok || startText == "" || endText == "" || strings.Contains(endText, "-") {
		return PortRange{}, usageErrorf("invalid range %q: expected start-end", value)
	}

	start, err := parsePortNumber(startText)
	if err != nil {
		return PortRange{}, err
	}
	end, err := parsePortNumber(endText)
	if err != nil {
		return PortRange{}, err
	}
	if start > end {
		return PortRange{}, usageErrorf("invalid range %q: start must be less than or equal to end", value)
	}

	return PortRange{Start: start, End: end}, nil
}

func usageErrorf(format string, args ...any) UsageError {
	return UsageError{Message: fmt.Sprintf(format, args...)}
}
