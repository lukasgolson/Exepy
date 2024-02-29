package common

import (
	"os"
	"os/exec"
)

func RunCommand(command string, args []string) error {
	return RunCommandInDir(command, args, "")
}

func RunCommandInDir(command string, args []string, dir string) error {
	cmd := exec.Command(command, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = dir

	println("Running command:", cmd.String())
	return cmd.Run()
}
