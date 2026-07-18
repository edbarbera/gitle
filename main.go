// gitle — git made friendly. A thin, opinionated wrapper around the real git
// binary that gives everyday version-control tasks plain-English names.
package main

import (
	"os"
	"runtime/debug"

	"github.com/edbarbera/gitle/cmd"
)

// version is the release version. GoReleaser overrides it at build time via
// -ldflags "-X main.version=v1.2.3". Left as "dev" for local builds.
var version = "dev"

// otlpAuthHeader is the literal "Basic <token>" value of a Grafana Cloud
// access policy token, baked in by GoReleaser at release build time (see
// .goreleaser.yaml). It is empty for local `go build`, which leaves
// telemetry disabled — override locally with the GITLE_OTLP_AUTH_HEADER
// env var if you need to test the telemetry path.
var otlpAuthHeader = ""

func main() {
	header := otlpAuthHeader
	if header == "" {
		header = os.Getenv("GITLE_OTLP_AUTH_HEADER")
	}
	cmd.Execute(resolveVersion(), header)
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
