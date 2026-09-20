//go:build !windows

package source

import "os/exec"

func hideConsole(*exec.Cmd) {}
