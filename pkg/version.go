package pkg

import _ "embed"
import "strings"

//go:embed VERSION.md
var versionFile []byte

var Version = strings.TrimSpace(string(versionFile))
