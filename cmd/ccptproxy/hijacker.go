package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateHijacker creates a shell script at {dir}/{command} that routes
// invocations through the proxy by executing ccptproxy --exec {command} "$@".
func GenerateHijacker(dir, command string) error {
	script := fmt.Sprintf("#!/bin/sh\nexec ccptproxy --exec %s \"$@\"\n", command)
	return os.WriteFile(filepath.Join(dir, command), []byte(script), 0755)
}
