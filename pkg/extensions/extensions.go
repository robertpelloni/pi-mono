package extensions

// Opt-in Extension definitions mapping to shittycodingagent.ai packages.
// Disabled by default to maintain the minimal nature of the Go port.

type Extension struct {
	Name        string
	Enabled     bool
	Description string
	EnableFunc  func() error
}

var Registry = map[string]*Extension{
	"pi-better-ctx": {
		Name:        "pi-better-ctx",
		Enabled:     false,
		Description: "Alternative context/clipboard management for pi-coding-agent.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-rewind-hook": {
		Name:        "pi-rewind-hook",
		Enabled:     false,
		Description: "Rewind extension for Pi agent - automatic git checkpoints with file/conversation restore.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-plan-md": {
		Name:        "pi-plan-md",
		Enabled:     false,
		Description: "Branch-based planning workflow extension for pi.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-auto-rename": {
		Name:        "pi-auto-rename",
		Enabled:     false,
		Description: "Auto-Rename Extension for pi.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
}
