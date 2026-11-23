package version

import (
	"runtime/debug"
	"strings"
)

var Version = "dev"

var readBuildInfo = func() (*debug.BuildInfo, bool) {
	return debug.ReadBuildInfo()
}

type Info struct {
	Version    string `json:"version"`
	Release    string `json:"release_version,omitempty"`
	Revision   string `json:"git_sha,omitempty"`
	ShortSHA   string `json:"revision,omitempty"`
	Modified   bool   `json:"modified,omitempty"`
	GoVersion  string `json:"go_version,omitempty"`
	ModulePath string `json:"module_path,omitempty"`
}

// Current returns the version string for display and wire metadata.
// Prefer the short VCS revision for the checked-out HEAD whenever Go embedded
// VCS metadata is available. Version remains a fallback for builds without VCS
// metadata.
func Current() string {
	if revision := vcsRevision(); revision != "" {
		return revision
	}
	return releaseVersion()
}

// releaseVersion is the -X stamped Version, or for an unstamped build the
// module version Go records on `go install module@version`.
func releaseVersion() string {
	version := strings.TrimSpace(Version)
	if version != "dev" && version != "" {
		return version
	}
	if info, ok := readBuildInfo(); ok && info != nil {
		if module := strings.TrimSpace(info.Main.Version); module != "" && module != "(devel)" {
			return module
		}
	}
	return version
}

func BuildInfo() Info {
	info := Info{
		Version: strings.TrimSpace(Current()),
		Release: releaseVersion(),
	}
	if buildInfo, ok := readBuildInfo(); ok && buildInfo != nil {
		info.GoVersion = strings.TrimSpace(buildInfo.GoVersion)
		if buildInfo.Main.Path != "" {
			info.ModulePath = buildInfo.Main.Path
		}
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				info.Revision = strings.TrimSpace(setting.Value)
				info.ShortSHA = shortRevision(setting.Value)
			case "vcs.modified":
				info.Modified = strings.EqualFold(strings.TrimSpace(setting.Value), "true")
			}
		}
	}
	if info.ShortSHA == "" {
		info.ShortSHA = shortRevision(info.Revision)
	}
	return info
}

func vcsRevision() string {
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return shortRevision(setting.Value)
		}
	}
	return ""
}

func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if len(revision) <= 8 {
		return revision
	}
	return revision[:8]
}
