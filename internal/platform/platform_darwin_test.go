//go:build darwin

package platform

import "testing"

func TestParsePSLineUsesCommandExecutableForName(t *testing.T) {
	proc, ok := parsePSLine("1007 999 charlesonunze /opt/homebrew/Cellar/python@3.14/3.14.6/Frameworks/Python.framework/Versions/3.14/Resources/Python.app/Contents/MacOS/Python -m http.server 54321")
	if !ok {
		t.Fatal("parsePSLine() ok = false, want true")
	}
	if proc.Name != "Python" {
		t.Fatalf("Name = %q, want %q", proc.Name, "Python")
	}
	if proc.Command != "/opt/homebrew/Cellar/python@3.14/3.14.6/Frameworks/Python.framework/Versions/3.14/Resources/Python.app/Contents/MacOS/Python -m http.server 54321" {
		t.Fatalf("Command = %q", proc.Command)
	}
}

func TestExtractPortFromName(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "*:3000 (LISTEN)", want: 3000},
		{name: "127.0.0.1:5432 (LISTEN)", want: 5432},
		{name: "[::1]:8080", want: 8080},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPortFromName(tt.name)
			if !ok {
				t.Fatal("extractPortFromName() ok = false, want true")
			}
			if got != tt.want {
				t.Fatalf("port = %d, want %d", got, tt.want)
			}
		})
	}
}
