// gitle — git made friendly. A thin, opinionated wrapper around the real git
// binary that gives everyday version-control tasks plain-English names.
package main

import (
	"runtime/debug"

	"github.com/edbarbera/gitle/cmd"
)

// version is the release version. GoReleaser overrides it at build time via
// -ldflags "-X main.version=v1.2.3". Left as "dev" for local builds.
var version = "dev"

func main() {
	cmd.Execute(resolveVersion())
}

// resolveVersion prefers the build-time value, then the version Go records when
// installed with `go install ...@vX`, and finally falls back to "dev".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}
