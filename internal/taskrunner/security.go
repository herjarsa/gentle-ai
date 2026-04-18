package taskrunner

import (
	"fmt"
	"strings"
)

// CommandDenylist contains shell command patterns that are blocked by default.
// Matching is case-insensitive. If any pattern is a substring of the command,
// the command is rejected unless the Executor is in dangerous mode.
//
// These patterns cover the most dangerous categories:
//   - Recursive delete (rm -rf, rm -r)
//   - Privilege escalation (sudo)
//   - Filesystem destruction (mkfs, dd, format)
//   - Pipe-to-shell exploits (curl | bash, curl | sh, wget | bash)
//   - Insecure permissions (chmod 777)
//   - Fork bombs
//   - Raw device writes
//
// Note: this denylist is defense-in-depth, not a complete sandbox.
// Sophisticated obfuscation may bypass some patterns. Use --dangerous with care.
var CommandDenylist = []string{
	"rm -rf",
	"rm -r",
	"sudo ",
	"sudo\t",
	"mkfs",
	"dd if=",
	"| bash",
	"| sh",
	"chmod 777",
	"chmod -R 777",
	":(){ :|:& };:",
	"> /dev/sd",
	"del /s /q",
	"rmdir /s /q",
	"format ",
}

// IsCommandAllowed checks whether a command is permitted under the denylist.
// Returns nil if the command is allowed. Returns an error describing the
// matched pattern if the command is blocked.
//
// Matching is case-insensitive to prevent trivial bypass via capitalization.
func IsCommandAllowed(command string) error {
	lower := strings.ToLower(command)
	for _, pattern := range CommandDenylist {
		lowerPattern := strings.ToLower(pattern)
		if strings.Contains(lower, lowerPattern) {
			return fmt.Errorf("matched blocked pattern %q", pattern)
		}
	}
	return nil
}
