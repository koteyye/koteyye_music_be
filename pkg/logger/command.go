package logger

import (
	"os/exec"
)

// NewCommand creates a new exec.Command
// This helper function allows for easy logging of command execution in the future
func NewCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
