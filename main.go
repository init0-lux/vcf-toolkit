package main

import (
	"runtime/debug"

	"github.com/init0-lux/vcf-toolkit/cmd"
)

// version is set at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(buildVersion())
}

// buildVersion falls back to the module version recorded by `go install`.
func buildVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
