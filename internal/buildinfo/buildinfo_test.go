package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestStringUsesInjectedValues(t *testing.T) {
	originalVersion, originalCommit, originalDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = originalVersion, originalCommit, originalDate
	})
	version, commit, date = "v2.0.0", "release-commit", "release-date"

	got := String()
	want := "nuke 2.0.0 (commit release-commit, built release-date)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestResolveUsesGoBuildInfoDefaults(t *testing.T) {
	build := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-09-21T02:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	got := resolve(build, true)
	want := info{
		Version: "1.2.3",
		Commit:  "abc123-dirty",
		Date:    "2026-09-21T02:00:00Z",
	}
	if got != want {
		t.Fatalf("resolve() = %+v, want %+v", got, want)
	}
}

func TestResolvePrefersInjectedValues(t *testing.T) {
	originalVersion, originalCommit, originalDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = originalVersion, originalCommit, originalDate
	})
	version, commit, date = "v2.0.0", "release-commit", "release-date"

	build := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-09-21T02:00:00Z"},
		},
	}

	got := resolve(build, true)
	want := info{
		Version: "2.0.0",
		Commit:  "release-commit",
		Date:    "release-date",
	}
	if got != want {
		t.Fatalf("resolve() = %+v, want %+v", got, want)
	}
}
