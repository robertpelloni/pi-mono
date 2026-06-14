package buildinfo

import (
	"runtime"
)

// Info holds build metadata set via ldflags at compile time.
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
}

// Default returns an Info populated with static defaults.
// At compile time ldflags should inject real values
// (e.g. -ldflags="-X github.com/badlogic/pi-mono/pkg/buildinfo.Version=v0.98.0").
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Get returns the current build information.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
	}
}
