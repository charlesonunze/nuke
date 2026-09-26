package buildinfo

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type info struct {
	Version string
	Commit  string
	Date    string
}

// String returns the version and build metadata for the current executable.
func String() string {
	build, ok := debug.ReadBuildInfo()
	current := resolve(build, ok)
	return fmt.Sprintf("nuke %s (commit %s, built %s)", current.Version, current.Commit, current.Date)
}

func resolve(build *debug.BuildInfo, ok bool) info {
	current := info{
		Version: normalizeVersion(version),
		Commit:  commit,
		Date:    date,
	}
	if !ok || build == nil {
		return current
	}

	if version == "dev" && build.Main.Version != "" && build.Main.Version != "(devel)" {
		current.Version = normalizeVersion(build.Main.Version)
	}

	settings := make(map[string]string, len(build.Settings))
	for _, setting := range build.Settings {
		settings[setting.Key] = setting.Value
	}
	if commit == "none" && settings["vcs.revision"] != "" {
		current.Commit = settings["vcs.revision"]
		if settings["vcs.modified"] == "true" {
			current.Commit += "-dirty"
		}
	}
	if date == "unknown" && settings["vcs.time"] != "" {
		current.Date = settings["vcs.time"]
	}

	return current
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(value, "v")
}
