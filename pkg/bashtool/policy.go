package bashtool

import (
	"fmt"
	"strings"
)

// SecurityPolicy defines restrictions for bash command execution.
type SecurityPolicy struct {
	BlockedPatterns []string
}

// DefaultSecurityPolicy returns a sensible default set of restrictions.
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		BlockedPatterns: []string{
			"rm -rf /", "rm -rf ~", "mkfs.", "dd if=", "format ",
			":(){:|:&};:", "shutdown", "reboot", "poweroff",
		},
	}
}

// IsCommandSafe checks if a bash command is allowed by the security policy.
func (p SecurityPolicy) IsCommandSafe(command string) error {
	lowerCmd := strings.ToLower(command)
	for _, pattern := range p.BlockedPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return fmt.Errorf("blocked: command contains restricted pattern '%s'", pattern)
		}
	}
	return nil
}
