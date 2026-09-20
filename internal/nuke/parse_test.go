package nuke

import (
	"errors"
	"testing"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "single port with trailing force",
			args: []string{"port", "3000", "--force"},
			want: Config{Command: CommandPort, Force: true, PortMode: PortModeExplicit, Ports: []int{3000}},
		},
		{
			name: "multiple ports",
			args: []string{"port", "3000", "5173", "8080"},
			want: Config{Command: CommandPort, PortMode: PortModeExplicit, Ports: []int{3000, 5173, 8080}},
		},
		{
			name: "short range",
			args: []string{"port", "-r", "3000-3999", "--dry-run"},
			want: Config{Command: CommandPort, DryRun: true, PortMode: PortModeRange, PortRange: &PortRange{Start: 3000, End: 3999}},
		},
		{
			name: "long range equals",
			args: []string{"port", "--range=8000-8010", "--yes"},
			want: Config{Command: CommandPort, Yes: true, PortMode: PortModeRange, PortRange: &PortRange{Start: 8000, End: 8010}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			assertConfig(t, got, tt.want)
		})
	}
}

func TestParsePID(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "single pid",
			args: []string{"pid", "1234"},
			want: Config{Command: CommandPID, ProcessMode: ProcessModeExplicit, PIDs: []int{1234}},
		},
		{
			name: "multiple pids with trailing force",
			args: []string{"pid", "1234", "5678", "--force"},
			want: Config{Command: CommandPID, Force: true, ProcessMode: ProcessModeExplicit, PIDs: []int{1234, 5678}},
		},
		{
			name: "short name",
			args: []string{"pid", "-n", "main"},
			want: Config{Command: CommandPID, ProcessMode: ProcessModeName, Name: "main"},
		},
		{
			name: "long name equals",
			args: []string{"pid", "--name=main", "--yes"},
			want: Config{Command: CommandPID, Yes: true, ProcessMode: ProcessModeName, Name: "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			assertConfig(t, got, tt.want)
		})
	}
}

func TestParseRejectsMixedTargetModes(t *testing.T) {
	tests := [][]string{
		{"port", "3000", "-r", "4000-5000"},
		{"port", "3000", "--range=4000-5000"},
		{"pid", "1234", "-n", "main"},
		{"pid", "1234", "--name=main"},
	}

	for _, args := range tests {
		t.Run(joinArgs(args), func(t *testing.T) {
			_, err := Parse(args)
			var usageErr UsageError
			if !errors.As(err, &usageErr) {
				t.Fatalf("Parse() error = %T %v, want UsageError", err, err)
			}
		})
	}
}

func TestParseRejectsUnknownJSONFlag(t *testing.T) {
	_, err := Parse([]string{"port", "3000", "--json"})
	var usageErr UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("Parse() error = %T %v, want UsageError", err, err)
	}
}

func assertConfig(t *testing.T, got Config, want Config) {
	t.Helper()

	if got.Command != want.Command ||
		got.Help != want.Help ||
		got.Force != want.Force ||
		got.DryRun != want.DryRun ||
		got.Yes != want.Yes ||
		got.PortMode != want.PortMode ||
		got.ProcessMode != want.ProcessMode ||
		got.Name != want.Name {
		t.Fatalf("Config scalar fields mismatch\ngot:  %+v\nwant: %+v", got, want)
	}

	assertInts(t, got.Ports, want.Ports, "Ports")
	assertInts(t, got.PIDs, want.PIDs, "PIDs")

	switch {
	case got.PortRange == nil && want.PortRange == nil:
	case got.PortRange != nil && want.PortRange != nil && *got.PortRange == *want.PortRange:
	default:
		t.Fatalf("PortRange = %+v, want %+v", got.PortRange, want.PortRange)
	}
}

func assertInts(t *testing.T, got []int, want []int, field string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", field, len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %d, want %d", field, i, got[i], want[i])
		}
	}
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}
